package aihorde

import (
	"encoding/json"
	"testing"

	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/stretchr/testify/assert"
)

func marshalParams(t *testing.T, params structs.Params) map[string]any {
	t.Helper()
	raw, err := json.Marshal(generationParams(params))
	assert.NoError(t, err)
	var decoded map[string]any
	assert.NoError(t, json.Unmarshal(raw, &decoded))
	return decoded
}

func TestGenerationParamsSendsTemperatureZero(t *testing.T) {
	decoded := marshalParams(t, structs.Params{Temperature: "0"})
	temp, ok := decoded["temperature"]
	assert.True(t, ok, "temperature was omitted: %v", decoded)
	assert.Equal(t, 0.0, temp)
}

func TestGenerationParamsSendsSetTemperature(t *testing.T) {
	decoded := marshalParams(t, structs.Params{Temperature: "0.2"})
	assert.InDelta(t, 0.2, decoded["temperature"], 1e-9)
}

func TestGenerationParamsKeepsDefaultTemperature(t *testing.T) {
	decoded := marshalParams(t, structs.Params{})
	assert.InDelta(t, 0.7, decoded["temperature"], 1e-9)

	decoded = marshalParams(t, structs.Params{Temperature: "warm"})
	assert.InDelta(t, 0.7, decoded["temperature"], 1e-9)

	decoded = marshalParams(t, structs.Params{Temperature: "3"})
	assert.InDelta(t, 0.7, decoded["temperature"], 1e-9)
}
