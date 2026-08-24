package helper

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aandrew-me/tgpt/v2/src/bubbletea"
	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/clipboard"
	"github.com/aandrew-me/tgpt/v2/src/providers"
	"github.com/aandrew-me/tgpt/v2/src/search"
	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/aandrew-me/tgpt/v2/src/tools"
	"github.com/aandrew-me/tgpt/v2/src/utils"
	http "github.com/bogdanfinn/fhttp"
	"github.com/fatih/color"
	"github.com/olekukonko/ts"
	"golang.org/x/mod/semver"
)

type Data struct {
	Version string `json:"version"`
}

type Response struct {
	Completion string `json:"completion"`
}

type ImgResponse struct {
	Images []string `json:"images"`
}

var (
	OperatingSystem string
	ShellName       string
	ShellOptions    []string
	ShellConfigFile string
)

type versionResponse struct {
	Version string `json:"version"`
}

var (
	bold       = color.New(color.Bold)
	boldBlue   = color.New(color.Bold, color.FgBlue)
	boldViolet = color.New(color.Bold, color.FgMagenta)
	codeText   = color.New(color.FgGreen, color.Bold)
)

var lastSuccessfulProvider string

type streamFormatter struct {
	tickCount       int
	previousWasTick bool
	isCode          bool
	isGreen         bool
	isTick          bool
	isRealCode      bool
	lineLength      int
	termWidth       int
	hasTermWidth    bool
	provider        string
}

func newStreamFormatter(provider string) *streamFormatter {
	f := &streamFormatter{provider: provider}
	if size, err := ts.GetSize(); err == nil {
		f.termWidth = size.Col()
		f.hasTermWidth = true
	}
	return f
}

func (f *streamFormatter) updateLineLength(word string) {
	if !f.hasTermWidth || f.provider == "gemini" {
		return
	}
	if word == "\n" {
		f.lineLength = 0

		return
	}
	wordLength := len([]rune(word))
	if f.termWidth-f.lineLength < wordLength {
		fmt.Print("\n")
		f.lineLength = 0
	}
	f.lineLength += wordLength
}

func (f *streamFormatter) writeChar(word string) {
	f.updateLineLength(word)

	if word == "`" {
		f.tickCount++
		f.isTick = true
		if f.tickCount == 2 && !f.previousWasTick {
			f.tickCount = 0
		} else if f.tickCount >= 6 && f.tickCount%2 == 0 && f.previousWasTick {
			f.tickCount = 0
		}
		f.isGreen = false
		f.isCode = false
	} else {
		if word == "\n" {
			f.lineLength = 0
		}
		f.isTick = false
		if f.tickCount == 1 {
			f.isGreen = true
		} else if f.tickCount >= 3 {
			f.isCode = true
		}
	}

	switch {
	case f.isCode:
		codeText.Print(word)
	case f.isGreen:
		boldBlue.Print(word)
	case !f.isTick:
		fmt.Print(word)
	default:
		if f.tickCount > 3 || f.isRealCode || (f.tickCount == 0 && f.previousWasTick) {
			fmt.Print(word)
		}
	}

	f.previousWasTick = word == "`"
}

func (f *streamFormatter) writeText(text string) {
	f.isRealCode = text == "``" || text == "```"
	for _, ch := range text {
		f.writeChar(string(ch))
	}
}

var xmlTagTargets = []string{"cmd>", "/cmd>", "search>", "/search>"}

func isXMLTagPrefix(s string) bool {
	if len(s) < 1 || s[0] != '<' {
		return false
	}
	rest := s[1:]
	for _, t := range xmlTagTargets {
		if strings.HasPrefix(t, rest) {
			return true
		}
	}
	return false
}

func isCompleteXMLTag(s string) bool {
	switch s {
	case "<cmd>", "</cmd>", "<search>", "</search>":
		return true
	}
	return false
}

type interactiveFormatter struct {
	*streamFormatter
	xmlBuffer strings.Builder
	inXMLTag  bool
}

func newInteractiveFormatter(provider string) *interactiveFormatter {
	return &interactiveFormatter{streamFormatter: newStreamFormatter(provider)}
}

func (f *interactiveFormatter) flushXMLBuffer() {
	buf := f.xmlBuffer.String()
	f.inXMLTag = false
	f.xmlBuffer.Reset()

	if buf == "" {
		return
	}
	f.streamFormatter.writeChar(buf[:1])
	f.writeText(buf[1:])
}

func (f *interactiveFormatter) writeText(text string) {
	for _, ch := range text {
		char := string(ch)
		if !f.inXMLTag {
			if ch == '<' {
				f.inXMLTag = true
				f.xmlBuffer.Reset()
				f.xmlBuffer.WriteRune(ch)
				continue
			}
			f.streamFormatter.writeChar(char)
			continue
		}
		f.xmlBuffer.WriteRune(ch)
		buf := f.xmlBuffer.String()
		switch {
		case isCompleteXMLTag(buf):
			f.inXMLTag = false
			f.xmlBuffer.Reset()
		case !isXMLTagPrefix(buf):
			f.flushXMLBuffer()
		}
	}
}

func GetData(input string, params structs.Params, extraOptions structs.ExtraOptions) ([]interface{}, string) {
	responseTxt, turnMessages, err := MakeRequestAndGetData(input, params, extraOptions)
	if err != nil {
		if errors.Is(err, bubbletea.ErrInterrupted) {
			bubbletea.RestoreTerminal()
			os.Exit(130)
		}
		return nil, ""
	}

	fmt.Print("\n\n")

	// turnMessages holds the fully-ordered messages for this turn (user input,
	// any tool call/tool result pairs, and the final assistant response) when
	// tool calling was involved. Fall back to a plain user/assistant pair
	// otherwise.
	if len(turnMessages) > 0 {
		return turnMessages, responseTxt
	}

	return []interface{}{
		structs.DefaultMessage{Content: input, Role: "user"},
		structs.DefaultMessage{Content: responseTxt, Role: "assistant"},
	}, responseTxt
}

var spinChars = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}

type Spinner struct {
	stop    atomic.Bool
	done    chan struct{}
	mu      sync.Mutex
	message string
	width   int
}

func StartSpinner(message string) *Spinner {
	s := &Spinner{
		message: message,
		done:    make(chan struct{}),
	}
	go s.run()
	return s
}

