package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/badlogic/pi-mono/pkg/ai"
)

// Agent defines the state and execution bounds for the autonomous loop.
type Agent struct {
	mu sync.RWMutex

	systemPrompt  string
	model         ai.ModelInfo
	thinkingLevel ai.ThinkingLevel
	tools         []AgentTool
	messages      []ai.Message

	isStreaming      bool
	streamingMessage *ai.AssistantMessage
	pendingToolCalls map[string]struct{}
	errorMessage     string

	listeners []AgentEventListener
	streamFn  ai.StreamFunction
	config    AgentLoopConfig
	steeringQueue     []ai.UserMessage
	followUpQueue     []ai.UserMessage
	cancelFn          context.CancelFunc
	exitDetector      *ExitDetector
}

// NewAgent creates a new Agent instance with the given dependencies.
func NewAgent(model ai.ModelInfo, tools []AgentTool, streamFn ai.StreamFunction, config AgentLoopConfig) *Agent {
	return &Agent{
		model:            model,
		tools:            tools,
		streamFn:         streamFn,
		config:           config,
		pendingToolCalls: make(map[string]struct{}),
		exitDetector:     NewExitDetector(),
	}
}

// --- AgentState interface ---

func (a *Agent) SystemPrompt() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.systemPrompt
}

func (a *Agent) SetSystemPrompt(s string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.systemPrompt = s
}

func (a *Agent) Model() ai.ModelInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.model
}

func (a *Agent) SetModel(m ai.ModelInfo) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.model = m
}

func (a *Agent) ThinkingLevel() ai.ThinkingLevel {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.thinkingLevel
}

func (a *Agent) SetThinkingLevel(t ai.ThinkingLevel) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.thinkingLevel = t
}

func (a *Agent) Tools() []AgentTool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	tools := make([]AgentTool, len(a.tools))
	copy(tools, a.tools)
	return tools
}

func (a *Agent) SetTools(t []AgentTool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tools = t
}

func (a *Agent) Messages() []ai.Message {
	a.mu.RLock()
	defer a.mu.RUnlock()
	msgs := make([]ai.Message, len(a.messages))
	copy(msgs, a.messages)
	return msgs
}

func (a *Agent) SetMessages(m []ai.Message) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.messages = m
}

func (a *Agent) IsStreaming() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.isStreaming
}

func (a *Agent) StreamingMessage() *ai.AssistantMessage {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.streamingMessage
}

func (a *Agent) PendingToolCalls() map[string]struct{} {
	a.mu.RLock()
	defer a.mu.RUnlock()
	m := make(map[string]struct{}, len(a.pendingToolCalls))
	for k, v := range a.pendingToolCalls {
		m[k] = v
	}
	return m
}

func (a *Agent) ErrorMessage() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.errorMessage
}

// --- Event system ---

// Subscribe adds an event listener. Returns an unsubscribe function.
func (a *Agent) Subscribe(listener AgentEventListener) func() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.listeners = append(a.listeners, listener)
	return func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		for i, l := range a.listeners {
			if &l == &listener {
				a.listeners = append(a.listeners[:i], a.listeners[i+1:]...)
				break
			}
		}
	}
}

func (a *Agent) emit(event AgentEvent) {
	a.mu.RLock()
	listeners := a.listeners
	a.mu.RUnlock()
	for _, l := range listeners {
		l(event)
	}
}

// --- Public API ---

// Prompt processes a user input and triggers the full agentic loop.
// This adds the user message, then runs the LLM -> tool-execution loop
// until the model stops with a non-tool reason or an error occurs.
func (a *Agent) Prompt(ctx context.Context, msg ai.UserMessage) error {
	a.mu.Lock()
	if a.isStreaming {
		a.mu.Unlock()
		return errors.New("agent is already processing a prompt")
	}
	a.isStreaming = true
	a.messages = append(a.messages, msg)
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.isStreaming = false
		a.streamingMessage = nil
		a.mu.Unlock()
	}()

	a.emit(AgentEvent{Type: EventAgentStart})

	// Emit message events for the user prompt
	a.emit(AgentEvent{Type: EventMessageStart, Message: msg})
	a.emit(AgentEvent{Type: EventMessageEnd, Message: msg})

	err := a.runLoop(ctx)

	a.emit(AgentEvent{Type: EventAgentEnd, Messages: a.Messages()})
	return err
}

