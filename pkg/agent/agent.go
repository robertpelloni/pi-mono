package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

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

	streamFn ai.StreamFunction
	config   AgentLoopConfig
}

// NewAgent creates a new Agent instance with the given dependencies.
func NewAgent(model ai.ModelInfo, tools []AgentTool, streamFn ai.StreamFunction, config AgentLoopConfig) *Agent {
	return &Agent{
		model:            model,
		tools:            tools,
		streamFn:         streamFn,
		config:           config,
		pendingToolCalls: make(map[string]struct{}),
	}
}

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
	// Return a copy to prevent mutation
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

// Subscribe adds an event listener. It returns a function to unsubscribe.
func (a *Agent) Subscribe(listener AgentEventListener) func() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.listeners = append(a.listeners, listener)

	// Return an unsubscribe function (simplified; in real production we'd use IDs or reflect matching)
	return func() {
		// Omitted for brevity in this PoC
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

// Prompt processes a user input and triggers the LLM response stream.
func (a *Agent) Prompt(ctx context.Context, msg ai.UserMessage) error {
	a.mu.Lock()
	if a.isStreaming {
		a.mu.Unlock()
		return errors.New("Agent is already processing a prompt")
	}
	a.isStreaming = true
	a.messages = append(a.messages, msg)
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.isStreaming = false
		a.streamingMessage = nil
		a.mu.Unlock()
		a.emit(AgentEvent{Type: EventAgentEnd, Messages: a.Messages()})
	}()

	a.emit(AgentEvent{Type: EventAgentStart})

	return a.runLoop(ctx)
}

// runLoop handles the underlying call to the AI StreamFunction and tool executions
func (a *Agent) runLoop(ctx context.Context) error {
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
	a.mu.RUnlock()

	a.emit(AgentEvent{Type: EventTurnStart})

	// Call the generic StreamFunction logic defined in pkg/ai
	stream := a.streamFn(ctx, a.model, contextPayload, a.config.SimpleStreamOptions)

	var finalMsg = &ai.AssistantMessage{
		API:      a.model.API,
		Provider: a.model.Provider,
		Model:    a.model.ID,
		Content:  []ai.Content{},
	}

	var activeTextContent *ai.TextContent
	var activeToolCall *ai.ToolCall

	for event := range stream {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		switch event.Type {
		case ai.EventTextDelta:
			if event.Delta != nil {
				if activeTextContent == nil {
					activeTextContent = &ai.TextContent{Text: *event.Delta}
					finalMsg.Content = append(finalMsg.Content, *activeTextContent)
				} else {
					// We need to update the last element in Content
					activeTextContent.Text += *event.Delta
					finalMsg.Content[len(finalMsg.Content)-1] = *activeTextContent
				}
			}
		case ai.EventToolCallStart:
			if event.ToolCall != nil {
				activeTextContent = nil

				activeToolCall = &ai.ToolCall{
					ID:        event.ToolCall.ID,
					Name:      event.ToolCall.Name,
					Arguments: make(map[string]any),
				}

				activeToolCall.Arguments["raw_args"] = ""
				finalMsg.Content = append(finalMsg.Content, *activeToolCall)
			}
		case ai.EventToolCallDelta:
			if activeToolCall != nil && event.Delta != nil {
				if currentArgs, ok := activeToolCall.Arguments["raw_args"].(string); ok {
					activeToolCall.Arguments["raw_args"] = currentArgs + *event.Delta
					finalMsg.Content[len(finalMsg.Content)-1] = *activeToolCall
				}
			}
		case ai.EventToolCallEnd:
			if activeToolCall != nil {
				if rawArgs, ok := activeToolCall.Arguments["raw_args"].(string); ok && rawArgs != "" {
					var parsedArgs map[string]any
					if err := json.Unmarshal([]byte(rawArgs), &parsedArgs); err == nil {
						activeToolCall.Arguments = parsedArgs
						finalMsg.Content[len(finalMsg.Content)-1] = *activeToolCall
					}
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
			return fmt.Errorf("API stream error: %s", a.errorMessage)
		case ai.EventDone:
			if event.Reason != nil {
				finalMsg.StopReason = *event.Reason
			}

			if activeToolCall != nil {
				if rawArgs, ok := activeToolCall.Arguments["raw_args"].(string); ok && rawArgs != "" {
					var parsedArgs map[string]any
					if err := json.Unmarshal([]byte(rawArgs), &parsedArgs); err == nil {
						activeToolCall.Arguments = parsedArgs
						finalMsg.Content[len(finalMsg.Content)-1] = *activeToolCall
					}
				}
				activeToolCall = nil
			}

			goto StreamComplete
		}

		eventCopy := event
		eventCopy.Partial = finalMsg
		a.mu.Lock()
		a.streamingMessage = finalMsg
		a.mu.Unlock()
		a.emit(AgentEvent{Type: EventMessageUpdate, AssistantMessageEvent: &eventCopy})
	}

StreamComplete:
	if finalMsg != nil {
		a.mu.Lock()
		a.messages = append(a.messages, *finalMsg)
		a.mu.Unlock()

		a.emit(AgentEvent{Type: EventMessageEnd, Message: *finalMsg})

		// Here we would implement Tool Execution based on pendingToolCalls in a full port.
		// For this core PoC phase, we acknowledge the turn end.
		a.emit(AgentEvent{Type: EventTurnEnd, Message: *finalMsg, ToolResults: nil})
	}

	return nil
}
