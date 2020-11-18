package aceptadora

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
)

// ImagePullerConfig configures the pulling options for different image repositories
type ImagePullerConfig struct {
	Repo []RepositoryConfig
}

// RepositoryConfig provides the details of access to a docker repository.
type RepositoryConfig struct {
	// Domain is used to specify which domain this config applies to, like `docker.io`
	Domain string
	// SkipPulling can be specified if images from this domain are not intended to be pulled
	// Useful for images previously built locally, or for local testing when repository credentials are not passed to the test
	SkipPulling bool

	// Auth provides the default docker library's field to authenticate and will be used for pulling.
	// Usually Username & Password fields should be filled.
	Auth types.AuthConfig
}

func NewImagePuller(t *testing.T, cfg ImagePullerConfig) *ImagePullerImpl {
	repos := make(map[string]RepositoryConfig, len(cfg.Repo))
	for _, repo := range cfg.Repo {
		repos[repo.Domain] = repo
	}
	return &ImagePullerImpl{
		t:       t,
		require: require.New(t),
		cfg:     cfg,
		repos:   repos,
	}
}

type ImagePuller interface {
	Pull(ctx context.Context, imageName string)
}

type ImagePullerImpl struct {
	t       *testing.T
	require *require.Assertions

	images sync.Map
	cfg    ImagePullerConfig
	repos  map[string]RepositoryConfig
}

func (i *ImagePullerImpl) Pull(ctx context.Context, imageName string) {
	imi, _ := i.images.LoadOrStore(imageName, &image{})
	im := imi.(*image)

	im.Do(func() {
		im.err = i.tryPullImage(ctx, imageName)
	})
	i.require.NoError(im.err, "Can't pull image %q: %s", imageName, im.err)
}

type image struct {
	sync.Once
	err error
}

func (i *ImagePullerImpl) tryPullImage(ctx context.Context, imageName string) error {
	ref, err := reference.ParseNamed(imageName)
	if err != nil {
		return fmt.Errorf("can't parse image name %q: %w", imageName, err)
	}
	domain := reference.Domain(ref)

	repoCfg, ok := i.repos[domain]
	if repoCfg.SkipPulling {
		i.t.Logf("Not pulling %s: disabled by config for domain %s", imageName, domain)
		return nil
	}

	var authStr string
	if ok {
		encodedJSON, err := json.Marshal(repoCfg.Auth)
		if err != nil {
			return fmt.Errorf("encoding JSON auth: %v", err)
		}
		authStr = base64.URLEncoding.EncodeToString(encodedJSON)
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		return fmt.Errorf("creating docker client: %v", err)
	}

	out, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{RegistryAuth: authStr})
	if err != nil {
		return fmt.Errorf("can't pull image %s: %v", imageName, err)
	}
	defer out.Close()

	_, _ = io.Copy(
		testLogsWriter{i.t, fmt.Sprintf("Image %q puller", imageName)},
		out,
	)

	return nil
}
