package suite

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/cabify/aceptadora"
	"github.com/colega/envconfig"
	"github.com/stretchr/testify/suite"
)

const expectedMockedDependencyInventedHTTPStatusCode = 288

type Config struct {
	Aceptadora  aceptadora.Config
	ImagePuller aceptadora.ImagePullerConfig

	// ServicesAddress is the address where services started by aceptadora can be found
	// It differs from env to env, and it's set up in env-specific configs
	ServicesAddress string
}

type acceptanceSuite struct {
	suite.Suite

	cfg        Config
	aceptadora *aceptadora.Aceptadora

	mockedDependencyListener net.Listener
}

func (s *acceptanceSuite) SetupSuite() {
	aceptadora.SetEnv(
		s.T(),
		aceptadora.OneOfEnvConfigs(
			aceptadora.EnvConfigWhenEnvVarPresent("../config/gitlab.env", "GITLAB_CI"),
			aceptadora.EnvConfigAlways("../config/default.env"),
		),
		aceptadora.EnvConfigAlways("acceptance.env"),
	)
	s.Require().NoError(envconfig.Process("ACCEPTANCE", &s.cfg))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	imagePuller := aceptadora.NewImagePuller(s.T(), s.cfg.ImagePuller)
	s.aceptadora = aceptadora.New(s.T(), imagePuller, s.cfg.Aceptadora)
	s.aceptadora.PullImages(ctx)

	s.startMockedProxyDependency()

	s.aceptadora.Run(ctx, "redis")
	s.Require().Eventually(func() bool {
		return tcpConnectionIsAccepted(s.cfg.ServicesAddress, 6379)
	}, time.Minute, 50*time.Millisecond, "redis didn't start")

	s.aceptadora.Run(ctx, "proxy")
	s.Require().Eventually(func() bool {
		return httpHealthcheckSucceeds(s.cfg.ServicesAddress, 8888)
	}, time.Minute, 50*time.Millisecond, "proxy didn't start")
}

func (s *acceptanceSuite) TestProxyCall() {
	// we call the proxy on some path, and proxy will call us, so we should see the same status code
	resp, err := http.DefaultClient.Get(fmt.Sprintf("http://%s:8888/some/random/path", s.cfg.ServicesAddress))
	s.Require().NoError(err)
	s.Require().Equal(expectedMockedDependencyInventedHTTPStatusCode, resp.StatusCode)
}

func (s *acceptanceSuite) TestProxyCallAfterRestart() {
	startTimeBefore, err := s.getStartTime()
	s.Require().NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.aceptadora.Restart(ctx, "proxy")

	var startTimeAfter int64
	s.Require().Eventually(func() bool {
		startTimeAfter, err = s.getStartTime()
		return err == nil
	}, time.Minute, 50*time.Millisecond, "could not get start time after restart")
	s.Require().Greater(startTimeAfter, startTimeBefore, "proxy didn't restart")
}

func (s *acceptanceSuite) TearDownSuite() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	s.aceptadora.StopAll(ctx)
	if s.mockedDependencyListener != nil {
		s.mockedDependencyListener.Close()
	}
}

func (s *acceptanceSuite) startMockedProxyDependency() {
	var err error
	// we listen on the port we've provided to the proxy
	s.mockedDependencyListener, err = net.Listen("tcp", ":8000")
	s.Require().NoError(err)

	go func() {
		handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(expectedMockedDependencyInventedHTTPStatusCode)
		})
		_ = http.Serve(s.mockedDependencyListener, handler)
	}()
}

func (s *acceptanceSuite) getStartTime() (int64, error) {
	url := fmt.Sprintf("http://%s:8888/status", s.cfg.ServicesAddress)
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("status endpoint returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	statusResponse := struct {
		StartedAt int64 `json:"started_at"`
	}{}
	if err := json.Unmarshal(body, &statusResponse); err != nil {
		return 0, err
	}
	return statusResponse.StartedAt, nil
}

func TestAcceptanceSuite(t *testing.T) {
	suite.Run(t, new(acceptanceSuite))
}

func tcpConnectionIsAccepted(host string, port int) bool {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// httpHealthcheckSucceeds will return true if the /status endpoint on a given host/port returns a 200 status code to a GET request
func httpHealthcheckSucceeds(host string, port int) bool {
	url := fmt.Sprintf("http://%s:%d/status", host, port)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		panic(fmt.Errorf("can't build http request for healthcheck, maybe wrong config? %s", err))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