func (s *Spinner) SetMessage(message string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

func (s *Spinner) Stop() {
	if s == nil {
		return
	}
	s.stop.Store(true)
	<-s.done
}

var (
	spinnerMu     sync.Mutex
	activeSpinner *Spinner
)

// Whether to show status during tool calls
func statusEnabled(extraOptions structs.ExtraOptions) bool {
	return !extraOptions.IsGetSilent && !extraOptions.IsGetWhole
}

func showStatus(enabled bool, message string) {
	if !enabled {
		return
	}
	spinnerMu.Lock()
	defer spinnerMu.Unlock()
	if activeSpinner == nil {
		activeSpinner = StartSpinner(message)
		return
	}
	activeSpinner.SetMessage(message)
}

func hideStatus() {
	spinnerMu.Lock()
	s := activeSpinner
	activeSpinner = nil
	spinnerMu.Unlock()
	s.Stop()
}

func (s *Spinner) run() {
	defer close(s.done)

	i := 0
	for !s.stop.Load() {
		s.mu.Lock()
		line := spinChars[i] + " " + s.message
		if runes := len([]rune(line)); runes > s.width {
			s.width = runes
		}
		s.mu.Unlock()

		fmt.Print("\r" + line)
		i = (i + 1) % len(spinChars)
		time.Sleep(80 * time.Millisecond)
	}

	s.mu.Lock()
	width := s.width
	s.mu.Unlock()
	fmt.Print("\r" + strings.Repeat(" ", width) + "\r\033[K")
}

func canWriteToDir(path string) bool {
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".perm_test_*")
	if err != nil {
		return false
	}
	tmpFile.Close()
	os.Remove(tmpFile.Name())
	return true
}

func isOwnedBySystemPkgMgr(execPath string) bool {
	if pacman, err := exec.LookPath("pacman"); err == nil {
		if err := exec.Command(pacman, "-Qo", execPath).Run(); err == nil {
			return true
		}
	}
	if dpkg, err := exec.LookPath("dpkg"); err == nil {
		if err := exec.Command(dpkg, "-S", execPath).Run(); err == nil {
			return true
		}
	}
	if rpm, err := exec.LookPath("rpm"); err == nil {
		if err := exec.Command(rpm, "-qf", execPath).Run(); err == nil {
			return true
		}
	}
	return false
}

func DetectPackageManager(executablePath string) (isPkgMgr bool, pkgName string, updateCmd string) {
	lowerPath := strings.ToLower(filepath.ToSlash(executablePath))

	// Scoop (Windows)
	if strings.Contains(lowerPath, "/scoop/") {
		return true, "Scoop", "scoop update tgpt"
	}

	// Chocolatey (Windows)
	if strings.Contains(lowerPath, "/chocolatey/") || strings.Contains(lowerPath, "/choco/") {
		return true, "Chocolatey", "choco upgrade tgpt"
	}

	// Homebrew (macOS / Linux)
	if strings.Contains(lowerPath, "/cellar/") || strings.Contains(lowerPath, "/homebrew/") || strings.Contains(lowerPath, "/linuxbrew/") {
		return true, "Homebrew", "brew upgrade tgpt"
	}

	// System package manager (Linux pacman / dpkg / rpm)
	if runtime.GOOS == "linux" && isOwnedBySystemPkgMgr(executablePath) {
		return true, "system package manager", "use your package manager (e.g. pacman, apt, dnf) to update"
	}

	// Go install (go/bin/tgpt)
	gobin := os.Getenv("GOBIN")
	if gobin != "" && strings.HasPrefix(lowerPath, strings.ToLower(filepath.ToSlash(gobin))) {
		return true, "Go", "go install github.com/aandrew-me/tgpt/v2@latest"
	}
	gopath := os.Getenv("GOPATH")
	if gopath != "" {
		for _, entry := range filepath.SplitList(gopath) {
			entrySlash := strings.ToLower(filepath.ToSlash(entry))
			if entrySlash != "" && strings.Contains(lowerPath, entrySlash+"/bin") {
				return true, "Go", "go install github.com/aandrew-me/tgpt/v2@latest"
			}
		}
	}
	if strings.Contains(lowerPath, "/go/bin/") {
		return true, "Go", "go install github.com/aandrew-me/tgpt/v2@latest"
	}

	return false, "", ""
}

func escapePowerShellArg(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func Update(localVersion string, executablePath string) {
	if runtime.GOOS == "android" {
		fmt.Println("This feature is not supported on your Operating System")
		return
	}

	if isPkgMgr, pkgName, updateCmd := DetectPackageManager(executablePath); isPkgMgr {
		fmt.Printf("tgpt was installed via a package manager (%s).\n", pkgName)
		fmt.Printf("Please update it using your package manager: %s\n", updateCmd)
		return
	}

	url := "https://raw.githubusercontent.com/aandrew-me/tgpt/main/version.txt"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating request:", err)
		return
	}

	httpClient, err := client.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating HTTP client:", err)
		return
	}

	res, err := httpClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error fetching remote version:", err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Error fetching version: HTTP %d\n", res.StatusCode)
		return
	}

	var verData versionResponse
	if err := json.NewDecoder(res.Body).Decode(&verData); err != nil {
		fmt.Fprintln(os.Stderr, "Error parsing version JSON:", err)
		return
	}

	localSemver := "v" + strings.TrimPrefix(localVersion, "v")
	remoteSemver := "v" + strings.TrimPrefix(verData.Version, "v")

	if semver.Compare(localSemver, remoteSemver) >= 0 {
		fmt.Println("You are already using the latest version:", remoteSemver)
		return
	}

	fmt.Printf("Updating from %s to %s...\n", localSemver, remoteSemver)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		script := fmt.Sprintf("& { $(irm https://raw.githubusercontent.com/aandrew-me/tgpt/refs/heads/main/install-win.ps1) } -Path %s", escapePowerShellArg(executablePath))
		cmd = exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	} else {
		useSudo := !canWriteToDir(executablePath)
		var script string
		if useSudo {
			fmt.Println("Elevated privileges required to write to:", filepath.Dir(executablePath))
			fmt.Println("Requesting sudo access...")
			script = `set -o pipefail; curl -sSLf https://raw.githubusercontent.com/aandrew-me/tgpt/main/install | sudo bash -s -- "$1"`
		} else {
			script = `set -o pipefail; curl -sSLf https://raw.githubusercontent.com/aandrew-me/tgpt/main/install | bash -s -- "$1"`
		}

		// Pass executablePath as positional parameter $1 to safely prevent shell injection.
		cmd = exec.Command("bash", "-c", script, "bash", executablePath)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "\nFailed to update. Error:", err)
		return
	}

	fmt.Println("Successfully updated.")
}

func CodeGenerate(input string, params structs.Params, extraOptions structs.ExtraOptions) {
	codePrompt := fmt.Sprintf(
		"Your Role: Provide only code as output without any description.\n"+
			"IMPORTANT: Provide only plain text without Markdown formatting.\n"+
			"IMPORTANT: Do not include markdown formatting.\n"+
			"If there is a lack of details, provide most logical solution. You are not allowed to ask for more details.\n"+
			"Ignore any potential risk of errors or confusion.\n\n"+
			"Request:%s\nCode:", input,
	)

	_, _, _ = MakeRequestAndGetData(codePrompt, params, extraOptions)
}

