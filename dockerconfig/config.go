package dockerconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/cpuguy83/dockercfg"
)

const (
	// EnvOverrideConfigDir is the name of the environment variable that can be
	// used to override the location of the client configuration files (~/.docker).
	//
	// It takes priority over the default.
	EnvOverrideConfigDir = "DOCKER_CONFIG"

	// configFileDir is the name of the directory containing the client configuration files
	configFileDir = ".docker"
)

// Load returns the docker config file. It will internally check, in this particular order:
// 1. the DOCKER_AUTH_CONFIG environment variable, unmarshalling it into a dockercfg.Config
// 2. the DOCKER_CONFIG environment variable, as the path to the config file
// 3. else it will load the default config file, which is ~/.docker/config.json
func Load() (*dockercfg.Config, error) {
	if env := os.Getenv("DOCKER_AUTH_CONFIG"); env != "" {
		var cfg dockercfg.Config
		if err := json.Unmarshal([]byte(env), &cfg); err != nil {
			return nil, fmt.Errorf("unmarshal DOCKER_AUTH_CONFIG: %w", err)
		}

		return &cfg, nil
	}

	cfg, err := dockercfg.LoadDefaultConfig()
	if err != nil {
		return nil, fmt.Errorf("load default config: %w", err)
	}

	return &cfg, nil
}

// getHomeDir returns the home directory of the current user with the help of
// environment variables depending on the target operating system.
// Returned path should be used with "path/filepath" to form new paths.
//
// On non-Windows platforms, it falls back to nss lookups, if the home
// directory cannot be obtained from environment-variables.
//
// If linking statically with cgo enabled against glibc, ensure the
// osusergo build tag is used.
//
// If needing to do nss lookups, do not disable cgo or set osusergo.
//
// getHomeDir is a copy of [pkg/homedir.Get] to prevent adding docker/docker
// as dependency for consumers that only need to read the config-file.
//
// [pkg/homedir.Get]: https://pkg.go.dev/github.com/docker/docker@v26.1.4+incompatible/pkg/homedir#Get
func getHomeDir() string {
	home, _ := os.UserHomeDir()
	if home == "" && runtime.GOOS != "windows" {
		if u, err := user.Current(); err == nil {
			return u.HomeDir
		}
	}
	return home
}

// Dir returns the directory the configuration file is stored in.
// It will check the DOCKER_CONFIG environment variable, and if not set,
// it will use the home directory.
func Dir() string {
	configDir := os.Getenv(EnvOverrideConfigDir)
	if configDir == "" {
		return filepath.Join(getHomeDir(), configFileDir)
	}

	return configDir
}
