package dmp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
)

type DMPResource struct {
	ID          int            `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Type        string         `json:"type,omitempty"`
	Schema      string         `json:"schema,omitempty"`
	Connection  map[string]any `json:"connection,omitempty"`
	Source      map[string]any `json:"source,omitempty"`
	Destination map[string]any `json:"destination,omitempty"`
	Transforms  []any          `json:"transforms,omitempty"`
	Vaults      []any          `json:"vaults,omitempty"`
	Config      map[string]any `json:"config,omitempty"`
	LogLevel    string         `json:"logLevel,omitempty"`
	LogLevels   map[string]any `json:"logLevels,omitempty"`
	Meta        *Meta          `json:"_meta,omitempty"`
}

type ResolvedResource struct {
	Key  string
	ID   int
	Name string
	Type string
}

type ResourceResolver struct {
	Client *HTTPClient
}

type CommonDestinationSpec struct {
	Key           string
	ConnectionKey string
}

type ScenarioSourceSpec struct {
	Key           string
	File          string
	ConnectionKey string
}

type CommonTransformSpec struct {
	Key string
}

type ScenarioPipelineSpec struct {
	Key            string
	File           string
	SourceKey      string
	DestinationKey string
}

func NewResourceResolver(client *HTTPClient) *ResourceResolver {
	return &ResourceResolver{
		Client: client,
	}
}

func (r *ResourceResolver) EnsureScenarioSources(
	rootDir string,
	scenarioName string,
	specs []ScenarioSourceSpec,
	connections map[string]ResolvedResource,
) (map[string]ResolvedResource, error) {
	resolved := make(map[string]ResolvedResource, len(specs))

	for _, spec := range specs {
		connection, ok := connections[spec.ConnectionKey]
		if !ok {
			return nil, fmt.Errorf("connection %q was not resolved for source %q", spec.ConnectionKey, spec.Key)
		}

		resource, payload, err := loadScenarioPayload(rootDir, scenarioName, spec.File)
		if err != nil {
			return nil, err
		}

		resource.Connection = map[string]any{
			"id":   connection.ID,
			"name": connection.Name,
		}

		resource.Meta = nil

		payload, err = json.Marshal(resource)
		if err != nil {
			return nil, fmt.Errorf("encode scenario DMP source payload %q: %w", resource.Name, err)
		}

		source, err := r.ensureResource(SourceType, spec.Key, resource.Name, payload, nil)
		if err != nil {
			return nil, err
		}

		resolved[spec.Key] = source
	}

	printResolvedResources("Resolved DMP scenario sources:", resolved)

	return resolved, nil
}

func (r *ResourceResolver) EnsureScenarioPipelines(
	rootDir string,
	scenarioName string,
	specs []ScenarioPipelineSpec,
	sources map[string]ResolvedResource,
	destinations map[string]ResolvedResource,
) (map[string]ResolvedResource, error) {
	resolved := make(map[string]ResolvedResource, len(specs))

	for _, spec := range specs {
		source, ok := sources[spec.SourceKey]
		if !ok {
			return nil, fmt.Errorf("source %q was not resolved for pipeline %q", spec.SourceKey, spec.Key)
		}

		destination, ok := destinations[spec.DestinationKey]
		if !ok {
			return nil, fmt.Errorf("destination %q was not resolved for pipeline %q", spec.DestinationKey, spec.Key)
		}

		resource, payload, err := loadScenarioPayload(rootDir, scenarioName, spec.File)
		if err != nil {
			return nil, err
		}

		resource.Source = map[string]any{
			"id":   source.ID,
			"name": source.Name,
		}
		resource.Destination = map[string]any{
			"id":   destination.ID,
			"name": destination.Name,
		}

		payload, err = json.Marshal(resource)
		if err != nil {
			return nil, fmt.Errorf("encode scenario DMP pipeline payload %q: %w", resource.Name, err)
		}

		pipeline, err := r.ensureResource(PipelineType, spec.Key, resource.Name, payload, nil)
		if err != nil {
			return nil, err
		}

		resolved[spec.Key] = pipeline
	}

	printResolvedResources("Resolved DMP scenario pipelines:", resolved)

	return resolved, nil
}

func loadScenarioPayload(rootDir string, scenarioName string, file string) (DMPResource, []byte, error) {
	path := filepath.Join(rootDir, scenarioName, file)

	raw, err := os.ReadFile(path)
	if err != nil {
		return DMPResource{}, nil, fmt.Errorf("read scenario DMP payload %q: %w", path, err)
	}

	expanded := []byte(os.ExpandEnv(string(raw)))

	var payload DMPResource
	if err := json.Unmarshal(expanded, &payload); err != nil {
		return DMPResource{}, nil, fmt.Errorf("decode scenario DMP payload %q after env expansion: %w", path, err)
	}

	if payload.Name == "" {
		return DMPResource{}, nil, fmt.Errorf("scenario DMP payload %q is missing name", path)
	}

	return payload, expanded, nil
}

func (r *ResourceResolver) EnsureCommonConnections(rootDir string, keys []string) (map[string]ResolvedResource, error) {
	resolved := make(map[string]ResolvedResource, len(keys))

	for _, key := range keys {
		resource, _, err := r.ensureCommonResource(rootDir, ConnectionType, "connections", key, func(payload []byte, resource DMPResource) error {
			if !resource.ShouldValidateConnection() {
				fmt.Printf("DMP connection %q validation skipped\n", resource.Name)

				return nil
			}

			return r.ValidateConnection(payload, resource.Name)
		})
		if err != nil {
			return nil, err
		}

		resolved[key] = resource
	}

	printResolvedResources("Resolved DMP connections:", resolved)

	return resolved, nil
}

func (r *ResourceResolver) EnsureCommonDestinations(
	rootDir string,
	specs []CommonDestinationSpec,
	connections map[string]ResolvedResource,
) (map[string]ResolvedResource, error) {
	resolved := make(map[string]ResolvedResource, len(specs))

	for _, spec := range specs {
		connection, ok := connections[spec.ConnectionKey]
		if !ok {
			return nil, fmt.Errorf("connection %q was not resolved for destination %q", spec.ConnectionKey, spec.Key)
		}

		resource, payload, err := loadCommonPayload(rootDir, "destinations", spec.Key)
		if err != nil {
			return nil, err
		}

		resource.Connection = map[string]any{
			"id":   connection.ID,
			"name": connection.Name,
		}

		resource.Meta = nil

		payload, err = json.Marshal(resource)
		if err != nil {
			return nil, fmt.Errorf("encode common DMP destination payload %q: %w", resource.Name, err)
		}

		destination, err := r.ensureResource(DestinationType, spec.Key, resource.Name, payload, nil)
		if err != nil {
			return nil, err
		}

		resolved[spec.Key] = destination
	}

	printResolvedResources("Resolved DMP destinations:", resolved)

	return resolved, nil
}

func (r *ResourceResolver) EnsureCommonTransforms(
	rootDir string,
	specs []CommonTransformSpec,
) (map[string]ResolvedResource, error) {
	resolved := make(map[string]ResolvedResource, len(specs))

	for _, spec := range specs {
		transform, _, err := r.ensureCommonResource(rootDir, TransformType, "transforms", spec.Key, nil)
		if err != nil {
			return nil, err
		}

		resolved[spec.Key] = transform
	}

	printResolvedResources("Resolved DMP transforms:", resolved)

	return resolved, nil
}

func (r *ResourceResolver) ensureCommonResource(
	rootDir string,
	resourceType ResourceType,
	commonDir string,
	key string,
	beforeCreate func(payload []byte, resource DMPResource) error,
) (ResolvedResource, []byte, error) {
	resource, payload, err := loadCommonPayload(rootDir, commonDir, key)
	if err != nil {
		return ResolvedResource{}, nil, err
	}

	resolved, err := r.ensureResource(resourceType, key, resource.Name, payload, beforeCreate)
	if err != nil {
		return ResolvedResource{}, nil, err
	}

	return resolved, payload, nil
}

func (r *ResourceResolver) ensureResource(
	resourceType ResourceType,
	key string,
	name string,
	payload []byte,
	beforeCreate func(payload []byte, resource DMPResource) error,
) (ResolvedResource, error) {
	existing, err := r.FindResourceByName(resourceType, name)
	if err != nil {
		return ResolvedResource{}, err
	}

	if existing != nil {
		fmt.Printf("DMP %s %q already exists with id %d\n", resourceType, existing.Name, existing.ID)

		return ResolvedResource{
			Key:  key,
			ID:   existing.ID,
			Name: existing.Name,
			Type: existing.Type,
		}, nil
	}

	var resource DMPResource
	if err := json.Unmarshal(payload, &resource); err != nil {
		return ResolvedResource{}, fmt.Errorf("decode DMP %s payload %q: %w", resourceType, name, err)
	}

	if beforeCreate != nil {
		if err := beforeCreate(payload, resource); err != nil {
			return ResolvedResource{}, err
		}
	}

	created, err := r.CreateResource(resourceType, payload)
	if err != nil {
		return ResolvedResource{}, err
	}

	fmt.Printf("DMP %s %q created with id %d\n", resourceType, created.Name, created.ID)

	return ResolvedResource{
		Key:  key,
		ID:   created.ID,
		Name: created.Name,
		Type: created.Type,
	}, nil
}

func (r *ResourceResolver) ListResources(resourceType ResourceType) ([]DMPResource, error) {
	resp, err := r.Client.Get(resourceType)
	if err != nil {
		return nil, fmt.Errorf("list DMP %s resources: %w", resourceType, err)
	}
	defer closeBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list DMP %s resources returned status %d: %s", resourceType, resp.StatusCode, string(body))
	}

	var resources []DMPResource
	if err := json.NewDecoder(resp.Body).Decode(&resources); err != nil {
		return nil, fmt.Errorf("decode DMP %s resources response: %w", resourceType, err)
	}

	return resources, nil
}

func (r *ResourceResolver) FindResourceByName(resourceType ResourceType, name string) (*DMPResource, error) {
	resources, err := r.ListResources(resourceType)
	if err != nil {
		return nil, err
	}

	for _, resource := range resources {
		if resource.Name == name {
			return &resource, nil
		}
	}

	return nil, nil
}

func (r *ResourceResolver) CreateResource(resourceType ResourceType, payload []byte) (DMPResource, error) {
	resp, err := r.Client.Post(resourceType, payload)
	if err != nil {
		return DMPResource{}, fmt.Errorf("create DMP %s resource: %w", resourceType, err)
	}
	defer closeBody(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return DMPResource{}, fmt.Errorf("create DMP %s resource returned status %d: %s", resourceType, resp.StatusCode, string(body))
	}

	var created DMPResource
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return DMPResource{}, fmt.Errorf("decode created DMP %s resource response: %w", resourceType, err)
	}

	return created, nil
}

func (r *ResourceResolver) ValidateConnection(payload []byte, name string) error {
	resp, err := r.Client.ValidateConnection(payload)
	if err != nil {
		return fmt.Errorf("validate DMP connection %q: %w", name, err)
	}
	defer closeBody(resp.Body)

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("validate DMP connection %q returned status %d: %s", name, resp.StatusCode, string(body))
	}

	fmt.Printf("DMP connection %q validated successfully\n", name)

	return nil
}

func loadCommonPayload(rootDir string, commonDir string, key string) (DMPResource, []byte, error) {
	path := filepath.Join(rootDir, "common", commonDir, key+".json")

	raw, err := os.ReadFile(path)
	if err != nil {
		return DMPResource{}, nil, fmt.Errorf("read common DMP payload %q: %w", path, err)
	}

	expanded := []byte(os.ExpandEnv(string(raw)))

	var payload DMPResource
	if err := json.Unmarshal(expanded, &payload); err != nil {
		return DMPResource{}, nil, fmt.Errorf("decode common DMP payload %q after env expansion: %w", path, err)
	}

	if payload.Name == "" {
		return DMPResource{}, nil, fmt.Errorf("common DMP payload %q is missing name", path)
	}

	return payload, expanded, nil
}

func printResolvedResources(title string, resources map[string]ResolvedResource) {
	keys := make([]string, 0, len(resources))
	for key := range resources {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	fmt.Println(title)

	for _, key := range keys {
		resource := resources[key]
		fmt.Printf("- %s: id=%d name=%s type=%s\n", key, resource.ID, resource.Name, resource.Type)
	}
}

func closeBody(body io.ReadCloser) {
	if err := body.Close(); err != nil {
		fmt.Printf("Failed to close response body: %v\n", err)
	}
}

func (r DMPResource) ShouldValidateConnection() bool {
	if r.Meta == nil || r.Meta.ValidateConnection == nil {
		return true
	}

	return *r.Meta.ValidateConnection
}
