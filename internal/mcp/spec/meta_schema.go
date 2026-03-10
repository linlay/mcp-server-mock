package spec

import "encoding/json"

const toolMetaSchemaJSON = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["type", "name", "description", "inputSchema"],
  "properties": {
    "type": {"type": "string", "minLength": 1},
    "name": {"type": "string", "minLength": 1},
    "label": {"type": "string", "minLength": 1},
    "description": {"type": "string", "minLength": 1},
    "afterCallHint": {"type": "string"},
    "inputSchema": {"type": "object"},
    "toolAction": {"type": "boolean"},
    "toolType": {"type": "string", "minLength": 1},
    "viewportKey": {"type": "string", "minLength": 1}
  },
  "additionalProperties": false
}`

var metaSchema map[string]any

func init() {
	if err := json.Unmarshal([]byte(toolMetaSchemaJSON), &metaSchema); err != nil {
		panic("invalid meta schema JSON: " + err.Error())
	}
}

func MetaSchema() map[string]any {
	return metaSchema
}
