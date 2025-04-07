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

func TestStore_load(t *testing.T) {
	t.Run("success", func(t *testing.T) {
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

func TestStore_list(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		// Setup test contexts
		contexts := map[string]metadata{
			"context1": {
				Name: "context1",
				Endpoints: map[string]*endpoint{
					"docker": {Host: "tcp://1.2.3.4:2375"},
				},
			},
			"nested/context2": {
				Name: "context2",
				Endpoints: map[string]*endpoint{
					"docker": {Host: "unix:///var/run/docker.sock"},
				},
			},
		}

		for path, meta := range contexts {
			setupTestContext(t, tmpDir, path, meta)
		}

		list, err := s.list()
		require.NoError(t, err)
		require.Len(t, list, 2)
	})

	t.Run("root-does-not-exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonExistentDir := filepath.Join(tmpDir, "does-not-exist")
		s := &store{root: nonExistentDir}

		list, err := s.list()
		require.NoError(t, err) // Should return empty list, not error
		require.Empty(t, list)
	})

	t.Run("corrupted-metadata-file", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		// Create a context directory with invalid JSON
		contextDir := filepath.Join(tmpDir, "invalid")
		require.NoError(t, os.MkdirAll(contextDir, 0755))
		require.NoError(t, os.WriteFile(
			filepath.Join(contextDir, metaFile),
			[]byte("invalid json"),
			0644,
		))

		_, err := s.list()
		require.Error(t, err)
		require.Contains(t, err.Error(), "parse metadata")
	})

	t.Run("mixed-valid-and-invalid-contexts", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		// Setup one valid context
		validMeta := metadata{
			Name: "valid",
			Endpoints: map[string]*endpoint{
				"docker": {Host: "tcp://1.2.3.4:2375"},
			},
		}
		setupTestContext(t, tmpDir, "valid", validMeta)

		// Setup an invalid context
		invalidDir := filepath.Join(tmpDir, "invalid")
		require.NoError(t, os.MkdirAll(invalidDir, 0755))
		require.NoError(t, os.WriteFile(
			filepath.Join(invalidDir, metaFile),
			[]byte("invalid json"),
			0644,
		))

		_, err := s.list()
		require.Error(t, err)
		require.Contains(t, err.Error(), "parse metadata")
	})

	t.Run("permission-denied", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("cannot test permission denied as root")
		}

		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		// Create a context with no read permissions
		contextDir := filepath.Join(tmpDir, "no-access")
		require.NoError(t, os.MkdirAll(contextDir, 0755))

		meta := metadata{
			Name: "test",
			Endpoints: map[string]*endpoint{
				"docker": {Host: "tcp://1.2.3.4:2375"},
			},
		}
		setupTestContext(t, tmpDir, "no-access", meta)

		// Remove read permissions
		require.NoError(t, os.Chmod(filepath.Join(contextDir, metaFile), 0000))
		defer os.Chmod(filepath.Join(contextDir, metaFile), 0644) // Restore permissions for cleanup

		_, err := s.list()
		require.Error(t, err)
		require.Contains(t, err.Error(), "permission denied")
	})

	t.Run("empty-but-valid-context-file", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		// Create a context with empty but valid JSON
		contextDir := filepath.Join(tmpDir, "empty")
		require.NoError(t, os.MkdirAll(contextDir, 0755))
		require.NoError(t, os.WriteFile(
			filepath.Join(contextDir, metaFile),
			[]byte("{}"),
			0644,
		))

		list, err := s.list()
		require.NoError(t, err)
		require.Len(t, list, 1)
		require.Empty(t, list[0].Name)
		require.Empty(t, list[0].Endpoints)
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
