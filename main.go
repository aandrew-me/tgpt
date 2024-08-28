package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/PuerkitoBio/goquery"
	"github.com/aandrew-me/tgpt/v2/structs"
	"github.com/aandrew-me/tgpt/v2/utils"
	"github.com/atotto/clipboard"
	Prompt "github.com/c-bata/go-prompt"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
	"github.com/olekukonko/ts"
)

const localVersion = "2.8.1"

var (
	bold                 = color.New(color.Bold)
	boldBlue             = color.New(color.Bold, color.FgBlue)
	blue                 = color.New(color.FgBlue)
	boldViolet           = color.New(color.Bold, color.FgMagenta)
	codeText             = color.New(color.BgBlack, color.FgGreen, color.Bold)
	stopSpin             = false
	programLoop          = true
	userInput            = ""
	lastResponse         = ""
	executablePath       = ""
	provider             *string
	apiModel             *string
	apiKey               *string
	temperature          *string
	top_p                *string
	max_length           *string
	preprompt            *string
	urlChat              *string
	logFile              *string
	shouldExecuteCommand *bool
	disableInputLimit    *bool
	//ws                   *bool
)

type SearchResult struct {
	Title   string
	Link    string
	Snippet string
}

func main() {
	execPath, err := os.Executable()
	if err == nil {
		executablePath = execPath
	}
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-terminate
		os.Exit(0)
	}()

	args := os.Args

	apiModel = flag.String("model", "", "Choose which model to use")
	provider = flag.String("provider", os.Getenv("AI_PROVIDER"), "Choose which provider to use")
	apiKey = flag.String("key", "", "Use personal API Key")
	temperature = flag.String("temperature", "", "Set temperature")
	top_p = flag.String("top_p", "", "Set top_p")
	max_length = flag.String("max_length", "", "Set max length of response")
	preprompt = flag.String("preprompt", "", "Set preprompt")
	urlChat = flag.String("url", "https://api.openai.com/v1/chat/completions", "url for openai providers")
	logFile = flag.String("log", "", "Filepath to log conversation to.")
	shouldExecuteCommand = flag.Bool(("y"), false, "Instantly execute the shell command")

	isQuiet := flag.Bool("q", false, "Gives response back without loading animation")
	flag.BoolVar(isQuiet, "quiet", false, "Gives response back without loading animation")

	isWhole := flag.Bool("w", false, "Gives response back as a whole text")
	flag.BoolVar(isWhole, "whole", false, "Gives response back as a whole text")

	isCode := flag.Bool("c", false, "Generate Code. (Experimental)")
	flag.BoolVar(isCode, "code", false, "Generate Code. (Experimental)")

	isShell := flag.Bool("s", false, "Generate and Execute shell commands.")
	flag.BoolVar(isShell, "shell", false, "Generate and Execute shell commands.")

	isImage := flag.Bool("img", false, "Generate images from text")
	flag.BoolVar(isImage, "image", false, "Generate images from text")

	isInteractive := flag.Bool("i", false, "Start normal interactive mode")
	flag.BoolVar(isInteractive, "interactive", false, "Start normal interactive mode")

	isMultiline := flag.Bool("m", false, "Start multi-line interactive mode")
	flag.BoolVar(isMultiline, "multiline", false, "Start multi-line interactive mode")

	isVersion := flag.Bool("v", false, "Gives response back as a whole text")
	flag.BoolVar(isVersion, "version", false, "Gives response back as a whole text")

	isHelp := flag.Bool("h", false, "Gives response back as a whole text")
	flag.BoolVar(isHelp, "help", false, "Gives response back as a whole text")

	isUpdate := flag.Bool("u", false, "Update program")
	flag.BoolVar(isUpdate, "update", false, "Update program")

	isChangelog := flag.Bool("cl", false, "See changelog of versions")
	flag.BoolVar(isChangelog, "changelog", false, "See changelog of versions")

	disableInputLimit := flag.Bool("disable-input-limit", false, "Disables the checking of 4000 character input limit")

	isWebsearch := flag.Bool("ws", false, "Normal search using duckduckgo.")
	flag.BoolVar(isWebsearch, "websearch", false, "Normal search using duckduckgo.")

	flag.Parse()

	prompt := flag.Arg(0)

	pipedInput := ""
	cleanPipedInput := ""

	stat, err := os.Stdin.Stat()
	if err != nil {
		fmt.Fprintln(os.Stderr, "accessing standard input:", err)
		os.Exit(1)
	}

	// Checking for piped text
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			pipedInput += scanner.Text()
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
			os.Exit(1)
		}
	}

	if len(pipedInput) > 0 {
		cleanPipedInputByte, err := json.Marshal(pipedInput)
		if err != nil {
			fmt.Fprintln(os.Stderr, "marshaling piped input to JSON:", err)
			os.Exit(1)
		}
		cleanPipedInput = string(cleanPipedInputByte)
		cleanPipedInput = cleanPipedInput[1 : len(cleanPipedInput)-1]

		safePipedBytes, err := json.Marshal(pipedInput + "\n")
		if err != nil {
			fmt.Fprintln(os.Stderr, "marshaling piped input to JSON:", err)
			os.Exit(1)
		}
		pipedInput = string(safePipedBytes)
		pipedInput = pipedInput[1 : len(pipedInput)-1]
	}

	contextTextByte, _ := json.Marshal("\n\nHere is text for the context:\n")
	contextText := string(contextTextByte)

	if len(args) > 1 {
		switch {

		case *isVersion:
			fmt.Println("tgpt", localVersion)
		case *isChangelog:
			getVersionHistory()
		case *isWhole:
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					fmt.Fprintln(os.Stderr, "You need to provide some text")
					fmt.Fprintln(os.Stderr, `Example: tgpt -w "What is encryption?"`)
					os.Exit(1)
				}
				getWholeText(*preprompt + trimmedPrompt + contextText + pipedInput)
			} else {
				formattedInput := getFormattedInputStdin()
				fmt.Println()
				getWholeText(*preprompt + formattedInput + cleanPipedInput)
			}
		case *isQuiet:
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					fmt.Fprintln(os.Stderr, "You need to provide some text")
					fmt.Fprintln(os.Stderr, `Example: tgpt -q "What is encryption?"`)
					os.Exit(1)
				}
				getSilentText(*preprompt + trimmedPrompt + contextText + pipedInput)
			} else {
				formattedInput := getFormattedInputStdin()
				fmt.Println()
				getSilentText(*preprompt + formattedInput + cleanPipedInput)
			}
		case *isShell:
			if len(prompt) > 1 {
				go loading(&stopSpin)
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					fmt.Fprintln(os.Stderr, "You need to provide some text")
					fmt.Fprintln(os.Stderr, `Example: tgpt -s "How to update system"`)
					os.Exit(1)
				}
				shellCommand(*preprompt + trimmedPrompt + contextText + pipedInput)
			} else {
				fmt.Fprintln(os.Stderr, "You need to provide some text")
				fmt.Fprintln(os.Stderr, `Example: tgpt -s "How to update system"`)
				os.Exit(1)
			}

		case *isCode:
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					fmt.Fprintln(os.Stderr, "You need to provide some text")
					fmt.Fprintln(os.Stderr, `Example: tgpt -c "Hello world in Python"`)
					os.Exit(1)
				}
				codeGenerate(*preprompt + trimmedPrompt + contextText + pipedInput)
			} else {
				fmt.Fprintln(os.Stderr, "You need to provide some text")
				fmt.Fprintln(os.Stderr, `Example: tgpt -c "Hello world in Python"`)
				os.Exit(1)
			}
		case *isUpdate:
			update()
		case *isInteractive:
			/////////////////////
			// Normal interactive
			/////////////////////

			bold.Print("Interactive mode started. Press Ctrl + C or type exit to quit.\n\n")

			previousMessages := ""
			threadID := utils.RandomString(36)
			history := []string{}

			for {
				blue.Println("╭─ You")
				input := Prompt.Input("╰─> ", historyCompleter,
					Prompt.OptionHistory(history),
					Prompt.OptionPrefixTextColor(Prompt.Blue),
					Prompt.OptionAddKeyBind(Prompt.KeyBind{
						Key: Prompt.ControlC,
						Fn:  exit,
					}),
				)

				if len(input) > 1 {
					input = strings.TrimSpace(input)
					if len(input) > 1 {
						if input == "exit" {
							bold.Println("Exiting...")
							if runtime.GOOS != "windows" {
								rawModeOff := exec.Command("stty", "-raw", "echo")
								rawModeOff.Stdin = os.Stdin
								_ = rawModeOff.Run()
								rawModeOff.Wait()
							}
							os.Exit(0)
						}
						if len(*logFile) > 0 {
							utils.LogToFile(input, "USER_QUERY", *logFile)
						}
						// Use preprompt for first message
						if previousMessages == "" {
							input = *preprompt + input
						}
						responseJson, responseTxt := getData(input, structs.Params{
							PrevMessages: previousMessages,
							ThreadID:     threadID,
							Provider:     *provider,
						}, structs.ExtraOptions{IsInteractive: true, DisableInputLimit: *disableInputLimit, IsNormal: true})
						if len(*logFile) > 0 {
							utils.LogToFile(responseTxt, "ASSISTANT_RESPONSE", *logFile)
						}
						// Check for LaTeX in the response
						if hasLatex(responseTxt) {
							fmt.Println("LaTeX detected in response. Rendering in web browser...")
							renderLaTeXInBrowser(responseTxt)
						} else {
							fmt.Println(responseTxt)
						}
						previousMessages += responseJson
						history = append(history, input)
						lastResponse = responseTxt
					}
				}
			}

		case *isMultiline:
			/////////////////////
			// Multiline interactive
			/////////////////////

			fmt.Print("\nPress Ctrl + D to submit, Ctrl + C to exit, Esc to unfocus, i to focus. When unfocused, press p to paste, c to copy response, b to copy last code block in response\n")

			previousMessages := ""
			threadID := utils.RandomString(36)

			for programLoop {
				fmt.Print("\n")
				p := tea.NewProgram(initialModel())
				_, err := p.Run()
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				if len(userInput) > 0 {
					if len(*logFile) > 0 {
						utils.LogToFile(userInput, "USER_QUERY", *logFile)
					}

					responseJson, responseTxt := getData(userInput, structs.Params{
						PrevMessages: previousMessages,
						Provider:     *provider,
						ThreadID:     threadID,
					}, structs.ExtraOptions{IsInteractive: true, DisableInputLimit: *disableInputLimit, IsNormal: true})
					previousMessages += responseJson
					lastResponse = responseTxt

					if len(*logFile) > 0 {
						utils.LogToFile(responseTxt, "ASSISTANT_RESPONSE", *logFile)
					}
				}

			}

		case *isImage:
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					fmt.Fprintln(os.Stderr, "You need to provide some text")
					fmt.Fprintln(os.Stderr, `Example: tgpt -img "cat"`)
					os.Exit(1)
				}
				generateImage(trimmedPrompt)
			} else {
				formattedInput := getFormattedInputStdin()
				fmt.Println()
				generateImage(*preprompt + formattedInput)
			}
		case *isHelp:
			showHelpMessage()

		case *isWebsearch:
			if len(prompt) < 1 {
				fmt.Println("Please provide a search query")
				os.Exit(1)
			}

			reader := bufio.NewReader(os.Stdin)
			var query string

			// Check if a command-line argument was provided
			if len(os.Args) > 1 {
				query = strings.Join(os.Args[1:], " ")
			} else {
				// If no argument provided, prompt the user for a query immediately
				fmt.Println("Enter a search query (or press Enter to exit):")
				query, _ = reader.ReadString('\n')
				query = strings.TrimSpace(query)

				if query == "" {
					fmt.Println("No query entered, exiting.")
					os.Exit(0)
				}
			}

			// Main search loop
			for {
				fmt.Fprintln(os.Stdout, "Searching DuckDuckGo for:", query)

				// Perform search and retrieve results
				results, err := searchDuckDuckGo(query)
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error performing search:", err)
					fmt.Println("Enter a new search query (or press Enter to exit):")
					query, _ = reader.ReadString('\n')
					query = strings.TrimSpace(query)

					if query == "" {
						fmt.Println("No query entered, exiting.")
						os.Exit(0)
					}
					continue // Continue to ask for another query if an error occurs
				}

				// Display the top 10 search results with a preview of the snippet
				fmt.Println("Top 10 Search Results:")
				for i, result := range results {
					fmt.Printf("%d: %s\n", i+1, result.Title)
					fmt.Printf("   Link: %s\n", result.Link)
					fmt.Printf("   Snippet: %s\n\n", result.Snippet)
				}

				// Ask the user to select a link to open
				fmt.Println("Enter the number of the link you want to open (or press Enter to search again):")
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)

				if input == "" {
					// If no link is selected, prompt for a new search query
					fmt.Println("Enter a new search query (or press Enter to exit):")
					query, _ = reader.ReadString('\n')
					query = strings.TrimSpace(query)

					if query == "" {
						fmt.Println("No query entered, exiting.")
						os.Exit(0)
					}
					continue
				}

				// Convert input to an integer
				linkIndex, err := strconv.Atoi(input)
				if err != nil || linkIndex < 1 || linkIndex > len(results) {
					fmt.Println("Invalid selection, please try again.")
					continue
				}

				// Open the selected link in the browser
				selectedLink := results[linkIndex-1].Link
				fmt.Println("Opening link:", selectedLink)
				err = openUrlInBrowser(selectedLink)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to open link: %v\n", err)
				}

				// After the link is opened, prompt for a new search query
				fmt.Println("Enter a new search query (or press Enter to exit):")
				query, _ = reader.ReadString('\n')
				query = strings.TrimSpace(query)

				if query == "" {
					fmt.Println("No query entered, exiting.")
					os.Exit(0)
				}
			}
		default:
			go loading(&stopSpin)
			formattedInput := strings.TrimSpace(prompt)

			if len(formattedInput) < 1 {
				fmt.Fprintln(os.Stderr, "You need to write something")
				os.Exit(1)
			}

			getData(*preprompt+formattedInput+contextText+pipedInput, structs.Params{}, structs.ExtraOptions{IsNormal: true, IsInteractive: false, DisableInputLimit: *disableInputLimit})
		}
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		input := scanner.Text()
		go loading(&stopSpin)
		formattedInput := strings.TrimSpace(input)
		getData(*preprompt+formattedInput+pipedInput, structs.Params{}, structs.ExtraOptions{IsInteractive: false, DisableInputLimit: *disableInputLimit})
	}
}

