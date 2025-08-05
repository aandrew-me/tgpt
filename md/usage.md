## Usage 

```
Usage: tgpt [Flags] [Prompt]

Flags:
-s, --shell                                        Generate and Execute shell commands. (Experimental) 
-c, --code                                         Generate Code. (Experimental)
-q, --quiet                                        Gives response back without loading animation
-w, --whole                                        Gives response back as a whole text
-img, --image                                      Generate images from text
--provider                                         Set Provider. Detailed information has been provided below. (Env: AI_PROVIDER)

Some additional options can be set. However not all options are supported by all providers. Not supported options will just be ignored.
--model                                            Set Model
--key                                              Set API Key. (Env: AI_API_KEY)
--url                                              Set OpenAI API endpoint url
--temperature                                      Set temperature
--top_p                                            Set top_p
--log                                              Set filepath to log conversation to (For interactive modes)  
--preprompt                                        Set preprompt
-y                                                 Execute shell command without confirmation

Options supported for image generation (with -image flag)
--out                                              Output image filename (Supported by pollinations)
--height                                           Output image height (Supported by pollinations)
--width                                            Output image width (Supported by pollinations)
--img_count                                        Output image count (Supported by arta)
--img_negative                                     Negative prompt (Supported by arta)
--img_ratio                                        Output image ratio (Supported by arta, some models may not support it)

Options:
-v, --version                                      Print version
-h, --help                                         Print help message
-i, --interactive                                  Start normal interactive mode
-m, --multiline                                    Start multi-line interactive mode
-is, --interactive-shell                           Start interactive shell mode
-cl, --changelog                                   See changelog of versions

Providers:
The default provider is phind. The AI_PROVIDER environment variable can be used to specify a different provider.
Available providers to use: deepseek, gemini, groq, isou, koboldai, ollama, openai, pollinations and phind      

Provider: deepseek
Uses deepseek-reasoner model by default. Requires API key. Recognizes the DEEPSEEK_API_KEY and DEEPSEEK_MODEL environment variables

Provider: groq
Requires a free API Key. Supported models: https://console.groq.com/docs/models

Provider: gemini
Requires a free API key. https://aistudio.google.com/apikey

Provider: isou
Free provider with web search

Provider: koboldai
Uses koboldcpp/HF_SPACE_Tiefighter-13B only, answers from novels

Provider: ollama
Needs to be run locally. Supports many models

Provider: openai
Needs API key to work and supports various models. Recognizes the OPENAI_API_KEY and OPENAI_MODEL environment variables. Supports custom urls with --url

Provider: phind
Uses Phind Model. Great for developers

Provider: pollinations
Completely free, default model is gpt-4o. Supported models: https://text.pollinations.ai/models

Image generation providers:

Provider: pollinations
Supported models: flux, turbo

Provider: arta
Supported models:
Medieval, Vincent Van Gogh, F Dev, Low Poly, Dreamshaper-xl, Anima-pencil-xl, Biomech, Trash Polka, No Style, Cheyenne-xl, Chicano, Embroidery tattoo, Red and Black, Fantasy Art, Watercolor, Dotwork, Old school colored, Realistic tattoo, Japanese_2, Realistic-stock-xl, F Pro, RevAnimated, Katayama-mix-xl, SDXL L, Cor-epica-xl, Anime tattoo, New School, Death metal, Old School, Juggernaut-xl, Photographic, SDXL 1.0, Graffiti, Mini tattoo, Surrealism, Neo-traditional, On limbs black, Yamers-realistic-xl, Pony-xl, Playground-xl, Anything-xl, Flame design, Kawaii, Cinematic Art, Professional, Flux, Black Ink, Epicrealism-xl

Supported ratios:
1:1, 2:3, 3:2, 3:4, 4:3, 9:16, 16:9, 9:21, 21:9

Examples:
tgpt "What is internet?"
tgpt -m
tgpt -s "How to update my system?"
tgpt --provider duckduckgo "What is 1+1"
tgpt --img "cat"
tgpt --img --out ~/my-cat.jpg --height 256 --width 256 "cat"
tgpt --provider openai --key "sk-xxxx" --model "gpt-3.5-turbo" "What is 1+1"
cat install.sh | tgpt "Explain the code"
```