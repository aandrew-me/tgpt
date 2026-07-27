package helper

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/clipboard"
	"github.com/aandrew-me/tgpt/v2/src/providers"
	"github.com/aandrew-me/tgpt/v2/src/search"
	"github.com/aandrew-me/tgpt/v2/src/structs"
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
	responseTxt, err := MakeRequestAndGetData(input, params, extraOptions)
	if err != nil {
		return nil, ""
	}

	if !extraOptions.IsGetSilent {
		fmt.Print("\n\n")
	}

	var msgObjectNew []interface{}

	msgObjectNew = []interface{}{
		structs.DefaultMessage{Content: input, Role: "user"},
		structs.DefaultMessage{Content: responseTxt, Role: "assistant"},
	}

	return msgObjectNew, responseTxt
}

func Loading(stop *atomic.Bool) {
	spinChars := []string{"⣾ ", "⣽ ", "⣻ ", "⢿ ", "⡿ ", "⣟ ", "⣯ ", "⣷ "}
	i := 0
	for !stop.Load() {
		fmt.Printf("\r%s Loading", spinChars[i])
		i = (i + 1) % len(spinChars)
		time.Sleep(80 * time.Millisecond)
	}
	fmt.Print("\r           \r")
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

func Update(localVersion string, executablePath string) {
	if runtime.GOOS == "windows" || runtime.GOOS == "android" {
		fmt.Println("This feature is not supported on your Operating System")
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
	cmd := exec.Command("bash", "-c", script, "bash", executablePath)
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
			"Request:%s\nCode:", input)

	_, _ = MakeRequestAndGetData(codePrompt, params, extraOptions)
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
		ShellName, OperatingSystem, input)
	GetCommand(shellPrompt, params, extraOptions)
}

func GetCommand(shellPrompt string, params structs.Params, extraOptions structs.ExtraOptions) {
	_, _ = MakeRequestAndGetData(shellPrompt, params, extraOptions)
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
	_, _ = MakeRequestAndGetData(input, params, extraOptions)
}

func GetLastCodeBlock(markdown string) string {
	lines := strings.Split(markdown, "\n")
	var codeBlock []string
	capturing := false

	for i := len(lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(lines[i], "```") {
			if capturing {
				capturing = false
				break
			}
			capturing = true
			continue
		}
		if capturing {
			codeBlock = append([]string{lines[i]}, codeBlock...)
		}
	}

	if capturing || len(codeBlock) == 0 {
		return ""
	}

	return strings.Join(codeBlock, "\n")
}

func HandleEachPart(resp *http.Response, input string, params structs.Params) string {
	scanner := bufio.NewScanner(resp.Body)
	formatter := newStreamFormatter(params.Provider)
	fullText := ""

	for scanner.Scan() {
		mainText := providers.GetMainText(scanner.Text(), params.Provider, input)
		if len(mainText) < 1 {
			continue
		}
		fullText += mainText
		formatter.writeText(mainText)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Some error has occurred. Error:", err)
		os.Exit(1)
	}

	return fullText
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

	file, err := os.OpenFile(historyPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.WriteString(prefix + command + "\n")
}

func MakeRequestAndGetData(input string, params structs.Params, extraOptions structs.ExtraOptions) (string, error) {
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
	var stopSpin atomic.Bool
	showSpinner := !extraOptions.IsGetSilent && !extraOptions.IsGetWhole && !isInteractive

	for i, provider := range providersToTry {
		params.Provider = provider
		params.ApiModel = originalModel

		key := strings.ToUpper(provider)
		if alias := os.Getenv("MODEL_ALIAS_" + key); alias != "" {
			params.ApiModel = alias
		}

		if showSpinner {
			go Loading(&stopSpin)
		}

		resp, err := providers.NewRequest(input, params, extraOptions)

		if err != nil {
			stopSpin.Store(true)

			if resp != nil {
				resp.Body.Close()
			}
			if i < len(providersToTry)-1 {
				fmt.Fprintf(os.Stderr, "\rProvider %s failed: %v\n", provider, err)
				continue
			}
			printConnectionErrorMsg(err)
		}

		if resp == nil {
			stopSpin.Store(true)
			continue
		}

		code := resp.StatusCode
		hasMore := i < len(providersToTry)-1

		if code >= 400 {
			stopSpin.Store(true)
			fmt.Print("\r")
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

			return "", fmt.Errorf("provider %s returned status %d", provider, code)
		}

		stopSpin.Store(true)
		fmt.Print("\r")

		if isInteractive {
			lastSuccessfulProvider = provider
		}

		if i > 0 {
			fmt.Printf("Fell back to \033[1m%s\033[0m\n", provider)
		}

		// --- Normal path (formatted output) ---
		if extraOptions.IsNormal {
			if !isInteractive {
				fmt.Print("\r          \r")
				bold.Println()
			} else {
				fmt.Println()
				boldViolet.Println("╭─ Bot")
			}

			if extraOptions.IsInteractiveShell || extraOptions.IsInteractiveFind {
				result := HandleEachPartInteractiveShell(resp, input, params)
				resp.Body.Close()
				return result, nil
			}
			result := HandleEachPart(resp, input, params)
			resp.Body.Close()
			return result, nil
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

			if !extraOptions.IsGetWhole && !extraOptions.IsGetSilent {
				fmt.Print(mainText)
			}
		}

		resp.Body.Close()

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "Some error has occurred. Error:", err)
			return "", err
		}

		if extraOptions.IsGetWhole {
			fmt.Println(fullText)
		}

		if extraOptions.IsGetSilent || extraOptions.IsGetCode {
			fmt.Println()
		}

		if extraOptions.IsGetCommand {
			lineCount := strings.Count(fullText, "\n") + 1
			if lineCount == 1 {
				if extraOptions.AutoExec {
					fmt.Println()
					ExecuteCommand(ShellName, ShellOptions, fullText)
				} else {
					bold.Print("\n\nExecute shell command? [y/n]: ")
					reader := bufio.NewReader(os.Stdin)
					userInput, _ := reader.ReadString('\n')
					userInput = strings.TrimSpace(userInput)
					if userInput == "y" || userInput == "" {
						ExecuteCommand(ShellName, ShellOptions, fullText)
					} else {
						clipboard.CopyToClipboard(fullText)
					}
				}
			}
		}

		return fullText, nil
	}

	return "", nil
}