// --- Core loop ---

// runLoop is the main agentic loop. It repeatedly:
//   1. Calls the LLM with the current context
//   2. Collects the assistant response
//   3. If the response contains tool calls, executes them and appends results
//   4. Loops back to step 1 with the updated context
//   5. Stops when the model gives a non-tool stop reason or an error occurs
func (a *Agent) runLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check for steering messages (injected between turns)
		if a.config.GetSteeringMessages != nil {
			steeringMsgs, err := a.config.GetSteeringMessages()
			if err == nil && len(steeringMsgs) > 0 {
				a.mu.Lock()
				for _, sm := range steeringMsgs {
					a.messages = append(a.messages, sm)
				}
				a.mu.Unlock()
				for _, sm := range steeringMsgs {
					a.emit(AgentEvent{Type: EventMessageStart, Message: sm})
					a.emit(AgentEvent{Type: EventMessageEnd, Message: sm})
				}
			}
		}

		a.emit(AgentEvent{Type: EventTurnStart})

		// Stream assistant response from the LLM
		assistantMsg, err := a.streamAssistantResponse(ctx)
		if err != nil {
			a.emit(AgentEvent{Type: EventTurnEnd, Message: ai.AssistantMessage{}, ToolResults: nil})
			return err
		}

		// Check stop reason: if error/aborted, end
		if assistantMsg.StopReason == ai.StopReasonError || assistantMsg.StopReason == ai.StopReasonAborted {
			a.emit(AgentEvent{Type: EventTurnEnd, Message: assistantMsg, ToolResults: nil})
			return nil
		}

		// Collect tool calls from the assistant message
		toolCalls := extractToolCalls(assistantMsg)

		// Antigravity Autopilot: Exit Detection
		if a.exitDetector != nil {
			var fullText strings.Builder
			for _, c := range assistantMsg.Content {
				if tc, ok := c.(ai.TextContent); ok {
					fullText.WriteString(tc.Text)
				}
			}
			exitResult := a.exitDetector.CheckResponse(fullText.String())
			if exitResult.ShouldExit {
				a.emit(AgentEvent{Type: EventTurnEnd, Message: assistantMsg, ToolResults: nil})
				return nil
			}
		}

		if len(toolCalls) == 0 {
			// No tool calls — agent turn is done
			a.emit(AgentEvent{Type: EventTurnEnd, Message: assistantMsg, ToolResults: nil})
			// Check for follow-up messages
			if a.config.GetFollowUpMessages != nil {
				followUps, err := a.config.GetFollowUpMessages()
				if err == nil && len(followUps) > 0 {
					a.mu.Lock()
					for _, fm := range followUps {
						a.messages = append(a.messages, fm)
					}
					a.mu.Unlock()
					for _, fm := range followUps {
						a.emit(AgentEvent{Type: EventMessageStart, Message: fm})
						a.emit(AgentEvent{Type: EventMessageEnd, Message: fm})
					}
					continue // loop back for another LLM call
				}
			}
			return nil
		}

		// Execute tool calls
		var toolResults []ai.ToolResultMessage
		if a.config.ToolExecution == ToolExecutionSequential {
			toolResults, err = a.executeToolCallsSequential(ctx, assistantMsg, toolCalls)
		} else {
			toolResults, err = a.executeToolCallsParallel(ctx, assistantMsg, toolCalls)
		}
		if err != nil {
			a.emit(AgentEvent{Type: EventTurnEnd, Message: assistantMsg, ToolResults: nil})
			return err
		}

		// Append tool results to message history
		a.mu.Lock()
		for _, tr := range toolResults {
			a.messages = append(a.messages, tr)
		}
		a.mu.Unlock()

		a.emit(AgentEvent{Type: EventTurnEnd, Message: assistantMsg, ToolResults: toolResults})

		// Loop continues — the next iteration will call the LLM again
		// with the tool results now in the message history
	}
}

