package docs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSwaggerJSONAdaptsHost(t *testing.T) {
	raw, err := GetSwaggerJSON("api.example.com", "https")
	require.NoError(t, err)
	var spec map[string]any
	require.NoError(t, json.Unmarshal(raw, &spec))
	servers := spec["servers"].([]any)
	assert.Equal(t, "https://api.example.com", servers[0].(map[string]any)["url"])
	assert.Contains(t, spec["paths"], "/api/posts/ai-generate")
}

func TestGetSwaggerJSONInvalidSpec(t *testing.T) {
	orig := SpecJSON
	SpecJSON = []byte("{")
	defer func() { SpecJSON = orig }()
	_, err := GetSwaggerJSON("h", "http")
	assert.Error(t, err)
}
