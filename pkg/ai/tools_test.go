package ai

import (
	"reflect"
	"testing"
)

func TestConvertResponsesTools(t *testing.T) {
	tools := []Tool{
		{
			Name:        "get_weather",
			Description: "Gets the weather for a location",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
	}

	t.Run("default options (no strict pointer)", func(t *testing.T) {
		converted := ConvertResponsesTools(tools, nil)
		if len(converted) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(converted))
		}

		c := converted[0]
		if c.Type != "function" {
			t.Errorf("expected type 'function', got %s", c.Type)
		}
		if c.Function.Name != "get_weather" {
			t.Errorf("expected name 'get_weather', got %s", c.Function.Name)
		}
		if c.Function.Strict != false {
			t.Errorf("expected strict false, got %v", c.Function.Strict)
		}
		if !reflect.DeepEqual(c.Function.Parameters, tools[0].Parameters) {
			t.Errorf("expected parameters to be deep equal")
		}
	})

	t.Run("strict enabled", func(t *testing.T) {
		strict := true
		opts := &ConvertResponsesToolsOptions{
			Strict: &strict,
		}
		converted := ConvertResponsesTools(tools, opts)

		c := converted[0]
		if c.Function.Strict != true {
			t.Errorf("expected strict true, got %v", c.Function.Strict)
		}
	})

	t.Run("strict disabled explicitly", func(t *testing.T) {
		strict := false
		opts := &ConvertResponsesToolsOptions{
			Strict: &strict,
		}
		converted := ConvertResponsesTools(tools, opts)

		c := converted[0]
		if c.Function.Strict != false {
			t.Errorf("expected strict false, got %v", c.Function.Strict)
		}
	})
}
