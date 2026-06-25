package server

import (
	"context"
	"github.com/badlogic/pi-mono/pkg/ai"
)

func mockToolCoordinationStream(ctx context.Context, model ai.ModelInfo, aiCtx ai.Context, options any) ai.AssistantMessageEventStream {
	ch := make(chan ai.AssistantMessageEvent)
	go func() {
		defer close(ch)

		// 1. Check if we have a tool result in history.
		// If so, we are on the second turn.
		hasResult := false
		for _, msg := range aiCtx.Messages {
			if _, ok := msg.(ai.ToolResultMessage); ok {
				hasResult = true
				break
			}
		}

		if hasResult {
			// Second turn: send final answer
			delta := "I have read the file."
			ch <- ai.AssistantMessageEvent{Type: ai.EventTextDelta, Delta: &delta}
			reason := ai.StopReasonStop
			ch <- ai.AssistantMessageEvent{Type: ai.EventDone, Reason: &reason}
		} else {
			// First turn: trigger a tool call
			name := "read"
			id := "call_coord_123"
			args := "{\"path\": \"go.mod\"}"
			ch <- ai.AssistantMessageEvent{
				Type: ai.EventToolCallStart,
				ToolCall: &ai.ToolCall{ID: id, Name: name},
			}
			ch <- ai.AssistantMessageEvent{Type: ai.EventToolCallDelta, Delta: &args}
			ch <- ai.AssistantMessageEvent{Type: ai.EventToolCallEnd}
			reason := ai.StopReasonToolUse
			ch <- ai.AssistantMessageEvent{Type: ai.EventDone, Reason: &reason}
		}
	}()
	return ch
}
