package dockerconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

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

// UserHomeConfigPath returns the path to the docker config in the current user's home dir.
func UserHomeConfigPath() (string, error) {
	home := getHomeDir()
	if home == "" {
		return "", fmt.Errorf("could not determine user home directory")
	}
	return filepath.Join(home, ".docker", FileName), nil
}

// ConfigPath returns the path to the docker cli config.
//
// It will either use the DOCKER_CONFIG env var if set, or the value from [UserHomeConfigPath]
// DOCKER_CONFIG would be the dir path where `config.json` is stored, this returns the path to config.json.
func ConfigPath() (string, error) {
	if p := os.Getenv(EnvOverrideDir); p != "" {
		return filepath.Join(p, FileName), nil
	}
	return UserHomeConfigPath()
}

// Dir returns the directory the configuration file is stored in.
// It will check the DOCKER_CONFIG environment variable, and if not set,
// it will use the home directory.
func Dir() string {
	configDir := os.Getenv(EnvOverrideDir)
	if configDir == "" {
		return filepath.Join(getHomeDir(), configFileDir)
	}

	return configDir
}

// Load returns the docker config file. It will internally check, in this particular order:
// 1. the DOCKER_AUTH_CONFIG environment variable, unmarshalling it into a Config
// 2. the DOCKER_CONFIG environment variable, as the path to the config file
// 3. else it will load the default config file, which is ~/.docker/config.json
func Load() (Config, error) {
	if env := os.Getenv("DOCKER_AUTH_CONFIG"); env != "" {
		var cfg Config
		if err := json.Unmarshal([]byte(env), &cfg); err != nil {
			return Config{}, fmt.Errorf("unmarshal DOCKER_AUTH_CONFIG: %w", err)
		}

		return cfg, nil
	}

	return LoadDefaultConfig()
}

// LoadDefaultConfig loads the docker cli config from the path returned from [ConfigPath].
func LoadDefaultConfig() (Config, error) {
	var cfg Config
	p, err := ConfigPath()
	if err != nil {
		return cfg, fmt.Errorf("config path: %w", err)
	}

	return cfg, FromFile(p, &cfg)
}

// FromFile loads config from the specified path into cfg.
func FromFile(configPath string, cfg *Config) error {
	f, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(&cfg); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}

	return nil
}
