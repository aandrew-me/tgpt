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
	PrevMessages []any
	SystemPrompt string
}

type ExtraOptions struct {
	IsGetSilent        bool
	IsGetWhole         bool
	IsGetCommand       bool
	IsNormal           bool
	IsGetCode          bool
	IsInteractive      bool
	IsInteractiveShell bool
	AutoExec           bool
	IsFind             bool // IsFind enable web search functionality
	IsInteractiveFind  bool // IsInteractiveFind enable interactive web search mode
	Verbose            bool // Verbose enable detailed search output
}

type CommonResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

type ImageParams struct {
	Params
	Height            int
	Width             int
	Out               string
	ImgNegativePrompt string
	ImgRatio          string
	ImgCount          string
}

type DefaultMessage struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

type UserMessagePhind struct {
	Content  string `json:"content"`
	Role     string `json:"role"`
	Metadata string `json:"metadata"`
}

type AssistantResponsePhind struct {
	Content  string `json:"content"`
	Role     string `json:"role"`
	Metadata string `json:"metadata"`
	Name     string `json:"name"`
}

type KimiResponse struct {
	Event string `json:"event"`
	Text  string `json:"text"`
}
