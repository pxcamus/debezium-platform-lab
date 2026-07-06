package dmp

// Resource Common interface for all resources type
type Resource interface {
	GetKey() string
	GetFile() string
	GetRefs() map[string]string
	GetType() ResourceType
}

type ResourceType string

const (
	ConnectionType  ResourceType = "connection"
	SourceType      ResourceType = "source"
	DestinationType ResourceType = "destination"
	PipelineType    ResourceType = "pipeline"
	TransformType   ResourceType = "transform"
)

type Manifest struct {
	Name         string                `yaml:"name"`
	Description  string                `yaml:"description"`
	Connections  []ConnectionResource  `yaml:"connections,omitempty"`
	Sources      []SourceResource      `yaml:"sources,omitempty"`
	Destinations []DestinationResource `yaml:"destinations,omitempty"`
	Transforms   []TransformResource   `yaml:"transforms,omitempty"`
	Pipelines    []PipelineResource    `yaml:"pipelines,omitempty"`
}

type Meta struct {
	ValidateConnection *bool `json:"validateConnection,omitempty" yaml:"validateConnection,omitempty"`
}

type ConnectionResource struct {
	Key  string            `yaml:"key"`
	File string            `yaml:"file"`
	Refs map[string]string `yaml:"refs,omitempty"`
	Meta Meta              `yaml:"_meta,omitempty"`
	Type ResourceType
}

func (c ConnectionResource) GetKey() string             { return c.Key }
func (c ConnectionResource) GetFile() string            { return c.File }
func (c ConnectionResource) GetRefs() map[string]string { return nil }
func (c ConnectionResource) GetType() ResourceType      { return ConnectionType }

type SourceResource struct {
	Key  string            `yaml:"key"`
	File string            `yaml:"file"`
	Refs map[string]string `yaml:"refs"`
}

func (s SourceResource) GetKey() string             { return s.Key }
func (s SourceResource) GetFile() string            { return s.File }
func (s SourceResource) GetRefs() map[string]string { return s.Refs }
func (s SourceResource) GetType() ResourceType      { return SourceType }

type DestinationResource struct {
	Key  string            `yaml:"key"`
	File string            `yaml:"file"`
	Refs map[string]string `yaml:"refs"`
}

func (d DestinationResource) GetKey() string             { return d.Key }
func (d DestinationResource) GetFile() string            { return d.File }
func (d DestinationResource) GetRefs() map[string]string { return d.Refs }
func (d DestinationResource) GetType() ResourceType      { return DestinationType }

type TransformResource struct {
	Key  string            `yaml:"key"`
	File string            `yaml:"file"`
	Refs map[string]string `yaml:"refs,omitempty"`
}

func (t TransformResource) GetKey() string             { return t.Key }
func (t TransformResource) GetFile() string            { return t.File }
func (t TransformResource) GetRefs() map[string]string { return t.Refs }
func (t TransformResource) GetType() ResourceType      { return TransformType }

type PipelineResource struct {
	Key  string            `yaml:"key"`
	File string            `yaml:"file"`
	Refs map[string]string `yaml:"refs"`
}

func (p PipelineResource) GetKey() string             { return p.Key }
func (p PipelineResource) GetFile() string            { return p.File }
func (p PipelineResource) GetRefs() map[string]string { return p.Refs }
func (p PipelineResource) GetType() ResourceType      { return PipelineType }
