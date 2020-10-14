package aceptadora

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// SetEnv loads the configuration from a file, choosing the proper one depending on the provided matchers.
// Every matcher returning true as second value will be executed.
// Matchers are executed in their provided order
func SetEnv(t *testing.T, pathForEnv ...func() (string, bool)) {
	for _, f := range pathForEnv {
		if path, shouldBeUsed := f(); shouldBeUsed {
			env, err := loadConfigFromFile(path)
			require.NoError(t, err, "Can't load config from %s: %s", path, err)
			for k, v := range env {
				os.Setenv(k, v)
			}
			t.Logf("Loaded env from %q", path)
		}
	}
}

// OneOfEnvConfigs provides a matcher that returns as the result the result of the first matching matcher.
// Useful when several matchers would match, like for instance a Github Actions environment is being a Linux at the same time.
func OneOfEnvConfigs(matchers ...func() (string, bool)) func() (string, bool) {
	return func() (string, bool) {
		for _, match := range matchers {
			if path, matches := match(); matches {
				return path, true
			}
		}
		return "", false
	}
}

// EnvConfigPathForDarwin is a matcher for SetEnv that matches Mac environments, we use the home directory prefix instead
// of OSTYPE as it isn't defined when running tests from Goland
func EnvConfigPathForDarwin(path string) func() (string, bool) {
	return func() (string, bool) {
		isMac := strings.HasPrefix(os.Getenv("HOME"), "/Users/")
		return path, isMac
	}
}

// EnvConfigWhenEnvVarPresent is a matcher for SetEnv that matches when the provided env var name is present
func EnvConfigWhenEnvVarPresent(path, envVarName string) func() (string, bool) {
	return func() (string, bool) {
		_, isGitlab := os.LookupEnv(envVarName)
		return path, isGitlab
	}
}

// EnvConfigCommon is a matcher for SetEnv that is always true, useful to load the common acceptance.env config once
// the env-specifics are loaded.
func EnvConfigCommon(path string) func() (string, bool) {
	return func() (string, bool) {
		return path, true
	}
}

// loadConfigFromFile parses the configuration file into a map, expanding the env var references to their values.
func loadConfigFromFile(filepath string) (map[string]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = os.ExpandEnv(line)
		if len(line) > 0 && line[0] != '#' {
			s := strings.SplitN(line, "=", 2)
			config[s[0]] = s[1]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}
	return config, nil
}

// mergeConfigs merges, overwriting values, the provided configs into a single one.
func mergeConfigs(cfgs ...map[string]string) map[string]string {
	cfg := map[string]string{}
	for _, another := range cfgs {
		for k, v := range another {
			cfg[k] = v
		}
	}
	return cfg
}
