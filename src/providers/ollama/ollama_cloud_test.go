package ollama

import (
	"encoding/json"
	"testing"

	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/stretchr/testify/assert"
)

func decodeCloudBody(t *testing.T, params structs.Params) map[string]any {
	t.Helper()
	encoded, err := cloudRequestBody("Hello", params, "gpt-oss:120b")
	assert.NoError(t, err)

	var decoded map[string]any
	assert.NoError(t, json.Unmarshal(encoded, &decoded))
	return decoded
}

func TestCloudRequestBodySampling(t *testing.T) {
	decoded := decodeCloudBody(t, structs.Params{Temperature: "0.2", Top_p: "0.9"})

	assert.Equal(t, 0.2, decoded["temperature"])
	assert.Equal(t, 0.9, decoded["top_p"])
}

func TestCloudRequestBodyOmitsUnsetSampling(t *testing.T) {
	decoded := decodeCloudBody(t, structs.Params{})

	assert.NotContains(t, decoded, "temperature")
	assert.NotContains(t, decoded, "top_p")
	assert.Equal(t, "gpt-oss:120b", decoded["model"])
	assert.Equal(t, true, decoded["stream"])
	assert.Len(t, decoded["messages"], 1)
}
