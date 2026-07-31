package structs

type Params struct {
	ApiModel        string
	ApiKey          string
	Provider        string
	Temperature     string
	Top_p           string
	Max_length      string
	Preprompt       string
	ThreadID        string
	Url             string
	PrevMessages    []any
	SystemPrompt    string
	RotateProviders string
	Tools           []any
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
	IsFind             bool   // IsFind enable web search functionality
	IsInteractiveFind  bool   // IsInteractiveFind enable interactive web search mode
	Verbose            bool   // Verbose enable detailed search output
	SearchProvider     string // Search provider: "exa" (default) or "google"
	IsToolFollowUp     bool   // IsToolFollowUp marks a request made to continue after tool execution
	ToolDepth          int    // ToolDepth tracks recursion depth of tool execution loops
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolCallDelta struct {
	Index    int              `json:"index"`
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type,omitempty"`
	Function ToolCallFunction `json:"function"`
}

type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

type CommonResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Delta struct {
			Content   string          `json:"content"`
			ToolCalls []ToolCallDelta `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
}

type ToolMessage struct {
	Role       string `json:"role"`
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name"`
	Content    string `json:"content"`
}

type AssistantToolCallMessage struct {
	Role      string     `json:"role"`
	Content   any        `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls"`
}

type PowerBrainResponse struct {
	Data string `json:"data"`
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