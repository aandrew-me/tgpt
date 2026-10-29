package structs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSamplingOptions(t *testing.T) {
	temperature, topP := Params{Temperature: "0.2", Top_p: "0.9"}.SamplingOptions()
	assert.Equal(t, 0.2, *temperature)
	assert.Equal(t, 0.9, *topP)

	temperature, topP = Params{Temperature: "0"}.SamplingOptions()
	assert.Equal(t, 0.0, *temperature)
	assert.Nil(t, topP)

	temperature, topP = Params{Temperature: "warm", Top_p: ""}.SamplingOptions()
	assert.Nil(t, temperature)
	assert.Nil(t, topP)
}
