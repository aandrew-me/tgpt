package structs

type Params struct {
	ApiModel     string
	ApiKey       string
	Provider     string
	Temperature  string
	Top_p        string
	Max_length   string
	Preprompt    string
	ThreadID     string
	Url          string
	PrevMessages string
}

type ExtraOptions struct {
	IsGetSilent       bool
	IsGetWhole        bool
	IsGetCommand      bool
	IsNormal          bool
	IsGetCode         bool
	IsInteractive     bool
	DisableInputLimit bool
}

type CommonResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}