func providersForRotation(params structs.Params) []string {
	if params.RotateProviders == "" {
		return []string{params.Provider}
	}
	raw := strings.Split(params.RotateProviders, ",")
	seen := make(map[string]bool, len(raw))
	list := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		if providers.IsValidProvider(p) {
			seen[p] = true
			list = append(list, p)
		}
	}
	if len(list) == 0 {
		list = []string{params.Provider}
	}
	return list
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

	boldBlue.Println("\nSome additional options can be set. However not all options are supported by all providers. Not supported options will just be ignored.")
	fmt.Printf("%-50v Set Model\n", "--model")
	fmt.Printf("%-50v Set API Key. (Env: AI_API_KEY)\n", "--key")
	fmt.Printf("%-50v Set API endpoint url. You need to provide the full URL. Supported by openai, opencode, ollama, litellm, groq, gemini, deepseek\n", "--url")
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
	fmt.Println("Available providers to use: anyapi, deepseek, gemini, groq, isou, koboldai, minimax, ollama, ollamacloud, openai, opencode, pollinations, powerbrain.")

	bold.Println("\nProvider: anyapi")
	fmt.Println("Multi-model API with 100k free anytokens per day. Recognizes ANYAPI_API_KEY and ANYAPI_MODEL env vars. Default model: openai/gpt-4o-mini. Supports chat and image generation. Docs: https://docs.anyapi.ai/")

	bold.Println("\nProvider: deepseek")
	fmt.Println("Uses DeepSeek-V4-Flash model by default. Requires API key. Recognizes DEEPSEEK_API_KEY and DEEPSEEK_MODEL env vars. Docs: https://api-docs.deepseek.com/")

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

	bold.Println("\nProvider: opencode")
	fmt.Println("Free provider using opencode.ai/zen API. Uses deepseek-v4-flash-free model by default. API key defaults to 'public'. Recognizes OPENCODE_API_KEY, OPENCODE_MODEL and OPENCODE_URL env vars. Available models: https://opencode.ai/docs/zen/#endpoints")

	bold.Println("\nProvider: openai")
	fmt.Println("Needs API key to work and supports various models. Recognizes OPENAI_API_KEY, CEREBRAS_API_KEY and OPENAI_MODEL env vars. Supports custom urls with --url")

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

	boldBlue.Println("\nConfiguration file")
	fmt.Println(`You can create a configuration file - config.conf in ~/.config/tgpt (%USERPROFILE%\.config\tgpt on Windows) or in current directory (has higher priority). The configuration file supports all the environment variables supported by tgpt.`)
	boldBlue.Println("\nExample config file:")
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
	}

    queryWithContext := fmt.Sprintf("Here is the output of the search results: %s\n\nBased on these search results, answer the user's question: %s", searchResults, input)

	response, _ := MakeRequestAndGetData(queryWithContext, params, searchOptions)

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
				structs.ExtraOptions{IsInteractiveFind: true, IsNormal: true},
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
