package koboldai

import (
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/stretchr/testify/assert"
)

func TestRequest(t *testing.T) {
	resp, err := NewRequest("What is 1+1", structs.Params{
		Provider:    "koboldai",
		Temperature: "0.5",
		Top_p:       "0.5",
		Max_length:  "300",
	})

	if err != nil {
		t.Fatalf("NewRequest returned an error: %v", err)

	}

	if resp == nil {
		t.Fatalf("NewRequest returned a nil response")
	}

	body, _ := io.ReadAll(resp.Body)

	assert := assert.New(t)

	fmt.Println("Statuscode:", resp.StatusCode)
	fmt.Println("Response:", string(body))

	assert.Nil(err, "Error should be Nil")
	assert.Equal(200, resp.StatusCode, "Response status code should be 200")
}

func TestRequestBodyUsesSamplingOptions(t *testing.T) {
	var decoded map[string]any
	body := newRequestBody("hi", structs.Params{Temperature: "0.2", Top_p: "0.9"})
	assert.NoError(t, json.Unmarshal([]byte(body), &decoded))
	assert.Equal(t, 0.2, decoded["temperature"])
	assert.Equal(t, 0.9, decoded["top_p"])
}

func TestRequestBodyDefaultsSamplingOptions(t *testing.T) {
	var decoded map[string]any
	body := newRequestBody("hi", structs.Params{Temperature: "warm"})
	assert.NoError(t, json.Unmarshal([]byte(body), &decoded))
	assert.Equal(t, 0.5, decoded["temperature"])
	assert.Equal(t, 0.5, decoded["top_p"])
}
