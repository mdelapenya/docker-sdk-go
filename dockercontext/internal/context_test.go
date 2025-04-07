package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractDockerHost(t *testing.T) {
	t.Run("context-found-with-host", func(t *testing.T) {
		host := requireDockerHost(t, "test-context", metadata{
			Name: "test-context",
			Endpoints: map[string]*endpoint{
				"docker": {Host: "tcp://1.2.3.4:2375"},
			},
		})
		require.Equal(t, "tcp://1.2.3.4:2375", host)
	})

	t.Run("context-found-without-host", func(t *testing.T) {
		requireDockerHostError(t, "test-context", metadata{
			Name: "test-context",
			Endpoints: map[string]*endpoint{
				"docker": {},
			},
		}, ErrDockerHostNotSet)
	})

	t.Run("context-not-found", func(t *testing.T) {
		requireDockerHostError(t, "missing", metadata{
			Name: "other-context",
			Endpoints: map[string]*endpoint{
				"docker": {Host: "tcp://1.2.3.4:2375"},
			},
		}, ErrDockerHostNotSet)
	})

	t.Run("nested-context-found", func(t *testing.T) {
		host := requireDockerHostInPath(t, "nested-context", "parent/nested-context", metadata{
			Name: "nested-context",
			Endpoints: map[string]*endpoint{
				"docker": {Host: "tcp://1.2.3.4:2375"},
			},
		})
		require.Equal(t, "tcp://1.2.3.4:2375", host)
	})
}

func TestStore(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		// Setup test contexts with paths as keys
		contexts := map[string]metadata{
			"context1": {
				Name: "context1",
				Context: &dockerContext{
					Description: "test context 1",
					Fields:      map[string]any{"foo": "bar"},
				},
				Endpoints: map[string]*endpoint{
					"docker": {Host: "tcp://1.2.3.4:2375"},
				},
			},
			"nested/context2": {
				Name: "context2",
				Context: &dockerContext{
					Description: "test context 2",
				},
				Endpoints: map[string]*endpoint{
					"docker": {Host: "unix:///var/run/docker.sock"},
				},
			},
		}

		// Create all test contexts
		for path, meta := range contexts {
			setupTestContext(t, tmpDir, path, meta)
		}

		// Test listing
		list, err := s.list()
		require.NoError(t, err)
		require.Len(t, list, len(contexts))

		// Create a map of found contexts by their path for verification
		found := make(map[string]*metadata)
		for _, ctx := range list {
			// Find the context directory relative to root
			contextDir := ""
			for path := range contexts {
				fullPath := filepath.Join(tmpDir, path)
				if hasMetaFile(fullPath) {
					meta := contexts[path]
					if meta.Name == ctx.Name {
						contextDir = path
						break
					}
				}
			}
			require.NotEmpty(t, contextDir, "could not find path for context %s", ctx.Name)
			found[contextDir] = ctx
		}

		// Verify each context was loaded correctly
		for path, want := range contexts {
			got, exists := found[path]
			require.True(t, exists, "context not found at path: %s", path)
			require.Equal(t, want.Name, got.Name)
			require.Equal(t, want.Context.Description, got.Context.Description)
			require.Equal(t, want.Endpoints["docker"].Host, got.Endpoints["docker"].Host)
		}
	})

	t.Run("load", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		want := metadata{
			Name: "test",
			Context: &dockerContext{
				Description: "test context",
				Fields:      map[string]any{"test": true},
			},
			Endpoints: map[string]*endpoint{
				"docker": {
					Host:          "tcp://localhost:2375",
					SkipTLSVerify: true,
				},
			},
		}

		contextDir := filepath.Join(tmpDir, "test")
		setupTestContext(t, tmpDir, "test", want)

		got, err := s.load(contextDir)
		require.NoError(t, err)
		require.Equal(t, want.Name, got.Name)
		require.Equal(t, want.Context.Description, got.Context.Description)
		require.Equal(t, want.Context.Fields, got.Context.Fields)
		require.Equal(t, want.Endpoints["docker"].Host, got.Endpoints["docker"].Host)
		require.Equal(t, want.Endpoints["docker"].SkipTLSVerify, got.Endpoints["docker"].SkipTLSVerify)
	})
}

// requireDockerHost creates a context and verifies host extraction succeeds
func requireDockerHost(t *testing.T, contextName string, meta metadata) string {
	t.Helper()
	tmpDir := t.TempDir()

	setupTestContext(t, tmpDir, contextName, meta)

	host, err := ExtractDockerHost(contextName, tmpDir)
	require.NoError(t, err)
	return host
}

// requireDockerHostInPath creates a context at a specific path and verifies host extraction
func requireDockerHostInPath(t *testing.T, contextName, path string, meta metadata) string {
	t.Helper()
	tmpDir := t.TempDir()

	setupTestContext(t, tmpDir, path, meta)

	host, err := ExtractDockerHost(contextName, tmpDir)
	require.NoError(t, err)
	return host
}

// requireDockerHostError creates a context and verifies expected error
func requireDockerHostError(t *testing.T, contextName string, meta metadata, wantErr error) {
	t.Helper()
	tmpDir := t.TempDir()

	setupTestContext(t, tmpDir, contextName, meta)

	_, err := ExtractDockerHost(contextName, tmpDir)
	require.ErrorIs(t, err, wantErr)
}

// setupTestContext creates a test context file in the specified location
func setupTestContext(t *testing.T, root, relPath string, meta metadata) {
	t.Helper()

	contextDir := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(contextDir, 0755))

	data, err := json.Marshal(meta)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(
		filepath.Join(contextDir, metaFile),
		data,
		0644,
	))
}
