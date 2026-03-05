package spec

type ToolSpec struct {
	Type          string         `yaml:"type"`
	Name          string         `yaml:"name"`
	Description   string         `yaml:"description"`
	AfterCallHint string         `yaml:"afterCallHint,omitempty"`
	InputSchema   map[string]any `yaml:"inputSchema"`

	Source string `yaml:"-"`
}