// streamAssistantResponse calls the LLM StreamFunction and collects the full response.
func (a *Agent) streamAssistantResponse(ctx context.Context) (ai.AssistantMessage, error) {
	a.mu.RLock()
	contextPayload := ai.Context{
		Messages: a.messages,
	}
	if a.systemPrompt != "" {
		sysPrompt := a.systemPrompt
		contextPayload.SystemPrompt = &sysPrompt
	}
	var aiTools []ai.Tool
	for _, t := range a.tools {
		aiTools = append(aiTools, t.ToAITool())
	}
	contextPayload.Tools = aiTools

	// Apply context transform if configured
	if a.config.TransformContext != nil {
		transformed, err := a.config.TransformContext(ctx, a.messages)
		if err == nil && transformed != nil {
			contextPayload.Messages = transformed
		}
	}
	a.mu.RUnlock()

	// Call the AI StreamFunction
	stream := a.streamFn(ctx, a.model, contextPayload, a.config.SimpleStreamOptions)

	finalMsg := &ai.AssistantMessage{
		API:      a.model.API,
		Provider: a.model.Provider,
		Model:    a.model.ID,
		Content:  []ai.Content{},
	}
	var activeTextContent *ai.TextContent
	var activeToolCall *ai.ToolCall

	a.emit(AgentEvent{Type: EventMessageStart, Message: *finalMsg})

	for event := range stream {
		select {
		case <-ctx.Done():
			return ai.AssistantMessage{}, ctx.Err()
		default:
		}

		switch event.Type {
		case ai.EventTextDelta:
			if event.Delta != nil {
				if activeTextContent == nil {
					activeTextContent = &ai.TextContent{Text: *event.Delta}
					finalMsg.Content = append(finalMsg.Content, *activeTextContent)
				} else {
					activeTextContent.Text += *event.Delta
					finalMsg.Content[len(finalMsg.Content)-1] = *activeTextContent
				}
			}

		case ai.EventThinkingStart:
			// Forward thinking events
			eventCopy := event
			eventCopy.Partial = finalMsg
			a.emit(AgentEvent{Type: EventMessageUpdate, AssistantMessageEvent: &eventCopy})

		case ai.EventThinkingDelta:
			if event.Delta != nil {
				eventCopy := event
				eventCopy.Partial = finalMsg
				a.emit(AgentEvent{Type: EventMessageUpdate, AssistantMessageEvent: &eventCopy})
			}

		case ai.EventToolCallStart:
			if event.ToolCall != nil {
				activeTextContent = nil
				activeToolCall = &ai.ToolCall{
					ID:   event.ToolCall.ID,
					Name: event.ToolCall.Name,
					Arguments: map[string]any{
						"__raw_args__": "",
					},
				}
				finalMsg.Content = append(finalMsg.Content, *activeToolCall)
			}

		case ai.EventToolCallDelta:
			if activeToolCall != nil && event.Delta != nil {
				if currentArgs, ok := activeToolCall.Arguments["__raw_args__"].(string); ok {
					activeToolCall.Arguments["__raw_args__"] = currentArgs + *event.Delta
					finalMsg.Content[len(finalMsg.Content)-1] = *activeToolCall
				}
			}

		case ai.EventToolCallEnd:
			if activeToolCall != nil {
				if rawArgs, ok := activeToolCall.Arguments["__raw_args__"].(string); ok && rawArgs != "" {
					var parsedArgs map[string]any
					if err := json.Unmarshal([]byte(rawArgs), &parsedArgs); err == nil {
						activeToolCall.Arguments = parsedArgs
					} else {
						activeToolCall.Arguments = map[string]any{"__raw_args__": rawArgs}
					}
					finalMsg.Content[len(finalMsg.Content)-1] = *activeToolCall
				}
				activeToolCall = nil
			}

		case ai.EventError:
			a.mu.Lock()
			if event.Error != nil && event.Error.ErrorMessage != nil {
				a.errorMessage = *event.Error.ErrorMessage
			} else {
				a.errorMessage = "unknown API error"
			}
			a.mu.Unlock()
			finalMsg.StopReason = ai.StopReasonError
			a.mu.Lock()
			a.messages = append(a.messages, *finalMsg)
			a.mu.Unlock()
			a.emit(AgentEvent{Type: EventMessageEnd, Message: *finalMsg})
			return *finalMsg, fmt.Errorf("API stream error: %s", a.errorMessage)

		case ai.EventDone:
			if event.Reason != nil {
				finalMsg.StopReason = *event.Reason
			}
			// Finalize any remaining active tool call
			if activeToolCall != nil {
				if rawArgs, ok := activeToolCall.Arguments["__raw_args__"].(string); ok && rawArgs != "" {
					var parsedArgs map[string]any
					if err := json.Unmarshal([]byte(rawArgs), &parsedArgs); err == nil {
						activeToolCall.Arguments = parsedArgs
					}
					finalMsg.Content[len(finalMsg.Content)-1] = *activeToolCall
				}
				activeToolCall = nil
			}
			// Done — break out of the stream loop
			goto StreamComplete
		}

		// Forward streaming updates to UI
		eventCopy := event
		eventCopy.Partial = finalMsg
		a.mu.Lock()
		a.streamingMessage = finalMsg
		a.mu.Unlock()
		a.emit(AgentEvent{Type: EventMessageUpdate, AssistantMessageEvent: &eventCopy})
	}

StreamComplete:
	// Set usage/timestamp if available
	finalMsg.Timestamp = time.Now().UnixMilli()

	a.mu.Lock()
	a.messages = append(a.messages, *finalMsg)
	a.mu.Unlock()

	a.emit(AgentEvent{Type: EventMessageEnd, Message: *finalMsg})
	return *finalMsg, nil
}

