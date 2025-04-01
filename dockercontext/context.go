package dockercontext

// The code in this file has been extracted from https://github.com/docker/cli,
// more especifically from https://github.com/docker/cli/blob/master/cli/context/store/metadatastore.go
// with the goal of not consuming the CLI package and all its dependencies.

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/docker-sdk-go/dockerconfig"
	"github.com/docker/docker-sdk-go/dockercontext/internal"
)

const (
	// DefaultContextName is the name reserved for the default context (config & env based)
	DefaultContextName = "default"

	// EnvOverrideContext is the name of the environment variable that can be
	// used to override the context to use. If set, it overrides the context
	// that's set in the CLI's configuration file, but takes no effect if the
	// "DOCKER_HOST" env-var is set (which takes precedence.
	EnvOverrideContext = "DOCKER_CONTEXT"

	// EnvOverrideHost is the name of the environment variable that can be used
	// to override the default host to connect to (DefaultDockerHost).
	//
	// This env-var is read by FromEnv and WithHostFromEnv and when set to a
	// non-empty value, takes precedence over the default host (which is platform
	// specific), or any host already set.
	EnvOverrideHost = "DOCKER_HOST"

	// contextsDir is the name of the directory containing the contexts
	contextsDir = "contexts"

	// metadataDir is the name of the directory containing the metadata
	metadataDir = "meta"
)

// ErrDockerHostNotSet is the error returned when the Docker host is not set in the Docker context
var ErrDockerHostNotSet = internal.ErrDockerHostNotSet

// Current returns the current context name, based on
// environment variables and the cli configuration file. It does not
// validate if the given context exists or if it's valid; errors may
// occur when trying to use it.
func Current() string {
	cfg, err := dockerconfig.Load()
	if err != nil {
		return DefaultContextName
	}

	if os.Getenv(EnvOverrideHost) != "" {
		return DefaultContextName
	}

	if ctxName := os.Getenv(EnvOverrideContext); ctxName != "" {
		return ctxName
	}

	if cfg.CurrentContext != "" {
		// We don't validate if this context exists: errors may occur when trying to use it.
		return cfg.CurrentContext
	}

	return DefaultContextName
}

// CurrentDockerHost returns the Docker host from the current Docker context.
// For that, it traverses the directory structure of the Docker configuration directory,
// looking for the current context and its Docker endpoint.
func CurrentDockerHost() (string, error) {
	metaRoot, err := metaRoot()
	if err != nil {
		return "", fmt.Errorf("meta root: %w", err)
	}

	return internal.ExtractDockerHost(Current(), metaRoot)
}

// metaRoot returns the root directory of the Docker context metadata.
func metaRoot() (string, error) {
	dir, err := dockerconfig.Dir()
	if err != nil {
		return "", fmt.Errorf("docker config dir: %w", err)
	}

	return filepath.Join(filepath.Join(dir, contextsDir), metadataDir), nil
}
