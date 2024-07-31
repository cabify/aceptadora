package aceptadora

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
)

const DefaultNetwork = "acceptance-testing"

type Runner struct {
	t       *testing.T
	require *require.Assertions

	puller ImagePuller

	name string
	svc  Service

	// docker stuff
	client           *client.Client
	container        container.CreateResponse
	response         types.HijackedResponse
	logsStreamDoneCh <-chan error
}

func NewRunner(t *testing.T, name string, svc Service, puller ImagePuller) *Runner {
	return &Runner{
		t:       t,
		require: require.New(t),
		name:    name,
		puller:  puller,
		svc:     svc,
	}
}

func (r *Runner) Start(ctx context.Context) {
	r.puller.Pull(ctx, r.svc.Image)

	r.createDockerClient()
	r.stopExisting(ctx)
	r.createContainer(ctx)
	r.networkConnect(ctx)
	r.attachAndStreamLogs(ctx)

	r.startContainer(ctx)
	r.t.Logf("Container %q started with ID %q", r.name, r.container.ID)
}

func (r *Runner) startContainer(ctx context.Context) {
	err := r.client.ContainerStart(ctx, r.container.ID, container.StartOptions{})
	r.require.NoError(err, "Can't start container %q for %q: %s", r.container.ID, r.name, err)
}

func (r *Runner) createDockerClient() {
	var err error
	r.client, err = client.NewClientWithOpts()
	r.require.NoError(err, "Unable to create a docker client: %v", err)
}

func (r *Runner) stopExisting(ctx context.Context) {
	listFilters := filters.NewArgs()
	listFilters.Add("name", r.name+"$")
	existing, _ := r.client.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: listFilters,
	})

	for _, c := range existing {
		r.t.Logf("Removing container %s:%s", c.Names[0], c.ID)
		if err := r.client.ContainerRemove(ctx, c.ID,
			container.RemoveOptions{
				RemoveVolumes: true,
				Force:         true,
			}); err != nil {
			r.t.Fatalf("Can't remove container %q for %q: %s", c.ID, r.name, err)
		}
	}
}

func (r *Runner) createContainer(ctx context.Context) {
	cfg := map[string]string{}
	for _, f := range r.svc.EnvFile {
		fcfg, err := loadConfigFromFile(f)
		r.require.NoError(err, "Can't load env config for %q from %q: %s", r.name, f, err)
		cfg = mergeConfigs(cfg, fcfg)
	}

	exposedPorts, portBindings, err := nat.ParsePortSpecs(r.svc.Ports)
	r.require.NoError(err, "Can't parse port specs for %q: %s", r.name, err)

	r.container, err = r.client.ContainerCreate(
		ctx,
		&container.Config{
			Image:        r.svc.Image,
			Env:          flatten(cfg),
			Cmd:          r.svc.Command,
			ExposedPorts: exposedPorts,
		},
		&container.HostConfig{
			PortBindings: portBindings,
			Binds:        r.svc.Binds,
		},
		nil,
		nil,
		r.name,
	)
	r.require.NoError(err, "Can't create container %q: %s", r.name, err)
}

func (r *Runner) networkConnect(ctx context.Context) {
	network := r.svc.Network
	if network == "" {
		network = DefaultNetwork
	}

	if _, err := r.client.NetworkInspect(ctx, network, types.NetworkInspectOptions{}); err != nil && client.IsErrNotFound(err) {
		_, err := r.client.NetworkCreate(ctx, network, types.NetworkCreate{})
		r.require.NoError(err, "Can't create network %q for container %q: %s", network, r.name, err)
	}
	err := r.client.NetworkConnect(ctx, network, r.container.ID, nil)
	r.require.NoError(err, "Can't connect %q to network %q: %s", r.name, network, err)
}

func (r *Runner) attachAndStreamLogs(ctx context.Context) {
	if r.svc.IgnoreLogs {
		return
	}
	var err error
	r.response, err = r.client.ContainerAttach(ctx, r.container.ID, container.AttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
		Logs:   true,
	})
	r.require.NoError(err, "Can't can't stream logs from %q: %s", r.name, err)
	r.logsStreamDoneCh = r.streamLogs(r.response)
}

// Stop will try to stop the container within the context provided.
func (r *Runner) Stop(ctx context.Context) error {
	return r.stop(ctx, nil)
}

// StopWithTimeout will stop the containers within the given timeout, if 0 it's just a force stop.
func (r *Runner) StopWithTimeout(ctx context.Context, timeout time.Duration) error {
	return r.stop(ctx, &timeout)
}

func (r *Runner) stop(ctx context.Context, timeout *time.Duration) error {
	if r == nil || r.client == nil {
		// nothing to stop
		return nil
	}

	timeoutSeconds := int(timeout.Seconds())
	stopOpts := container.StopOptions{Timeout: &timeoutSeconds}
	if err := r.client.ContainerStop(ctx, r.container.ID, stopOpts); err != nil {
		r.t.Errorf("Error stopping container %s: %v", r.container.ID, err)
	}

	var err error
	if r.logsStreamDoneCh != nil {
		select {
		case err = <-r.logsStreamDoneCh:
		case <-ctx.Done():
			err = ctx.Err()
		}
		if err != nil {
			r.t.Errorf("Error interrupting streaming of logs from %s: %v", r.container.ID, err)
		}

		r.response.Close()
	}
	return err
}

func (r *Runner) streamLogs(resp types.HijackedResponse) <-chan error {
	done := make(chan error)

	go func() {
		_, err := stdcopy.StdCopy(
			testLogsWriter{r.t, fmt.Sprintf("Container %q STDOUT", r.name)},
			testLogsWriter{r.t, fmt.Sprintf("Container %q STDERR", r.name)},
			resp.Reader,
		)
		done <- err
	}()
	return done
}

type testLogsWriter struct {
	t    *testing.T
	name string
}

func (tw testLogsWriter) Write(data []byte) (n int, err error) {
	s := 0
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			if i > s+1 {
				tw.t.Logf("Logs from %s: %s", tw.name, string(data[s:i]))
			}
			s = i + 1
		} else if i == len(data)-1 && i > s+1 {
			tw.t.Logf("Logs from %s: %s", tw.name, string(data[s:i]))
		}
	}
	return len(data), nil
}

func flatten(config map[string]string) []string {
	flat := make([]string, 0, len(config))
	for k, v := range config {
		flat = append(flat, fmt.Sprintf("%s=%s", k, v))
	}
	return flat
}