// --- Tool execution ---

// extractToolCalls returns all ToolCall content from an assistant message.
func extractToolCalls(msg ai.AssistantMessage) []ai.ToolCall {
	var calls []ai.ToolCall
	for _, c := range msg.Content {
		if tc, ok := c.(ai.ToolCall); ok {
			calls = append(calls, tc)
		}
	}
	return calls
}

// executeToolCallsSequential runs tool calls one after another.
func (a *Agent) executeToolCallsSequential(ctx context.Context, assistantMsg ai.AssistantMessage, toolCalls []ai.ToolCall) ([]ai.ToolResultMessage, error) {
	var results []ai.ToolResultMessage
	for _, tc := range toolCalls {
		result, err := a.executeSingleToolCall(ctx, assistantMsg, tc)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}
	return results, nil
}

// executeToolCallsParallel runs tool calls concurrently.
func (a *Agent) executeToolCallsParallel(ctx context.Context, assistantMsg ai.AssistantMessage, toolCalls []ai.ToolCall) ([]ai.ToolResultMessage, error) {
	type indexedResult struct {
		index  int
		result ai.ToolResultMessage
		err    error
	}
	ch := make(chan indexedResult, len(toolCalls))

	for i, tc := range toolCalls {
		go func(idx int, toolCall ai.ToolCall) {
			result, err := a.executeSingleToolCall(ctx, assistantMsg, toolCall)
			ch <- indexedResult{index: idx, result: result, err: err}
		}(i, tc)
	}

	results := make([]ai.ToolResultMessage, len(toolCalls))
	for range toolCalls {
		ir := <-ch
		if ir.err != nil {
			return results[:ir.index], ir.err
		}
		results[ir.index] = ir.result
	}
	return results, nil
}

