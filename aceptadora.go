package aceptadora

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Config is intended to be loaded by "github.com/colega/envconfig"
type Config struct {
	YAMLDir  string `default:"./"`
	YAMLName string `default:"aceptadora.yml"`
}

type Aceptadora struct {
	cfg     Config
	t       *testing.T
	require *require.Assertions

	yaml YAML

	imagePuller ImagePuller

	services map[string]*Runner
	order    []string
}

// New creates a new Aceptadora. It will try to load the YAML config from the path provided by Config
// If something goes wrong, it will use testing.T to fail.
func New(t *testing.T, imagePuller ImagePuller, cfg Config) *Aceptadora {
	if _, ok := os.LookupEnv("TESTER_ADDRESS"); !ok {
		os.Setenv("TESTER_ADDRESS", getLocalIP())
	}

	os.Setenv("YAMLDIR", cfg.YAMLDir)
	yamlPath := cfg.YAMLDir + "/" + cfg.YAMLName
	yaml, err := LoadYAML(yamlPath)
	require.NoError(t, err, "Can't load YAML from %q: %s", yamlPath, err)

	return &Aceptadora{
		t:           t,
		require:     require.New(t),
		cfg:         cfg,
		yaml:        yaml,
		imagePuller: imagePuller,
		services:    map[string]*Runner{},
	}
}

// PullImages pulls all the images mentioned in aceptadora.yml
// This allows doing this outside of the context of the test, and avoid unrelated flaky timeouts in the tests
// happening when most of the context has been consumed by pulling the image
func (a *Aceptadora) PullImages(ctx context.Context) {
	for svcName, svc := range a.yaml.Services {
		t0 := time.Now()
		a.imagePuller.Pull(ctx, svc.Image)
		a.t.Logf("Pulled image %q for %q in %s", svc.Image, svcName, time.Since(t0))
	}
}

// Run will start a given service (from aceptadora.yml) and register it for stopping later
func (a *Aceptadora) Run(ctx context.Context, name string) {
	if _, ok := a.yaml.Services[name]; !ok {
		a.t.Fatalf("There's no service with name %q", name)
	}
	if _, ok := a.services[name]; ok {
		a.t.Fatalf("Trying to start again the service %q", name)
	}

	runner := NewRunner(a.t, name, a.yaml.Services[name], a.imagePuller)
	runner.Start(ctx)
	a.services[name] = runner
	a.order = append(a.order, name)
}

// StopAll will stop all the services in the reverse order
// If you need to explicitly stop some service in first place, use Stop() previously.
func (a *Aceptadora) StopAll(ctx context.Context) {
	for i := len(a.order) - 1; i >= 0; i-- {
		a.Stop(ctx, a.order[i])
	}
}

// Stop will try to stop the service with the name provided
// It will fail fatally if such service isn't defined
// It will skip the service if it's already stopped, and set it to nil once stopped, making this call idempotent
// Stop is not thread safe.
func (a *Aceptadora) Stop(ctx context.Context, name string) {
	svc, ok := a.services[name]
	if !ok {
		a.t.Fatalf("There's no service %q to stop", name)
	}
	if svc == nil {
		return
	}

	err := svc.Stop(ctx)
	assert.NoError(a.t, err, "Can't stop service %q in time: %s", err)
	a.services[name] = nil
}

// getLocalIP returns the non loopback local IP of the host
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
