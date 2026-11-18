// Package docs menyediakan spesifikasi OpenAPI (Swagger) yang beradaptasi dengan host pemanggil.
package docs

import (
	_ "embed"
	"encoding/json"
)

//go:embed swagger.json
var embeddedSpec []byte

// SpecJSON adalah spesifikasi OpenAPI mentah (dapat diganti pada pengujian).
var SpecJSON = embeddedSpec

// GetSwaggerJSON mengembalikan spesifikasi dengan server URL sesuai host & scheme pemanggil.
func GetSwaggerJSON(host, scheme string) ([]byte, error) {
	var spec map[string]any
	if err := json.Unmarshal(SpecJSON, &spec); err != nil {
		return nil, err
	}
	spec["servers"] = []map[string]string{{"url": scheme + "://" + host}}
	return json.MarshalIndent(spec, "", "  ")
}