func SetShellAndOSVars() {
	switch runtime.GOOS {
	case "windows":
		OperatingSystem = "Windows"
		if len(os.Getenv("PSModulePath")) > 0 {
			ShellName = "powershell.exe"
			ShellOptions = []string{"-Command"}
		} else {
			ShellName = "cmd.exe"
			ShellOptions = []string{"/C"}
		}
		ShellConfigFile = ""
		return
	case "darwin":
		OperatingSystem = "MacOS"
	case "linux":
		distro := ""
		if path, err := exec.LookPath("lsb_release"); err == nil {
			if result, err := exec.Command(path, "-si").Output(); err == nil {
				distro = strings.TrimSpace(string(result))
			}
		}
		OperatingSystem = "Linux/" + distro
	default:
		OperatingSystem = runtime.GOOS
	}

	homeDir := os.Getenv("HOME")
	shellEnv := os.Getenv("SHELL")

	switch {
	case strings.Contains(shellEnv, "zsh"):
		ShellName = shellEnv
		ShellConfigFile = homeDir + "/.zshrc"
	case strings.Contains(shellEnv, "bash"):
		ShellName = shellEnv
		ShellConfigFile = homeDir + "/.bashrc"
	case strings.Contains(shellEnv, "fish"):
		ShellName = shellEnv
		ShellConfigFile = homeDir + "/.config/fish/config.fish"
	case shellEnv != "":
		ShellName = shellEnv
		ShellConfigFile = homeDir + "/.bashrc"
	default:
		if _, err := exec.LookPath("bash"); err == nil {
			ShellName = "bash"
			ShellConfigFile = homeDir + "/.bashrc"
		} else {
			ShellName = "/bin/sh"
			ShellConfigFile = homeDir + "/.profile"
		}
	}
	ShellOptions = []string{"-c"}
}

func ShellCommand(input string, params structs.Params, extraOptions structs.ExtraOptions) {
	SetShellAndOSVars()
	shellPrompt := fmt.Sprintf(
		"Your role: Provide only plain text without Markdown formatting. "+
			"Do not show any warnings or information regarding your capabilities. "+
			"Do not provide any description. If you need to store any data, assume it will be stored in the chat. "+
			"Provide only %s command for %s without any description. "+
			"If there is a lack of details, provide most logical solution. "+
			"Ensure the output is a valid shell command. If multiple steps required try to combine them together. "+
			"Prompt: %s\n\nCommand:",
		ShellName, OperatingSystem, input,
	)
	GetCommand(shellPrompt, params, extraOptions)
}

func GetCommand(shellPrompt string, params structs.Params, extraOptions structs.ExtraOptions) {
	_, _, _ = MakeRequestAndGetData(shellPrompt, params, extraOptions)
}

type RESPONSE struct {
	Tagname string `json:"tag_name"`
	Body    string `json:"body"`
}

func GetVersionHistory() {
	req, err := http.NewRequest("GET", "https://api.github.com/repos/aandrew-me/tgpt/releases/latest", nil)
	if err != nil {
		fmt.Fprint(os.Stderr, "Some error has occurred\n\n")
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	httpClient, err := client.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating client:", err)
		os.Exit(1)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		fmt.Fprint(os.Stderr, "Check your internet connection\n\n")
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var release RESPONSE
	if err := json.Unmarshal(resBody, &release); err != nil {
		fmt.Fprintln(os.Stderr, "Error parsing release JSON:", err)
		os.Exit(1)
	}

	boldBlue.Println("Release", release.Tagname)
	fmt.Println(release.Body)
	fmt.Println()
}

func GetWholeText(input string, extraOptions structs.ExtraOptions, params structs.Params) {
	_, _, _ = MakeRequestAndGetData(input, params, extraOptions)
}

func GetLastCodeBlock(markdown string) string {
	return utils.GetLastCodeBlock(markdown)
}

func formatToolArgs(rawArgs string) string {
	var args map[string]any
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		if len(rawArgs) > 100 {
			runes := []rune(rawArgs)
			return string(runes[:100]) + "..."
		}
		return rawArgs
	}

	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	maxValLen := 60
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v := args[k]
		var valStr string
		switch val := v.(type) {
		case string:
			if len(val) > maxValLen {
				runes := []rune(val)
				valStr = fmt.Sprintf("%q...", string(runes[:maxValLen]))
			} else {
				valStr = fmt.Sprintf("%q", val)
			}
		default:
			b, err := json.Marshal(val)
			str := string(b)
			if err != nil {
				str = fmt.Sprintf("%v", val)
			}
			if len(str) > maxValLen {
				runes := []rune(str)
				valStr = string(runes[:maxValLen]) + "..."
			} else {
				valStr = str
			}
		}
		parts = append(parts, fmt.Sprintf("%s=%s", k, valStr))
	}

	result := strings.Join(parts, ", ")
	maxTotalLen := 120
	if len(result) > maxTotalLen {
		runes := []rune(result)
		return string(runes[:maxTotalLen]) + "..."
	}
	return result
}

type toolCallAccumulator struct {
	id   string
	name string
	args strings.Builder
}

