package koboldai

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	http "github.com/bogdanfinn/fhttp"

	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/structs"
)

type Response struct {
	Token string `json:"token"`
}

// newRequestBody builds the generate request body. The sampling values default
// to 0.5 and are replaced by --temperature / --top_p when the user sets them.
func newRequestBody(input string, params structs.Params) string {
	safeInput, _ := json.Marshal(input)

	temperature := "0.5"

	top_p := "0.5"

	if t, p := params.SamplingOptions(); t != nil || p != nil {
		if t != nil {
			temperature = strconv.FormatFloat(*t, 'f', -1, 64)
		}
		if p != nil {
			top_p = strconv.FormatFloat(*p, 'f', -1, 64)
		}
	}

	max_length := "300"

	return fmt.Sprintf(`{
		"prompt": %v,
		"temperature": %v,
		"top_p": %v,
		"max_length": %v
	  }
	`, string(safeInput), temperature, top_p, max_length)
}

func NewRequest(input string, params structs.Params) (*http.Response, error) {
	client, err := client.NewClient()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	data := strings.NewReader(newRequestBody(input, params))

	req, err := http.NewRequest("POST", "https://koboldai-koboldcpp-tiefighter.hf.space/api/extra/generate/stream", data)

	if err != nil {
		fmt.Println("\nSome error has occurred.")
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	// Setting all the required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Return response
	return (client.Do(req))
}

func GetMainText(line string) (mainText string) {
	var obj = "{}"
	if after, ok := strings.CutPrefix(line, "data: "); ok {
		obj = after
	}

	var d Response

	if err := json.Unmarshal([]byte(obj), &d); err != nil {
		return ""
	}

	if d.Token != "" {
		mainText = d.Token
		return mainText
	}

	return ""
}
