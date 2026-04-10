package tools

import (
	"context"
	"encoding/json"

	"github.com/mariozechner/pi-mono/go-packages/ai/types"
)

// AgentTool represents a tool available to the agent.
type AgentTool interface {
	Definition() types.Tool
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}