// executeSingleToolCall handles the full lifecycle of one tool call:
// resolve tool, prepare args, before-hook, execute, after-hook, emit events.
func (a *Agent) executeSingleToolCall(ctx context.Context, assistantMsg ai.AssistantMessage, tc ai.ToolCall) (ai.ToolResultMessage, error) {
	// Clean up __raw_args__ if present
	args := tc.Arguments
	if rawArgs, ok := args["__raw_args__"].(string); ok {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(rawArgs), &parsed); err == nil {
			args = parsed
		} else {
			// Fallback: treat raw string as a single "input" parameter
			args = map[string]any{"input": rawArgs}
		}
	}

	// Resolve the tool by name
	var tool *AgentTool
	a.mu.RLock()
	for i := range a.tools {
		if a.tools[i].Name == tc.Name {
			tool = &a.tools[i]
			break
		}
	}
	a.mu.RUnlock()

	if tool == nil {
		return a.emitToolCallError(ctx, tc, fmt.Sprintf("tool %q not found", tc.Name)), nil
	}

	// Prepare arguments if the tool has a PrepareArguments hook
	if tool.PrepareArguments != nil {
		prepared, err := tool.PrepareArguments(args)
		if err != nil {
			return a.emitToolCallError(ctx, tc, fmt.Sprintf("argument preparation failed: %v", err)), nil
		}
		args = prepared
	}

	// Advanced Reasoning: if tools contain a "plan" or "reasoning" step, we trace it specifically
	if tc.Name == "request_plan_review" || tc.Name == "react_fallback" {
		a.emit(AgentEvent{
			Type:      "thinking_start", // Treat planning blocks as deep thinking logic visually
		})
		defer a.emit(AgentEvent{Type: "thinking_end"})
	}

	// Emit tool_execution_start
	a.emit(AgentEvent{
		Type:      EventToolExecutionStart,
		ToolCallID: tc.ID,
		ToolName:  tc.Name,
		Args:      args,
	})

	// BeforeToolCall hook
	if a.config.BeforeToolCall != nil {
		callCtx := BeforeToolCallContext{
			AssistantMessage: assistantMsg,
			ToolCall:         tc,
			Args:             args,
			AgentContext: AgentContext{
				SystemPrompt: a.systemPrompt,
				Messages:     a.messages,
				Tools:        a.tools,
			},
		}
		beforeResult, err := a.config.BeforeToolCall(ctx, callCtx)
		if err != nil {
			return a.emitToolCallError(ctx, tc, fmt.Sprintf("beforeToolCall hook error: %v", err)), nil
		}
		if beforeResult != nil && beforeResult.Block {
			msg := "tool execution was blocked"
			return a.emitToolCallOutcome(ctx, tc, AgentToolResult{
				Content: []ai.Content{ai.TextContent{Text: msg}},
			}, true), nil
		}
	}

	// Execute the tool
	var toolResult AgentToolResult
	var execErr error

	onUpdate := func(partialResult AgentToolResult) {
		a.emit(AgentEvent{
			Type:      EventToolExecutionUpdate,
			ToolCallID: tc.ID,
			ToolName:  tc.Name,
			Args:      args,
			PartialResult: partialResult,
		})
	}

	toolResult, execErr = tool.Execute(ctx, tc.ID, args, onUpdate)

	isError := execErr != nil
	if isError {
		errMsg := execErr.Error()
		toolResult = AgentToolResult{
			Content: []ai.Content{ai.TextContent{Text: errMsg}},
		}
	}

	// AfterToolCall hook
	if a.config.AfterToolCall != nil {
		callCtx := AfterToolCallContext{
			AssistantMessage: assistantMsg,
			ToolCall:         tc,
			Args:             args,
			Result:           toolResult,
			IsError:          isError,
			AgentContext: AgentContext{
				SystemPrompt: a.systemPrompt,
				Messages:     a.messages,
				Tools:        a.tools,
			},
		}
		afterResult, err := a.config.AfterToolCall(ctx, callCtx)
		if err == nil && afterResult != nil {
			if afterResult.Content != nil {
				toolResult.Content = afterResult.Content
			}
			if afterResult.Details != nil {
				toolResult.Details = afterResult.Details
			}
			if afterResult.IsError != nil {
				isError = *afterResult.IsError
			}
		}
	}

	return a.emitToolCallOutcome(ctx, tc, toolResult, isError), nil
}

