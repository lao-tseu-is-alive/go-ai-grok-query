### Release 0.2.3 :
##### Configuration and Flexibility:
Added configuration override of base url of any supported llm via env variables
##### Code quality and robustness:
Added unit test and a set of e2e test with a mock server

### Release 0.2.2 :

##### Error Handling :
Added better log handling to debug errors in requests.

##### Configuration and Flexibility:
Added provider flag with easy selection of a default model, added system prompt flag

##### Code quality and readability:
refactored config package to remove code duplication.

### Release 0.2.1 :

##### Error Handling & Library-Neutrality:
Removed library-hostile behaviors like os.Exit(1) from the llm package (e.g., in tools.go and Check function). Replaced with proper error propagation, allowing calling code (like your main program) to handle failures. Added input validation and graceful error wrapping with fmt.Errorf.

##### Modularity and Pluggability:
Introduced interfaces (e.g., ToolExecutor) and structs for better extensibility, such as making tools like WeatherTool pluggable via methods instead of global functions. Suggested patterns like ExampleToolRegistry for managing multiple tools without hardcoding.

##### Thread-Safety and Concurrency:
Added mutexes (e.g., sync.RWMutex in Conversation) and methods like MessagesCopy() to prevent race conditions when accessing shared data, especially in long-running or concurrent contexts (e.g., futures or goroutines).

##### Type Safety and Compilation Fixes:
Resolved type mismatches (e.g., maps.Copy failing due to http.Header vs. map[string]string). Improved JSON handling, generics usage (e.g., in httpRequest), and removed unused variables/statements.

##### Readability and Documentation:
Added GoDoc comments, descriptive variable names, consistent formatting, and logging via slog. Replaced terse code (e.g., p in functions) with clearer constructs. Ensured Markdown-ready docstrings.

##### Configuration and Flexibility:
Enhanced provider setup with better defaults, validation for required fields (e.g., API key checks), and support for extra headers/limits. Made config optional where possible to reduce boilerplate.

##### Specific File Updates:

+ toolCalling.go: Converted to use structs (e.g., WeatherTool), added error checks, fixed undefined calls (e.g., getCurrentWeather → tool.Execute), and integrated safe conversation access.
tools.go: Exported helpers like FirstNonEmpty, removed side effects, and improved message serialization.

+ conversation.go: Added validation, thread-safety, and immutable copies.

+ openai_compatible.go: Fixed header merging loops, removed unused wireResponse, and refined unmarshaling.

+ provider.go and others: Deduplicated logic and added better error messages.

##### Why These Changes

+ Problems in Original Code: Issues like potential panics (e.g., nil pointer derefs), lack of concurrency safety, hardcoded solutions, and compile errors reduced the code's reliability for production use.
+ Goals Aligned with Best Practices: Followed Go idioms (e.g., error-as-value), SOLID principles (e.g., dependency injection via interfaces), and your requests for readability/easy maintenance. Made it "library-ready" (no exits) and adapted for real-world multi-tool, multi-provider scenarios.

##### Key Benefits
+ Usability: Easier to use in different contexts (e.g., web apps, CLIs) without worrying about crashes. Tools are now swap-in/swappable.

+ Maintainability: Cleaner separation of concerns, more tests-friendly (e.g., no global state), and self-documenting code reduces future bugs.

+ Robustness: Better handling of edge cases (e.g., empty responses, invalid inputs) with retries/panics. Improved security (e.g., no raw exits leaking control flow).

+ Performance/Simplicity: Efficient without over-engineering; supports streaming/tools without complexity blowup.

+ Scalability: Easing addition of new providers or tools with minimal rework.


### Release 0.2.0 :

The main evolutions in the llm package are a shift from a minimal, 
provider-specific shape to a unified, capability-aware interface 
that supports modern LLM features like function/tool calling, JSON/structured outputs, 
and streaming, while cleanly adapting differences among OpenAI-compatible APIs, Gemini, 
and local runtimes like Ollama. 

This makes the codebase more extensible, testable, and future-proof for providers such as OpenAI GPT-4/5, Gemini 2.5, and OpenRouter.

#### New Providers

+ OpenAi for using the famous [GPT 4 and 5 models](https://platform.openai.com/docs/models)
+ OpenRouter for using an amazing list of [models](https://openrouter.ai/models) at unbeatable prices

#### New unified interface
 + A new Provider interface now takes a context and a rich LLMRequest and returns an LLMResponse, with an optional Stream method for deltas; this replaces the rigid Query(system, user) signature and allows tools, response formats, and provider-specific options to flow through one place. This pattern mirrors how function calling and JSON mode are configured in modern APIs.

 + A single OpenAI-style adapter can serve both OpenAI and OpenRouter by changing only the base URL and headers, because OpenRouter implements the OpenAI chat/completions schema; optional X-Title and HTTP-Referer headers are supported for attribution.

#### Portable request/response model
 + Messages, tools, tool_choice, response_format, temperature, top_p, max_tokens, and extras are encapsulated in LLMRequest, aligning with OpenAI’s tools/tool_choice and response_format JSON mode while remaining flexible for other providers. This enables function calling, structured outputs, and reproducibility flags in a consistent way.

 + LLMResponse carries assistant text, normalized tool calls, finish reason, usage, and raw provider JSON for debugging, simplifying client logic across providers. This also matches common expectations from Chat Completions responses.

#### Function/tool calling readiness
 + The model supports declaring tools with JSON Schema and receiving normalized tool calls, which is the standard approach for building agentic loops where the model emits function arguments and the host executes the function, then returns the result for the next turn. This pattern is essential for production agents and is documented widely for OpenAI-style APIs.

 + With this abstraction, OpenAI and OpenRouter use the tools array and tool_choice; Gemini will map to function declarations and functionCall/response parts via its contents/parts format in the provider adapter, while exposing the same higher-level ToolCall objects to the caller.

#### JSON/structured outputs support
 + ResponseFormat enables JSON mode and structured outputs in OpenAI-compatible APIs, producing guaranteed-JSON or schema-constrained results that are robust to parsing, which is especially important for automation, integrations, and validation. Provider adapters can pass this through where supported and ignore otherwise.

#### Gemini-correct schema
 + Gemini integration transitions from an OpenAI-shaped message schema to the correct generateContent schema with contents/parts and systemInstruction, as per Google’s documentation. This resolves prior incompatibilities and sets the stage for Gemini function calling in a follow-up step.

#### OpenRouter compatibility via configuration
 + OpenRouter is supported by reusing the OpenAI adapter with a different base URL and headers; this consolidates code paths and ensures that tools, tool_choice, and response_format behave the same while leveraging OpenRouter’s multi-model marketplace. This also enables simple migration between OpenAI and OpenRouter.

Why this is important
 + Extensibility: Adding providers or models now requires only implementing an adapter that maps to the same high-level types, rather than duplicating bespoke request/response structs for each service. This lowers maintenance and accelerates adoption of new APIs.

 + Reliability and correctness: Normalized tool calls, JSON mode, and usage fields make agent loops and downstream parsing more reliable, which is critical in production settings where deterministic formats and error handling matter.

 + Future-proofing: The interface anticipates streaming, structured outputs, and provider-specific extras, which are becoming standard across vendors. This reduces churn when APIs evolve and lets the application code remain stable.

In short, the package moves from a minimal wrapper to a robust, cross-provider abstraction aligned with current LLM API patterns: tools/function calling, JSON/structured outputs, and flexible provider configuration, with correct Gemini schema handling and OpenRouter reuse of OpenAI compatibility. This unlocks reliable agents, portable chat, and easier multi-provider support