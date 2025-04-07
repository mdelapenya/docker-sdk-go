package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

const (
	metaFile = "meta.json"

	// dockerEndpoint is the name of the docker endpoint in a stored context
	dockerEndpoint string = "docker"
)

// dockerContext is a typed representation of what we put in Context metadata
type dockerContext struct {
	Description      string
	AdditionalFields map[string]any
}

// metadataStore is a store for metadata about Docker contexts
type metadataStore struct {
	root      string
	cfg       config
	metadatas []Metadata
}

// ExtractDockerHost extracts the Docker host from the given Docker context
func ExtractDockerHost(dockerContext string, metaRoot string) (string, error) {
	ms := &metadataStore{
		root: metaRoot,
		cfg:  newConfig(),
	}

	md, err := ms.list()
	if err != nil {
		return "", fmt.Errorf("list metadata: %w", err)
	}
	ms.metadatas = md

	for _, m := range ms.metadatas {
		if m.Name == dockerContext {
			ep, ok := m.Endpoints[dockerEndpoint].(endpointMeta)
			if ok {
				return ep.Host, nil
			}
		}
	}

	return "", ErrDockerHostNotSet
}

// typeGetter is a func used to determine the concrete type of a context or
// endpoint metadata by returning a pointer to an instance of the object
// eg: for a context of type DockerContext, the corresponding typeGetter should return new(DockerContext)
type typeGetter func() any

// NamedTypeGetter is a TypeGetter associated with a name
type namedTypeGetter struct {
	name       string
	typeGetter typeGetter
}

// endpointTypeGetter returns a namedTypeGetter with the specified name and getter
func endpointTypeGetter(name string, getter typeGetter) namedTypeGetter {
	return namedTypeGetter{
		name:       name,
		typeGetter: getter,
	}
}

// endpointMeta contains fields we expect to be common for most context endpoints
type endpointMeta struct {
	Host          string `json:",omitempty"`
	SkipTLSVerify bool
}

var defaultStoreEndpoints = []namedTypeGetter{
	endpointTypeGetter(dockerEndpoint, func() any { return &endpointMeta{} }),
}

// config is used to configure the metadata marshaler of the context ContextStore
type config struct {
	contextType   typeGetter
	endpointTypes map[string]typeGetter
}

// newConfig creates a config object
func newConfig() config {
	res := config{
		contextType:   func() any { return &dockerContext{} },
		endpointTypes: make(map[string]typeGetter),
	}

	for _, e := range defaultStoreEndpoints {
		res.endpointTypes[e.name] = e.typeGetter
	}

	return res
}

// Metadata contains Metadata about a context and its endpoints
type Metadata struct {
	Name      string         `json:",omitempty"`
	Metadata  any            `json:",omitempty"`
	Endpoints map[string]any `json:",omitempty"`
}

func (s *metadataStore) contextDir(id contextdir) string {
	return filepath.Join(s.root, string(id))
}

type untypedContextMetadata struct {
	Metadata  json.RawMessage            `json:"metadata,omitempty"`
	Endpoints map[string]json.RawMessage `json:"endpoints,omitempty"`
	Name      string                     `json:"name,omitempty"`
}

// getByID returns the metadata for a context by its ID
func (s *metadataStore) getByID(id contextdir) (Metadata, error) {
	fileName := filepath.Join(s.contextDir(id), metaFile)
	bytes, err := os.ReadFile(fileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Metadata{}, fmt.Errorf("context file not found: %w", err)
		}
		return Metadata{}, fmt.Errorf("read file: %w", err)
	}

	var untyped untypedContextMetadata
	r := Metadata{
		Endpoints: make(map[string]any),
	}

	if err := json.Unmarshal(bytes, &untyped); err != nil {
		return Metadata{}, fmt.Errorf("parsing %s (metadata): %w", fileName, err)
	}

	r.Name = untyped.Name
	if r.Metadata, err = parseTypedOrMap(untyped.Metadata, s.cfg.contextType); err != nil {
		return Metadata{}, fmt.Errorf("parsing %s (context type): %w", fileName, err)
	}

	for k, v := range untyped.Endpoints {
		if r.Endpoints[k], err = parseTypedOrMap(v, s.cfg.endpointTypes[k]); err != nil {
			return Metadata{}, fmt.Errorf("parsing %s (endpoint types): %w", fileName, err)
		}
	}

	return r, err
}

// list returns a list of all Docker contexts
func (s *metadataStore) list() ([]Metadata, error) {
	ctxDirs, err := listRecursivelyMetadataDirs(s.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	res := make([]Metadata, 0, len(ctxDirs))
	for _, dir := range ctxDirs {
		c, err := s.getByID(contextdir(dir))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("read metadata: %w", err)
		}
		res = append(res, c)
	}

	return res, nil
}

// contextdir is a type used to represent a context directory
type contextdir string

// isContextDir checks if the given path is a context directory,
// which means it contains a meta.json file.
func isContextDir(path string) bool {
	s, err := os.Stat(filepath.Join(path, metaFile))
	if err != nil {
		return false
	}

	return !s.IsDir()
}

// listRecursivelyMetadataDirs lists all directories that contain a meta.json file
func listRecursivelyMetadataDirs(root string) ([]string, error) {
	fileEntries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, fileEntry := range fileEntries {
		if fileEntry.IsDir() {
			if isContextDir(filepath.Join(root, fileEntry.Name())) {
				result = append(result, fileEntry.Name())
			}

			subs, err := listRecursivelyMetadataDirs(filepath.Join(root, fileEntry.Name()))
			if err != nil {
				return nil, err
			}

			for _, s := range subs {
				result = append(result, filepath.Join(fileEntry.Name(), s))
			}
		}
	}

	return result, nil
}

// parseTypedOrMap parses a JSON payload into a typed object or a map
func parseTypedOrMap(payload []byte, getter typeGetter) (any, error) {
	if len(payload) == 0 || string(payload) == "null" {
		return nil, nil
	}

	if getter == nil {
		var res map[string]any
		if err := json.Unmarshal(payload, &res); err != nil {
			return nil, err
		}
		return res, nil
	}

	typed := getter()
	if err := json.Unmarshal(payload, typed); err != nil {
		return nil, err
	}

	return reflect.ValueOf(typed).Elem().Interface(), nil
}
