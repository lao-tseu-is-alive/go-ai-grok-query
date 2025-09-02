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