// Multiline input
type errMsg error

type model struct {
	textarea textarea.Model
	err      error
}

func initialModel() model {
	size, _ := ts.GetSize()
	termWidth := size.Col()
	ti := textarea.New()
	ti.SetWidth(termWidth)
	ti.CharLimit = 200000
	ti.ShowLineNumbers = false
	ti.Placeholder = "Enter your prompt"
	ti.SetValue(*preprompt)
	*preprompt = ""
	ti.Focus()

	return model{
		textarea: ti,
		err:      nil,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			if m.textarea.Focused() {
				m.textarea.Blur()
			}
		case tea.KeyCtrlC:
			programLoop = false
			userInput = ""
			return m, tea.Quit

		case tea.KeyCtrlD:
			userInput = m.textarea.Value()

			if len(userInput) > 1 {
				m.textarea.Blur()
				return m, tea.Quit
			}
		case tea.KeyTab:
			if m.textarea.Focused() {
				m.textarea.InsertString("\t")
			}
		default:
			if m.textarea.Focused() {
				m.textarea, cmd = m.textarea.Update(msg)
				m.textarea.SetHeight(min(20, max(6, m.textarea.LineCount()+1)))
				cmds = append(cmds, cmd)
			}
		}

		// Command mode
		if !m.textarea.Focused() {
			switch msg.String() {
			case "i":
				m.textarea.Focus()
			case "c":
				if len(lastResponse) == 0 {
					break
				}
				err := clipboard.WriteAll(lastResponse)
				if err != nil {
					fmt.Println("Could not write to clipboard")
				}
			case "b":
				if len(lastResponse) == 0 {
					break
				}
				lastCodeBlock := getLastCodeBlock(lastResponse)
				err := clipboard.WriteAll(lastCodeBlock)
				if err != nil {
					fmt.Println("Could not write to clipboard")
				}
			case "p":
				m.textarea.Focus()
				clip, err := clipboard.ReadAll()
				msg.Runes = []rune(clip)
				if err != nil {
					fmt.Println("Could not read from clipboard")
				}
				userInput = clip
				m.textarea, cmd = m.textarea.Update(msg)
				m.textarea.SetHeight(min(20, max(6, m.textarea.LineCount()+1)))
				cmds = append(cmds, cmd)
			}
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return m.textarea.View()
}

func getFormattedInputStdin() (formattedInput string) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := scanner.Text()
	return strings.TrimSpace(input)
}

