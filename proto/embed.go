package proto

import _ "embed"

//go:embed generated/openapi.yaml
var SwaggerJsonDoc []byte
