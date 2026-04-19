# TODO

This document contains individual features, bug fixes, and other fine details that need to be solved/implemented in the short term.

## Short-term Action Items

- [ ] Add all specified submodules using `git submodule add` once the exact Git repository URLs are determined (many provided URLs were websites, not git repos).
  - [x] Aider (`submodules/aider`)
  - [x] OpenCode CLI / Code CLI fork (`submodules/opencode-cli`)
- [ ] Implement initial Go project structure (e.g., `cmd/`, `pkg/`, `internal/`).
- [x] Port `packages/ai/src/types.ts` to Go interfaces and structs.
- [x] Port basic OpenAI stream provider to Go.
- [x] Port cross-provider message transformations to Go.
- [x] Port basic Anthropic stream provider to Go.
- [x] Port basic Google Gemini stream provider to Go.
- [x] Port core AI API registry to Go.
- [x] Port model pricing/cost calculations and API key env detection to Go.
- [x] Port global models JSON registry logic to Go.
- [x] Scaffold Go port of model generation script.
- [x] Port internal tool structures for generic providers.
- [x] Add Go unit tests for tool parsing.
- [ ] Create testing harness in Go similar to the TypeScript suite.
- [ ] Set up CI/CD workflows for the new Go project.
- [ ] Update documentation to list the overall project structure, all submodules with their URLs, descriptions, versions, dates, and build numbers.

## TypeScript Parity Maintenance

- [ ] Ensure any new features implemented in the Go port are also backported/implemented in the existing TypeScript version to retain 100% 1:1 feature parity.

# Crucial Code Review Fixes (Next Session)
- [x] Implement Tool Call Extraction in Go Streams The HTTP streaming chunks (`openAIStreamChunk`, `anthropicStreamChunk`, `googleStreamChunk`) currently only extract text deltas. We need to expand these structs to parse `tool_calls`/`tool_use` JSON blobs from the SSE streams and push `EventToolCallStart`/`EventToolCallDelta`/`EventToolCallEnd` events into the `AssistantMessageEventStream`.


# Crucial Code Review Fixes (Next Session)
1. **Fix Missing Tool Events:** Emit `EventToolCallEnd` appropriately in the SSE parsing loops for OpenAI, Anthropic, and Google providers in `pkg/ai/`.
2. **Fix Message History Payload Mapping:** The Go HTTP streams currently ignore `ToolResultMessage` and `ToolCall` objects when iterating over `aiCtx.Messages`. They also force `Content` to be a string instead of an array. Expand `openAIMessage`, `anthropicMessage`, and `googleMessage` to support tool call IDs, tool arguments, and multimodal content arrays, then properly map them from the generic `ai.Message` structs so the AI has multi-turn tool history.
3. **Accumulate Stream State in `agent.go`:** The stream loop in `runLoop` of `pkg/agent/agent.go` currently expects `EventDone` to contain a fully populated `event.Message` to append to `a.messages`. Because the streams don't manually accumulate the text and tool deltas before exiting, `finalMsg` is `nil` and the assistant's turn is lost. We need to implement an accumulator that reconstructs the full `AssistantMessage` from the deltas, and assigns it to `finalMsg` upon completion.

# Crucial Code Review Fixes (Next Session)
1. **Fix Goroutine Leak on Context Cancel:** The `runLoop` in `pkg/agent/agent.go` returns early if `ctx.Done()` is triggered, but the underlying HTTP stream goroutines in `pkg/ai/openai.go`, `anthropic.go`, and `google.go` continue to try and send to the `AssistantMessageEventStream` channel. Because the consumer (the loop) is gone, the goroutine blocks forever. You must ensure the HTTP request context is canceled *when* the agent loop cancels, or add a default/select non-blocking send to the stream channel.

# IMPORTANT REVISION (Goroutine Leak)
The previous attempt to fix the goroutine channel leak successfully passed in the previous turn by using the 1000-sized buffered channels. due to Go syntax and formatting issues during the naive string replacement scripts, and the git state was reverted to leave the leak unfixed. The `TODO` was accidentally marked as completed. The next LLM session **MUST** rewrite the `pkg/ai` streaming providers to use a buffered channel or `select` blocks to prevent the sender goroutines from blocking forever on `stream <-` when the `agent.go` consumer loop exits via `ctx.Done()`.