func showHelpMessage() {
	boldBlue.Println(`Usage: tgpt [Flags] [Prompt]`)

	boldBlue.Println("\nFlags:")
	fmt.Printf("%-50v Generate and Execute shell commands. (Experimental) \n", "-s, --shell")
	fmt.Printf("%-50v Generate Code. (Experimental)\n", "-c, --code")
	fmt.Printf("%-50v Gives response back without loading animation\n", "-q, --quiet")
	fmt.Printf("%-50v Gives response back as a whole text\n", "-w, --whole")
	fmt.Printf("%-50v Generate images from text\n", "-img, --image")
	fmt.Printf("%-50v Set Provider. Detailed information has been provided below. (Env: AI_PROVIDER)\n", "--provider")

	boldBlue.Println("\nSome additional options can be set. However not all options are supported by all providers. Not supported options will just be ignored.")
	fmt.Printf("%-50v Set Model\n", "--model")
	fmt.Printf("%-50v Set API Key\n", "--key")
	fmt.Printf("%-50v Set OpenAI API endpoint url\n", "--url")
	fmt.Printf("%-50v Set temperature\n", "--temperature")
	fmt.Printf("%-50v Set top_p\n", "--top_p")
	fmt.Printf("%-50v Set max response length\n", "--max_length")
	fmt.Printf("%-50v Set filepath to log conversation to (For interactive modes)\n", "--log")
	fmt.Printf("%-50v Set preprompt\n", "--preprompt")
	fmt.Printf("%-50v Execute shell command without confirmation\n", "-y")

	boldBlue.Println("\nOptions:")
	fmt.Printf("%-50v Print version \n", "-v, --version")
	fmt.Printf("%-50v Print help message \n", "-h, --help")
	fmt.Printf("%-50v Start normal interactive mode \n", "-i, --interactive")
	fmt.Printf("%-50v Start multi-line interactive mode \n", "-m, --multiline")
	fmt.Printf("%-50v See changelog of versions \n", "-cl, --changelog")

	if runtime.GOOS != "windows" {
		fmt.Printf("%-50v Update program \n", "-u, --update")
	}

	boldBlue.Println("\nProviders:")
	fmt.Println("The default provider is phind. The AI_PROVIDER environment variable can be used to specify a different provider.")
	fmt.Println("Available providers to use: blackboxai, duckduckgo, groq, koboldai, ollama, openai and phind")

	bold.Println("\nProvider: blackboxai")
	fmt.Println("Uses BlackBox model. Great for developers")

	bold.Println("\nProvider: duckduckgo")
	fmt.Println("Available models: gpt-4o-mini (default), meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo, mistralai/Mixtral-8x7B-Instruct-v0.1, claude-3-haiku-20240307")

	bold.Println("\nProvider: groq")
	fmt.Println("Requires a free API Key. Supports LLaMA2-70b & Mixtral-8x7b")

	bold.Println("\nProvider: koboldai")
	fmt.Println("Uses koboldcpp/HF_SPACE_Tiefighter-13B only, answers from novels")

	bold.Println("\nProvider: ollama")
	fmt.Println("Needs to be run locally. Supports many models")

	bold.Println("\nProvider: openai")
	fmt.Println("Needs API key to work and supports various models. Recognizes the OPENAI_API_KEY and OPENAI_MODEL environment variables. Supports custom urls with --url")

	bold.Println("\nProvider: phind")
	fmt.Println("Uses Phind Model. Great for developers")

	boldBlue.Println("\nExamples:")
	fmt.Println(`tgpt "What is internet?"`)
	fmt.Println(`tgpt -m`)
	fmt.Println(`tgpt -s "How to update my system?"`)
	fmt.Println(`tgpt --provider duckduckgo "What is 1+1"`)
	fmt.Println(`tgpt --provider openai --key "sk-xxxx" --model "gpt-3.5-turbo" "What is 1+1"`)
	fmt.Println(`cat install.sh | tgpt "Explain the code"`)
}

