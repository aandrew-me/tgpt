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
}

type ExtraOptions struct {
	ThreadID     string
	PrevMessages string
	Provider     string
}
