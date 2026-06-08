package server

import (
	"context"
	"github.com/badlogic/pi-mono/pkg/ai"
)

func mockStreamIntegration(ctx context.Context, model ai.ModelInfo, aiCtx ai.Context, options any) ai.AssistantMessageEventStream {
	ch := make(chan ai.AssistantMessageEvent)
	go func() {
		defer close(ch)
		delta := "mock-response"
		ch <- ai.AssistantMessageEvent{Type: ai.EventTextDelta, Delta: &delta}
		reason := ai.StopReasonStop
		ch <- ai.AssistantMessageEvent{Type: ai.EventDone, Reason: &reason}
	}()
	return ch
}