func historyCompleter(d Prompt.Document) []Prompt.Suggest {
	s := []Prompt.Suggest{}
	return Prompt.FilterHasPrefix(s, d.GetWordAfterCursor(), true)
}

func exit(_ *Prompt.Buffer) {
	bold.Println("Exiting...")

	if runtime.GOOS != "windows" {
		rawModeOff := exec.Command("stty", "-raw", "echo")
		rawModeOff.Stdin = os.Stdin
		_ = rawModeOff.Run()
		rawModeOff.Wait()
	}

	os.Exit(0)
}

func searchDuckDuckGo(query string) ([]SearchResult, error) {
	// Prepare the search query URL for DuckDuckGo (GET request)
	baseURL := "https://html.duckduckgo.com/html/"
	searchURL := fmt.Sprintf("%s?q=%s", baseURL, url.QueryEscape(query))

	// Create the GET request
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	// Set the User-Agent header to mimic a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check the status code of the response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch search results: status code %d", resp.StatusCode)
	}

	// Parse the HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	// Extract the titles, links, and snippets, and filter out ads
	var results []SearchResult
	doc.Find(".result").Each(func(i int, s *goquery.Selection) {
		// Check if the result is marked as an ad (assuming an ad has a specific class like `.result--ad`)
		if s.HasClass("result--ad") {
			// Skip the ads
			return
		}

		if i >= 10 { // Limit to top 10 non-ad results
			return
		}

		// Extract title
		title := s.Find(".result__title").Text()

		// Extract link
		link, exists := s.Find(".result__a").Attr("href")
		if !exists {
			return
		}

		// Extract snippet
		snippet := s.Find(".result__snippet").Text()

		// Append the result
		results = append(results, SearchResult{
			Title:   strings.TrimSpace(title),
			Link:    link,
			Snippet: strings.TrimSpace(snippet),
		})
	})

	// Check if any results were found
	if len(results) == 0 {
		return nil, fmt.Errorf("no results found, check CSS selector or response structure")
	}

	return results, nil
}