// emitToolCallOutcome sends the tool_execution_end, message_start/end events
// and returns the ToolResultMessage to be appended to the conversation.
func (a *Agent) emitToolCallOutcome(ctx context.Context, tc ai.ToolCall, result AgentToolResult, isError bool) ai.ToolResultMessage {
	a.emit(AgentEvent{
		Type:      EventToolExecutionEnd,
		ToolCallID: tc.ID,
		ToolName:  tc.Name,
		Result:    result,
		IsError:   isError,
	})

	toolResultMsg := ai.ToolResultMessage{
		ToolCallID: tc.ID,
		ToolName:  tc.Name,
		Content:   result.Content,
		Details:   result.Details,
		IsError:   isError,
		Timestamp: time.Now().UnixMilli(),
	}

	a.emit(AgentEvent{Type: EventMessageStart, Message: toolResultMsg})
	a.emit(AgentEvent{Type: EventMessageEnd, Message: toolResultMsg})

	return toolResultMsg
}

// emitToolCallError is a convenience for emitting error tool results.
func (a *Agent) emitToolCallError(ctx context.Context, tc ai.ToolCall, errMsg string) ai.ToolResultMessage {
	return a.emitToolCallOutcome(ctx, tc, AgentToolResult{
		Content: []ai.Content{ai.TextContent{Text: errMsg}},
		Details: map[string]any{"error": errMsg},
	}, true)
}


// Steer queues a steering message that interrupts the current turn.
func (a *Agent) Steer(msg ai.UserMessage) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.steeringQueue = append(a.steeringQueue, msg)
}

// FollowUp queues a follow-up message to be processed after the agent finishes.
func (a *Agent) FollowUp(msg ai.UserMessage) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.followUpQueue = append(a.followUpQueue, msg)
}

// Continue resumes the agent loop (used after retry or compaction recovery).
func (a *Agent) Continue() {
	a.mu.Lock()
	if a.isStreaming {
		a.mu.Unlock()
		return
	}
	a.isStreaming = true
	a.mu.Unlock()
	go func() {
		defer func() {
			a.mu.Lock()
			a.isStreaming = false
			a.mu.Unlock()
		}()
		a.emit(AgentEvent{Type: EventAgentStart})
		ctx := context.Background()
		if err := a.runLoop(ctx); err != nil {
			// Log error but do not crash
		}
		a.emit(AgentEvent{Type: EventAgentEnd, Messages: a.Messages()})
	}()
}

// Abort cancels the current agent operation.
func (a *Agent) Abort() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancelFn != nil {
		a.cancelFn()
		a.cancelFn = nil
	}
}

// WaitForIdle blocks until the agent is not streaming.
func (a *Agent) WaitForIdle() {
	for {
		a.mu.RLock()
		streaming := a.isStreaming
		a.mu.RUnlock()
		if !streaming {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// HasQueuedMessages returns whether there are steering or follow-up messages pending.
func (a *Agent) HasQueuedMessages() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.steeringQueue) > 0 || len(a.followUpQueue) > 0
}

// ClearAllQueues removes all queued steering and follow-up messages.
func (a *Agent) ClearAllQueues() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.steeringQueue = nil
	a.followUpQueue = nil
}

// ExecuteToolCall executes a single tool by name with the given arguments.
func (a *Agent) ExecuteToolCall(ctx context.Context, toolName string, args map[string]interface{}) (*AgentToolResult, error) {
	a.mu.RLock()
	var tool *AgentTool
	for i := range a.tools {
		if a.tools[i].Name == toolName {
			tool = &a.tools[i]
			break
		}
	}
	a.mu.RUnlock()
	if tool == nil {
		return nil, fmt.Errorf("tool %q not found", toolName)
	}
	result, err := tool.Execute(ctx, "", args, nil)
	if err != nil {
		return &AgentToolResult{
			Content: []ai.Content{ai.TextContent{Text: err.Error()}},
		}, err
	}
	return &result, nil
}

