package openai

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requestBodyFor(t *testing.T, params structs.Params) map[string]any {
	t.Helper()

	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &body))
	}))
	defer server.Close()

	params.Url = server.URL
	resp, err := NewRequest("hello", params)
	require.NoError(t, err)
	resp.Body.Close()

	return body
}

func TestRequestCarriesTemperatureAndTopP(t *testing.T) {
	body := requestBodyFor(t, structs.Params{Temperature: "0.2", Top_p: "0.9"})

	assert.Equal(t, 0.2, body["temperature"])
	assert.Equal(t, 0.9, body["top_p"])
}

func TestRequestLeavesOutUnsetSampling(t *testing.T) {
	body := requestBodyFor(t, structs.Params{})

	assert.NotContains(t, body, "temperature")
	assert.NotContains(t, body, "top_p")
}
