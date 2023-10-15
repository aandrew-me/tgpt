package main

import (
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequest(t *testing.T) {
	resp, err := newRequest("What is 1+1")
	body, _ := io.ReadAll(resp.Body)

	assert := assert.New(t)

	fmt.Println("Statuscode:", resp.StatusCode);
	fmt.Println("Response:", string(body))

	assert.Nil(err, "Error should be Nil")
	assert.Equal(200, resp.StatusCode, "Response status code should be 200")

}