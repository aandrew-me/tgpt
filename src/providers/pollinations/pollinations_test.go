package pollinations

import (
	"encoding/json"
	"testing"

	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/stretchr/testify/assert"
)

func TestRequestBodyIncludesSamplingOptions(t *testing.T) {
	encoded, err := json.Marshal(newRequestBody(structs.Params{Temperature: "0.2", Top_p: "0.9"}))
	assert.NoError(t, err)

	var decoded map[string]any
	assert.NoError(t, json.Unmarshal(encoded, &decoded))
	assert.Equal(t, 0.2, decoded["temperature"])
	assert.Equal(t, 0.9, decoded["top_p"])
}

func TestRequestBodyOmitsUnsetSamplingOptions(t *testing.T) {
	encoded, err := json.Marshal(newRequestBody(structs.Params{}))
	assert.NoError(t, err)

	var decoded map[string]any
	assert.NoError(t, json.Unmarshal(encoded, &decoded))
	assert.NotContains(t, decoded, "temperature")
	assert.NotContains(t, decoded, "top_p")
}
