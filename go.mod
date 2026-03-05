module mcp-server-mock

go 1.26

require (
	github.com/santhosh-tekuri/jsonschema/v5 v5.0.0
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/santhosh-tekuri/jsonschema/v5 => ./third_party/jsonschema

// Empty stub to satisfy yaml.v3's transitive test dependency on gopkg.in/check.v1
replace gopkg.in/check.v1 => ./third_party/gocheck