func HandleEachPart(resp *http.Response, input string, params structs.Params, extraOptions structs.ExtraOptions) (string, []interface{}) {
	scanner := bufio.NewScanner(resp.Body)
	formatter := newStreamFormatter(params.Provider)
	fullText := ""
	toolCallMap := make(map[int]*toolCallAccumulator)

	for scanner.Scan() {
		line := scanner.Text()
		mainText := providers.GetMainText(line, params.Provider, input)
		if len(mainText) > 0 {
			fullText += mainText
			formatter.writeText(mainText)
		}

		var obj = "{}"
		if after, ok := strings.CutPrefix(line, "data: "); ok {
			obj = after
		}
		var d structs.CommonResponse
		if err := json.Unmarshal([]byte(obj), &d); err == nil && len(d.Choices) > 0 {
			for _, tcDelta := range d.Choices[0].Delta.ToolCalls {
				acc, ok := toolCallMap[tcDelta.Index]
				if !ok {
					acc = &toolCallAccumulator{}
					toolCallMap[tcDelta.Index] = acc
				}
				if tcDelta.ID != "" {
					acc.id = tcDelta.ID
				}
				if tcDelta.Function.Name != "" {
					acc.name = tcDelta.Function.Name
				}
				if tcDelta.Function.Arguments != "" {
					acc.args.WriteString(tcDelta.Function.Arguments)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Some error has occurred. Error:", err)
		os.Exit(1)
	}

	if len(toolCallMap) > 0 {
		keys := make([]int, 0, len(toolCallMap))
		for k := range toolCallMap {
			keys = append(keys, k)
		}
		sort.Ints(keys)

		var toolCalls []structs.ToolCall
		for _, k := range keys {
			acc := toolCallMap[k]
			if acc != nil && acc.name != "" {
				toolCalls = append(toolCalls, structs.ToolCall{
					ID:   acc.id,
					Type: "function",
					Function: structs.ToolCallFunction{
						Name:      acc.name,
						Arguments: acc.args.String(),
					},
				})
			}
		}

		if len(toolCalls) > 0 {
			resp.Body.Close()

			if fullText != "" && !strings.HasSuffix(fullText, "\n") {
				fmt.Println()
			}

			// Terminal follow-up: never process tool calls again.
			if extraOptions.ToolDepth >= 5 && extraOptions.IsToolFollowUp {
				return fullText, nil
			}

			if extraOptions.ToolDepth >= 5 {
				fmt.Fprintln(os.Stderr, "\nReached maximum tool execution depth. Stopping tool calls.")
				noToolsParams := params
				noToolsParams.Tools = nil
				followUpOptions := extraOptions
				followUpOptions.IsToolFollowUp = true
				followUpText, followUpTurnMessages, _ := MakeRequestAndGetData("", noToolsParams, followUpOptions)
				if len(followUpTurnMessages) > 0 {
					return fullText + followUpText, followUpTurnMessages
				}
				finalAssistantMsg := structs.DefaultMessage{
					Role:    "assistant",
					Content: followUpText,
				}
				return fullText + followUpText, []any{finalAssistantMsg}
			}

			turnMessages := make([]any, 0)

			if input != "" {
				userMsg := structs.DefaultMessage{
					Role:    "user",
					Content: input,
				}
				params.PrevMessages = append(params.PrevMessages, userMsg)
				turnMessages = append(turnMessages, userMsg)
			}

			assistantMsg := structs.AssistantToolCallMessage{
				Role:      "assistant",
				ToolCalls: toolCalls,
			}
			params.PrevMessages = append(params.PrevMessages, assistantMsg)
			turnMessages = append(turnMessages, assistantMsg)

			statusOn := statusEnabled(extraOptions)

			for _, tc := range toolCalls {
				if extraOptions.Verbose {
					boldBlue.Printf("\n[Tool Call] %s(%s)\n", tc.Function.Name, tc.Function.Arguments)
				}

				if tc.Function.Name == "execute_command" && !extraOptions.AutoExec {
					hideStatus()
				} else {
					showStatus(statusOn, "Running "+tc.Function.Name)
				}

				preConfirmCtx := context.Background()
				if extraOptions.AutoExec {
					preConfirmCtx = context.WithValue(preConfirmCtx, tools.AutoExecKey, true)
				}

				var toolOutput string
				var err error
				proceed, cancelMsg, confirmErr := tools.PreConfirm(preConfirmCtx, tc.Function.Name, tc.Function.Arguments)
				if confirmErr != nil {
					hideStatus()
					if errors.Is(confirmErr, bubbletea.ErrInterrupted) {
						bubbletea.RestoreTerminal()
						os.Exit(130)
					}
					if cancelMsg != "" {
						toolOutput = cancelMsg
					} else {
						toolOutput = confirmErr.Error()
					}
					err = confirmErr
				} else if !proceed {
					toolOutput = cancelMsg
					err = fmt.Errorf("%s", cancelMsg)
				} else {
					// The confirmation (if any) has already been obtained above,
					// so the 60s execution timeout starts only now and is not
					// consumed by time spent waiting on user input.
					execCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
					execCtx = context.WithValue(execCtx, tools.ConfirmedKey, true)
					if extraOptions.AutoExec {
						execCtx = context.WithValue(execCtx, tools.AutoExecKey, true)
					}

					toolOutput, err = tools.DefaultRegistry.Execute(execCtx, tc.Function.Name, tc.Function.Arguments)
					cancel()
				}
				hideStatus()

				if err != nil && proceed {
					toolOutput = fmt.Sprintf("Error executing tool: %v", err)
				}

				if !extraOptions.Verbose && extraOptions.IsNormal {
					mark := "\u2705"
					if err != nil {
						mark = "\u274c"
					}
					boldViolet.Printf("Used Tool %s(%s) %s\n", tc.Function.Name, formatToolArgs(tc.Function.Arguments), mark)
				}

				if extraOptions.Verbose {
					bold.Printf("[Tool Output] %s\n", toolOutput)
				}

				toolMsg := structs.ToolMessage{
					Role:       "tool",
					ToolCallID: tc.ID,
					Name:       tc.Function.Name,
					Content:    toolOutput,
				}
				params.PrevMessages = append(params.PrevMessages, toolMsg)
				turnMessages = append(turnMessages, toolMsg)
			}

			followUpOptions := extraOptions
			followUpOptions.IsToolFollowUp = true
			followUpOptions.ToolDepth++

			followUpText, followUpTurnMessages, _ := MakeRequestAndGetData("", params, followUpOptions)

			if len(followUpTurnMessages) > 0 {
				turnMessages = append(turnMessages, followUpTurnMessages...)
				return fullText + followUpText, turnMessages
			}

			finalAssistantMsg := structs.DefaultMessage{
				Role:    "assistant",
				Content: followUpText,
			}
			turnMessages = append(turnMessages, finalAssistantMsg)

			return fullText + followUpText, turnMessages
		}
	}

	return fullText, nil
}

func HandleEachPartInteractiveShell(resp *http.Response, input string, params structs.Params) string {
	scanner := bufio.NewScanner(resp.Body)
	formatter := newInteractiveFormatter(params.Provider)
	fullText := ""

	for scanner.Scan() {
		mainText := providers.GetMainText(scanner.Text(), params.Provider, input)
		if len(mainText) < 1 {
			continue
		}
		fullText += mainText
		formatter.writeText(mainText)
	}

	// Flush any buffered non-tag content left in the XML buffer
	if formatter.inXMLTag && formatter.xmlBuffer.Len() > 0 {
		formatter.flushXMLBuffer()
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Some error has occurred. Error:", err)
		return ""
	}

	return fullText
}

func printConnectionErrorMsg(err error) {
	bold.Fprintln(os.Stderr, "\rSome error has occurred. Check your internet connection.")
	fmt.Fprintln(os.Stderr, "\nError:", err)
	os.Exit(1)
}

func handleStatus400(resp *http.Response) {
	bold.Fprintln(os.Stderr, "\rSome error has occurred. Statuscode:", resp.StatusCode)
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Println(string(respBody))
	os.Exit(1)
}

func ExecuteCommand(shellName string, shellOptions []string, fullLine string) string {
	return ExecuteCommandWithCapture(shellName, shellOptions, fullLine, false, false)
}

func ExecuteCommandWithAlias(shellName string, shellOptions []string, fullLine string) string {
	return ExecuteCommandWithCapture(shellName, shellOptions, fullLine, false, true)
}

func ExecuteCommandWithCapture(shellName string, shellOptions []string, fullLine string, captureOutput bool, useAliases bool) string {
	if runtime.GOOS != "windows" {
		rawModeOff := exec.Command("stty", "-raw", "echo")
		rawModeOff.Stdin = os.Stdin
		_ = rawModeOff.Run()
	}

	var cmd *exec.Cmd
	if useAliases && runtime.GOOS != "windows" && ShellConfigFile != "" {
		if _, err := os.Stat(ShellConfigFile); err == nil {
			quotedCfg := "'" + strings.ReplaceAll(ShellConfigFile, "'", `'\''`) + "'"
			sourceCmd := fmt.Sprintf("source %s && %s", quotedCfg, fullLine)
			cmd = exec.Command(shellName, shellOptions[0], sourceCmd)
		} else {
			cmd = exec.Command(shellName, append(shellOptions, fullLine)...)
		}
	} else {
		cmd = exec.Command(shellName, append(shellOptions, fullLine)...)
	}

	var result string
	if captureOutput {
		output, err := cmd.CombinedOutput()
		if err != nil {
			result = fmt.Sprintf("Command failed with error: %v\nOutput: %s", err, string(output))
		} else {
			result = string(output)
		}
		fmt.Print(result)
	} else {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	AddToShellHistory(fullLine)
	return result
}

func AddToShellHistory(command string) {
	shell := os.Getenv("SHELL")
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		return
	}

	var historyPath string
	var prefix string

	shellBase := filepath.Base(shell)

	switch {
	case shellBase == "bash":
		historyPath = os.Getenv("HISTFILE")
		if historyPath == "" {
			historyPath = homeDir + "/.bash_history"
		}
		prefix = ""
	case shellBase == "zsh":
		historyPath = os.Getenv("HISTFILE")
		if historyPath == "" {
			historyPath = homeDir + "/.zsh_history"
		}
		prefix = fmt.Sprintf(": %d:0;", time.Now().Unix())
	default:
		return
	}

	file, err := os.OpenFile(historyPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.WriteString(prefix + command + "\n")
}

func GetToolsSystemPrompt() string {
	SetShellAndOSVars()
	today := time.Now().Format("2006-01-02")
	return fmt.Sprintf(
		"You are tgpt, a terminal assistant. Today is %s. "+
			"The shell environment you are in is %s. The operating system you are on is %s.",
		today, ShellName, OperatingSystem,
	)
}

func MakeRequestAndGetData(input string, params structs.Params, extraOptions structs.ExtraOptions) (string, []interface{}, error) {
	if extraOptions.ToolDepth >= 5 {
		params.Tools = nil
	}

	if len(params.Tools) > 0 && params.SystemPrompt == "" {
		params.SystemPrompt = GetToolsSystemPrompt()
	}

	providersToTry := providersForRotation(params)

	isInteractive := extraOptions.IsInteractive || extraOptions.IsInteractiveShell || extraOptions.IsInteractiveFind

	// In interactive mode, try the last successful provider first
	if isInteractive && lastSuccessfulProvider != "" {
		for i, p := range providersToTry {
			if p == lastSuccessfulProvider {
				providersToTry = append(providersToTry[i:], providersToTry[:i]...)
				break
			}
		}
	}

	originalModel := params.ApiModel

	for i, provider := range providersToTry {
		params.Provider = provider
		params.ApiModel = originalModel

		key := strings.ToUpper(provider)
		if alias := os.Getenv("MODEL_ALIAS_" + key); alias != "" {
			params.ApiModel = alias
		}

		showStatus(statusEnabled(extraOptions), "Loading")

		resp, err := providers.NewRequest(input, params, extraOptions)
		if err != nil {
			hideStatus()

			if resp != nil {
				resp.Body.Close()
			}
			if i < len(providersToTry)-1 {
				fmt.Fprintf(os.Stderr, "\rProvider %s failed: %v\n", provider, err)
				continue
			}
			printConnectionErrorMsg(err)
		}

		code := resp.StatusCode
		hasMore := i < len(providersToTry)-1

		if code >= 400 {
			hideStatus()
			if hasMore {
				resp.Body.Close()
				fmt.Fprintf(os.Stderr, "\rProvider %s failed (status %d)\n", provider, code)
				continue
			}
			if !isInteractive {
				handleStatus400(resp)
			}
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			fmt.Fprintln(os.Stderr, "Some error has occurred, try again")
			fmt.Fprintln(os.Stderr, string(respBody))

			return "", nil, fmt.Errorf("provider %s returned status %d", provider, code)
		}

		hideStatus()

		if isInteractive {
			lastSuccessfulProvider = provider
		}

		if i > 0 {
			fmt.Printf("Fell back to \033[1m%s\033[0m\n", provider)
		}

		// --- Normal path (formatted output) ---
		if extraOptions.IsNormal {
			if extraOptions.IsToolFollowUp {
				// no header
			} else if !isInteractive {
				fmt.Print("\r          \r")
				bold.Println()
			} else {
				fmt.Println()
				boldViolet.Println("╭─ Bot")
			}

			if extraOptions.IsInteractiveShell || extraOptions.IsInteractiveFind {
				result := HandleEachPartInteractiveShell(resp, input, params)
				resp.Body.Close()
				return result, nil, nil
			}
			resultText, resultMessages := HandleEachPart(resp, input, params, extraOptions)
			resp.Body.Close()
			return resultText, resultMessages, nil
		}

		// --- Non-normal path (raw streaming) ---
		if extraOptions.IsGetCommand {
			fmt.Print("\r          \r")
		}

		scanner := bufio.NewScanner(resp.Body)
		fullText := ""

		for scanner.Scan() {
			mainText := providers.GetMainText(scanner.Text(), params.Provider, input)
			if len(mainText) < 1 {
				continue
			}
			fullText += mainText

			if !extraOptions.IsGetWhole {
				fmt.Print(mainText)
			}
		}

		resp.Body.Close()

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "Some error has occurred. Error:", err)
			return "", nil, err
		}

		if extraOptions.IsGetWhole {
			fmt.Println(fullText)
		}

		if extraOptions.IsGetSilent || extraOptions.IsGetCode || extraOptions.IsGetCommand {
			fmt.Println()
		}

		if extraOptions.IsGetCommand {
			lineCount := strings.Count(fullText, "\n") + 1
			if lineCount == 1 {
				if extraOptions.AutoExec {
					ExecuteCommand(ShellName, ShellOptions, fullText)
				} else {
					confirmed, err := bubbletea.ConfirmMenu("\nExecute shell command?", true)
					if errors.Is(err, bubbletea.ErrInterrupted) {
						return "", nil, err
					}
					if confirmed {
						ExecuteCommand(ShellName, ShellOptions, fullText)
					} else {
						clipboard.CopyToClipboard(fullText)
					}
				}
			}
		}

		return fullText, nil, nil
	}

	return "", nil, nil
}

func providersForRotation(params structs.Params) []string {
	if params.RotateProviders == "" {
		return []string{params.Provider}
	}
	raw := strings.Split(params.RotateProviders, ",")
	seenRaw := make(map[string]bool, len(raw))
	list := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p == "" || seenRaw[p] {
			continue
		}
		if providers.IsValidProvider(p) {
			seenRaw[p] = true
			list = append(list, p)
		}
	}
	if len(list) == 0 {
		fmt.Fprintf(os.Stderr, "\rWarning: all rotation providers are invalid, falling back to %s\n", params.Provider)
		return []string{params.Provider}
	}

	// Deduplicate all providers while preserving order, prepending
	// the primary provider as the first attempt so rotation entries
	// act as true fallbacks.
	deduped := make([]string, 0, len(list)+1)
	seen := make(map[string]struct{}, len(list)+1)
	if params.Provider != "" && providers.IsValidProvider(params.Provider) {
		deduped = append(deduped, params.Provider)
		seen[params.Provider] = struct{}{}
	}
	for _, p := range list {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		deduped = append(deduped, p)
	}
	return deduped
}

func ShowHelpMessage() {
	boldBlue.Println(`Usage: tgpt [Flags] [Prompt]`)

	boldBlue.Println("\nFlags:")
	fmt.Printf("%-50v Generate and Execute shell commands. \n", "-s, --shell")
	fmt.Printf("%-50v Generate Code.\n", "-c, --code")
	fmt.Printf("%-50v Gives response back without loading animation and extra text\n", "-q, --quiet")
	fmt.Printf("%-50v Gives response back as a whole text instead of streaming it\n", "-w, --whole")
	fmt.Printf("%-50v Generate images from text\n", "-img, --image")
	fmt.Printf("%-50v Set Provider. Detailed information has been provided below. (Env: AI_PROVIDER for chat and IMG_PROVIDER for image gen.)\n", "--provider")
	fmt.Printf("%-50v Find information using web search \n", "-f, --find")
	fmt.Printf("%-50v Search provider for web search: exa (default) or google (Env: SEARCH_PROVIDER).\n%-50v Exa works without api key with rate limits and supports EXA_API_KEY env variable.\n%-50v google requires TGPT_GOOGLE_API_KEY and TGPT_GOOGLE_SEARCH_ENGINE_ID env variables.\n%-50s Check SEARCH_SETUP.md for google: https://github.com/aandrew-me/tgpt/blob/main/SEARCH_SETUP.md\n", "--search-provider", "", "", "")
	fmt.Printf("%-50v Enable built-in tool calling (all or comma-separated list: %s)\n", "-t, --tools [tools]", strings.Join(tools.AllBuiltinTools, ", "))
	fmt.Printf("%-50v Enable MCP (Model Context Protocol) and auto-detect configuration file\n", "--mcp")
	fmt.Printf("%-50v Path to MCP server configuration JSON file (Env: MCP_CONFIG). See 'Tool calling & MCP' section below.\n", "--mcp-config")
	fmt.Printf("%-50v Command to run a stdio MCP server directly, e.g. --mcp-server \"npx -y some-mcp-server\"\n", "--mcp-server")
	fmt.Printf("%-50v Interactively configure and test a new MCP server in mcp_config.json\n", "--mcp-add")
	fmt.Printf("%-50v Interactively remove an existing MCP server from mcp_config.json\n", "--mcp-remove")

	boldBlue.Println("\nSome additional options can be set. However not all options are supported by all providers. Not supported options will just be ignored.")
	fmt.Printf("%-50v Set Model\n", "--model")
	fmt.Printf("%-50v Set API Key. (Env: AI_API_KEY)\n", "--key")
	fmt.Printf("%-50v Set API endpoint url. You need to provide the full URL. Supported by openai, opencode, openrouter, ollama, litellm, groq, gemini, deepseek, omniroute, atlascloud\n", "--url")
	fmt.Printf("%-50v Set filepath to log conversation to (For interactive modes)\n", "--log")
	fmt.Printf("%-50v Set preprompt\n", "--preprompt")
	fmt.Printf("%-50v Comma-separated fallback providers (Env: AI_ROTATE_PROVIDERS)\n", "--rotate")
	fmt.Printf("%-50v Execute shell command without confirmation\n", "-y")

	boldBlue.Println("\nOptions supported for image generation (with -image flag)")
	fmt.Printf("%-50v Output image filename (Supported by pollinations)\n", "--out")
	fmt.Printf("%-50v Output image height (Supported by pollinations)\n", "--height")
	fmt.Printf("%-50v Output image width (Supported by pollinations)\n", "--width")

	boldBlue.Println("\nOptions:")
	fmt.Printf("%-50v Print version \n", "-v, --version")
	fmt.Printf("%-50v Print help message \n", "-h, --help")
	fmt.Printf("%-50v Start normal interactive mode \n", "-i, --interactive")
	fmt.Printf("%-50v Start multi-line interactive mode \n", "-m, --multiline")
	fmt.Printf("%-50v Start interactive shell mode. (Doesn't work with all providers) \n", "-is, --interactive-shell")
	fmt.Printf("%-50v Interactive find mode with web search \n", "-if, --interactive-find")
	fmt.Printf("%-50v Start interactive shell mode with aliases and functions \n", "-ia, --interactive-alias")
	fmt.Printf("%-50v See changelog of latest version \n", "-cl, --changelog")

	if runtime.GOOS != "windows" {
		fmt.Printf("%-50v Update program \n", "-u, --update")
	}

	boldBlue.Println("\nProviders:")
	fmt.Println("The default provider is opencode. The AI_PROVIDER environment variable can be used to specify a different provider.")
	fmt.Println("Available providers to use: anyapi, aihorde, aitopia, atlascloud, deepseek, deepseek-web, fx, gemini, groq, isou, koboldai, minimax, ollama, ollamacloud, omniroute, openai, openrouter, opencode, pollinations, powerbrain.")

	bold.Println("\nProvider: anyapi")
	fmt.Println("Multi-model API with 100k free anytokens per day. Recognizes ANYAPI_API_KEY and ANYAPI_MODEL env vars. Default model: openai/gpt-4o-mini. Supports chat and image generation. Docs: https://docs.anyapi.ai/")

	bold.Println("\nProvider: atlascloud")
	fmt.Println("OpenAI-compatible Atlas Cloud API. Recognizes ATLASCLOUD_API_KEY, ATLASCLOUD_MODEL, ATLASCLOUD_URL and ATLASCLOUD_BASE_URL env vars. Default model: qwen/qwen3.8-max. Docs: https://www.atlascloud.ai/")

	bold.Println("\nProvider: aihorde")
	fmt.Println("A free, community-powered generation service: volunteers share spare computer power so anyone can generate images and text. Supports AIHORDE_MODEL and AIHORDE_API_KEY env variables. Site: https://aihorde.net/")

	bold.Println("\nProvider: aitopia")
	fmt.Println("Free provider, uses gpt-4o-mini model.")

	bold.Println("\nProvider: deepseek")
	fmt.Println("Uses DeepSeek-V4-Flash model by default. Requires API key. Recognizes DEEPSEEK_API_KEY and DEEPSEEK_MODEL env vars. Docs: https://api-docs.deepseek.com/")

	bold.Println("\nProvider: deepseek-web")
	fmt.Println("Web-based DeepSeek provider using chat.deepseek.com. Requires userToken (via DEEPSEEK_WEB_TOKEN env var or --key) and a JS runtime (node, bun, or deno) in PATH for solving Proof-of-Work.")
	fmt.Println("How to get userToken:")
	fmt.Println("  1. Log in to https://chat.deepseek.com in your browser.")
	fmt.Println("  2. Open Developer Tools (F12) -> Application -> Local Storage -> https://chat.deepseek.com.")
	fmt.Println("  3. Find the key 'userToken' and copy its value (either the inner 'value' string or full JSON).")
	fmt.Println("Supports DEEPSEEK_WEB_THINKING=true (for deep thinking) and DEEPSEEK_WEB_SEARCH=true (for web search).")

	bold.Println("\nProvider: fx")
	fmt.Println("Free provider using fx.sh gateway. Default model: zai/glm-5.2. Site: https://fx.sh/")

	bold.Println("\nProvider: groq")
	fmt.Println("Requires a free API key. Recognizes GROQ_API_KEY and GROQ_MODEL env vars. Models: https://console.groq.com/docs/models")

	bold.Println("\nProvider: gemini")
	fmt.Println("Requires a free API key. Recognizes GEMINI_API_KEY and GEMINI_MODEL env vars. https://aistudio.google.com/apikey")

	bold.Println("\nProvider: isou")
	fmt.Println("Free provider with web search. Site: https://isou.chat/")

	bold.Println("\nProvider: koboldai")
	fmt.Println("Uses koboldcpp/HF_SPACE_Tiefighter-13B only, answers from novels")

	bold.Println("\nProvider: litellm")
	fmt.Println("Proxy/gateway provider. Recognizes LITELLM_API_KEY and LITELLM_MODEL env vars. Requires --model flag or LITELLM_MODEL env var. Supports custom URLs via LITELLM_URL or --url.")

	bold.Println("\nProvider: minimax")
	fmt.Println("Requires API key. Uses MiniMax-M2.7 model by default. Recognizes MINIMAX_API_KEY and MINIMAX_MODEL env vars. https://platform.minimaxi.com")

	bold.Println("\nProvider: ollama")
	fmt.Println("Needs to be run locally. Supports many models. ")

	bold.Println("\nProvider: ollamacloud")
	fmt.Println("Uses Ollama Cloud API. Recognizes OLLAMA_API_KEY and OLLAMA_MODEL env vars. Default model: gpt-oss:120b")

	bold.Println("\nProvider: omniroute")
	fmt.Println("OpenAI compatible provider. Default base URL: http://localhost:20128/v1. Default model: auto (supports auto/coding, auto/fast, auto/cheap). Recognizes OMNIROUTE_API_KEY, OMNIROUTE_MODEL, OMNIROUTE_URL, OMNIROUTE_BASE_URL env vars.")

	bold.Println("\nProvider: opencode")
	fmt.Println("Free provider using opencode.ai/zen API. Uses deepseek-v4-flash-free model by default. API key defaults to 'public'. Recognizes OPENCODE_API_KEY, OPENCODE_MODEL and OPENCODE_URL env vars. Available models: https://opencode.ai/docs/zen/#endpoints")

	bold.Println("\nProvider: openai")
	fmt.Println("Needs API key to work and supports various models. Recognizes OPENAI_API_KEY, CEREBRAS_API_KEY and OPENAI_MODEL env vars. Supports custom urls with --url")

	bold.Println("\nProvider: openrouter")
	fmt.Println("OpenAI-compatible OpenRouter API. Default model: openrouter/free. Recognizes OPENROUTER_API_KEY, OPENROUTER_MODEL, OPENROUTER_URL and OPENROUTER_BASE_URL env vars. Docs: https://openrouter.ai/")

	bold.Println("\nProvider: pollinations")
	fmt.Println("Works without an API key. Recognizes POLLINATIONS_API_KEY and POLLINATIONS_MODEL env vars. Free API key: https://enter.pollinations.ai/")

	bold.Println("\nProvider: powerbrain")
	fmt.Println("Free provider using powerbrainai.com API. Uses gpt-5 model by default. No API key required.")

	boldBlue.Println("\nImage generation providers:")

	bold.Println("\nProvider: magicstudio")
	fmt.Println("Free provider, very fast.")

	bold.Println("\nProvider: anyapi")
	fmt.Println("Requires API key via ANYAPI_API_KEY env var or --key. Supports many models including google/gemini-2.5-flash-image")

	bold.Println("\nProvider: pollinations")
	fmt.Println("Supported models: flux, turbo")

	boldBlue.Println("\nTool calling & MCP:")
	fmt.Println("tgpt can let the model call tools (functions) while answering, and can also connect to Model Context Protocol (MCP) servers to add more tools at runtime.")
	fmt.Println("Not all providers support tool calling; unsupported providers will just ignore the tool definitions.")

	bold.Println("\nBuilt-in tools (enabled with -t / --tools [tools]):")
	fmt.Println("web_search_exa       Search the web using Exa (supports EXA_API_KEY env var)")
	fmt.Println("web_search_firecrawl Search the web using Firecrawl (supports FIRECRAWL_API_KEY env var)")
	fmt.Println("read_directory       List the contents of a directory")
	fmt.Println("read_file            Read content from a text file")
	fmt.Println("execute_command      Execute a shell command")
	fmt.Println("web_fetch            Fetch the contents of a webpage or URL")
	fmt.Println("write_file           Write content to a file")
	fmt.Println("edit_file            Edit a file by replacing old_content with new_content")
	fmt.Println("grep                 Search file contents using regular expressions")
	fmt.Println("glob                 Find files and directories matching a glob pattern")

	bold.Println("\nMCP servers:")
	fmt.Println("Use --mcp to enable MCP and auto-detect configuration file, --mcp-config to specify a JSON file, or --mcp-server to run a single stdio MCP server directly.")
	fmt.Println("Tools discovered from MCP servers are made available to the model for tool calling.")
	fmt.Println("\nSupported server fields in mcp_config.json:")
	fmt.Println("  • Stdio servers:     \"command\", \"args\" (array), \"env\" (array of KEY=VALUE strings)")
	fmt.Println("  • HTTP/SSE servers:  \"url\", \"type\" (\"streamable-http\"|\"sse\"), \"headers\" (map of key-value pairs)")
	fmt.Println("  • Authentication:    Pass Bearer tokens or API keys via \"headers\" (e.g., \"Authorization\": \"Bearer <key>\")")

	boldBlue.Println("\nExample MCP config file (mcp_config.json):")
	codeText.Println(`{`)
	codeText.Println(`  "mcpServers": {`)
	codeText.Println(`    "filesystem": {`)
	codeText.Println(`      "command": "npx",`)
	codeText.Println(`      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/dir"],`)
	codeText.Println(`      "env": ["LOG_LEVEL=info"]`)
	codeText.Println(`    },`)
	codeText.Println(`    "firecrawl": {`)
	codeText.Println(`      "url": "https://mcp.firecrawl.dev/v2/mcp",`)
	codeText.Println(`      "headers": {`)
	codeText.Println(`        "Authorization": "Bearer YOUR_FIRECRAWL_API_KEY"`)
	codeText.Println(`      }`)
	codeText.Println(`    }`)
	codeText.Println(`  }`)
	codeText.Println(`}`)

	boldBlue.Println("\nTool calling & MCP examples:")
	fmt.Println(`tgpt --mcp-add                                           # Interactively configure a new MCP server`)
	fmt.Println(`tgpt --mcp-remove                                        # Interactively remove an MCP server`)
	fmt.Println(`tgpt -t "What files are in the current directory?"`)
	fmt.Println(`tgpt -t web_search_exa,read_file "Search and read specified file"`)
	fmt.Println(`tgpt --mcp "Use MCP tools from auto-detected mcp_config.json"`)
	fmt.Println(`tgpt --mcp-server "npx -y @modelcontextprotocol/server-filesystem /path/to/dir" "List the files in /path/to/dir"`)
	fmt.Println(`tgpt --mcp-config mcp_config.json "Use the filesystem tool to read README.md"`)
	fmt.Println(`tgpt -t --mcp-config mcp_config.json "Use both built-in tools and MCP tools"`)

	boldBlue.Println("\nConfiguration file")
	userProfileEnv := "%USERPROFILE%"
	fmt.Println("You can create a configuration file - config.conf in ~/.config/tgpt (" + userProfileEnv + "\\.config\\tgpt on Windows) or in current directory (has higher priority). The configuration file supports all the environment variables supported by tgpt.")
	boldBlue.Println("\nExample config file:")
	codeText.Println("EXA_API_KEY=sk_xxxxx")
	codeText.Println("FIRECRAWL_API_KEY=fc_xxxxx")
	codeText.Println("POLLINATIONS_API_KEY=sk_xxxxx")
	codeText.Println("ANYAPI_API_KEY=sk_xxxxxxxx")
	codeText.Println("GROQ_API_KEY=gsk_xxxx")
	codeText.Println("HTTP_PROXY=http://localhost:8080")
	codeText.Println("AI_PROVIDER=groq")
	codeText.Println("OPENCODE_MODEL=mimo-v2.5-free")
	codeText.Println("AI_ROTATE_PROVIDERS=deepseek,groq,anyapi,opencode")

	boldBlue.Println("\nExamples:")
	fmt.Println(`tgpt "What is internet?"`)
	fmt.Println(`tgpt -m`)
	fmt.Println(`tgpt -s "How to update my system?"`)
	fmt.Println(`tgpt -c "Write a function in Go that reverses a string"`)
	fmt.Println(`tgpt --provider deepseek "What is 1+1"`)
	fmt.Println(`tgpt --provider deepseek-web --key "YOUR_USER_TOKEN" "What is 1+1"`)
	fmt.Println(`tgpt --img "cat"`)
	fmt.Println(`tgpt --img --out ~/my-cat.jpg --height 256 --width 256 "cat"`)
	fmt.Println(`tgpt --provider openai --key "sk-xxxx" --model "gpt-5.6" "What is 1+1"`)
	fmt.Println(`cat install.sh | tgpt "Explain the code"`)
}

func SearchQuery(input string, params structs.Params, extraOptions structs.ExtraOptions, isQuiet bool, logFile string) {
	if extraOptions.Verbose {
		fmt.Printf("DEBUG: searchQuery called with input: %s\n", input)
	}

	// For one-shot find mode (-f), skip confirmation. For interactive find mode (-if), show confirmation
	skipConfirmation := extraOptions.IsFind && !extraOptions.IsInteractiveFind
	searchResults, err := search.ProcessSearchWithConfirmation(input, params, extraOptions.Verbose, skipConfirmation, isQuiet, nil, extraOptions.SearchProvider)
	if err != nil {
		fmt.Printf("Search failed: %v\n", err)
		return
	}

	if searchResults == "Search cancelled by user." {
		fmt.Println(searchResults)
		return
	}

	if len(logFile) > 0 {
		utils.LogToFile(input, "SEARCH_QUERY", logFile)
	}

	// Use IsNormal so the response text is returned (and printed by HandleEachPart)
	searchOptions := structs.ExtraOptions{
		IsNormal:    !isQuiet,
		IsGetSilent: isQuiet,
		AutoExec:    extraOptions.AutoExec,
	}

	queryWithContext := fmt.Sprintf("Here is the output of the search results: %s\n\nBased on these search results, answer the user's question: %s", searchResults, input)

	response, _, _ := MakeRequestAndGetData(queryWithContext, params, searchOptions)

	if len(logFile) > 0 {
		utils.LogToFile(response, "SEARCH_RESPONSE", logFile)
	}
}

func InteractiveFindSession(params structs.Params, extraOptions structs.ExtraOptions, logFile string, inputReader func() (string, error)) func(string) {
	var previousMessages []any

	threadID := utils.RandomString(36)

	promptFind := "You are an intelligent search assistant. When a user asks a question that requires current information, web search, or factual lookup, " +
		"wrap your search intent in XML tags like <search>search query here</search>. " +
		"For follow-up questions about previous search results, you can reference the context. " +
		"For general conversation that doesn't need search, respond normally without search tags. " +
		"Examples:\n" +
		"User: What's the weather like in Paris today?\n" +
		"Assistant: <search>current weather Paris France today</search>\n" +
		"User: Tell me more about the first result\n" +
		"Assistant: Based on the previous search results, [provide more detail from context]\n" +
		"User: How are you?\n" +
		"Assistant: I'm doing well, thank you! How can I help you with finding information today?"

	searchRegex := regexp.MustCompile(`<search>(.*?)</search>`)

	getAndPrintFindResponse := func(input string) {
		input = strings.TrimSpace(input)
		if len(input) <= 1 {
			return
		}
		if input == "exit" {
			bold.Println("Exiting...")
			os.Exit(0)
		}

		if len(logFile) > 0 {
			utils.LogToFile(input, "USER_QUERY", logFile)
		}

		// Use preprompt for first message
		if len(previousMessages) == 0 && len(params.Preprompt) > 0 {
			input = params.Preprompt + input
		}

		// Set up conversation context
		params.PrevMessages = previousMessages
		params.ThreadID = threadID
		params.SystemPrompt = promptFind

		responseObjects, responseTxt := GetData(input, params, structs.ExtraOptions{
			IsInteractiveFind: true,
			IsGetSilent:       true,
			AutoExec:          extraOptions.AutoExec,
		})
		matches := searchRegex.FindStringSubmatch(responseTxt)

		if len(matches) > 1 {
			// Search intent detected
			searchQuery := strings.TrimSpace(matches[1])
			if extraOptions.Verbose {
				fmt.Printf("DEBUG: Search intent detected: '%s'\n", searchQuery)
			}

			searchResults, err := search.ProcessSearchWithConfirmation(searchQuery, params, extraOptions.Verbose, false, false, inputReader, extraOptions.SearchProvider)
			if err != nil {
				fmt.Printf("Search failed: %v\n", err)
				return
			}

			if searchResults == "Search cancelled by user." {
				fmt.Println(searchResults)
				return
			}

			previousMessages = append(previousMessages, responseObjects...)

			searchContextMsg := structs.DefaultMessage{
				Role:    "system",
				Content: fmt.Sprintf("Search results for '%s':\n%s", searchQuery, searchResults),
			}
			previousMessages = append(previousMessages, searchContextMsg)

			params.PrevMessages = previousMessages
			finalResponseObjects, finalResponseTxt := GetData(
				fmt.Sprintf("Based on these search results, answer the user's question: %s", input),
				params,
				structs.ExtraOptions{IsInteractiveFind: true, IsNormal: true, AutoExec: extraOptions.AutoExec},
			)

			if len(logFile) > 0 {
				utils.LogToFile(finalResponseTxt, "ASSISTANT_RESPONSE", logFile)
			}

			previousMessages = append(previousMessages, finalResponseObjects...)
		} else {
			fmt.Println()
			boldViolet.Println("╭─ Bot")

			formatter := newStreamFormatter(params.Provider)
			formatter.writeText(responseTxt)
			fmt.Print("\n\n")

			if len(logFile) > 0 {
				utils.LogToFile(responseTxt, "ASSISTANT_RESPONSE", logFile)
			}

			previousMessages = append(previousMessages, responseObjects...)
		}
	}

	return getAndPrintFindResponse
}
