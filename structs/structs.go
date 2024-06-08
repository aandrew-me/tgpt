package structs

type Params struct {
	ApiModel    string
	ApiKey      string
	Provider    string
	Temperature string
	Top_p       string
	Max_length  string
	Preprompt   string
	ThreadID    string
	Url         string
}

type ExtraOptions struct {
	ThreadID     string
	PrevMessages string
	Provider     string
}

type CommonResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}
