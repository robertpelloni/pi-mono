package ai

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func BenchmarkHarness_ExecuteTool(b *testing.B) {
	reg := &Registry{}
	h := NewHarness(reg)
	ctx := context.Background()

	args := map[string]interface{}{
		"type": "RequestCommandOutput",
		"params": map[string]interface{}{
			"command": "echo perf-test",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := h.ExecuteTool(ctx, "warp_action", args)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestHarness_Concurrency(t *testing.T) {
	reg := &Registry{}
	h := NewHarness(reg)
	ctx := context.Background()
	concurrency := 10
	iterations := 100

	done := make(chan bool)
	for i := 0; i < concurrency; i++ {
		go func(id int) {
			for j := 0; j < iterations; j++ {
				args := map[string]interface{}{
					"type": "readfile",
					"params": map[string]interface{}{
						"path": "go.mod",
					},
				}
				_, err := h.ExecuteTool(ctx, "wave_action", args)
				if err != nil {
					t.Errorf("worker %d failed: %v", id, err)
				}
			}
			done <- true
		}(i)
	}

	for i := 0; i < concurrency; i++ {
		<-done
	}
}

func TestHarness_EndToEnd(t *testing.T) {
	reg := &Registry{}
	h := NewHarness(reg)
	ctx := context.Background()

	testCases := []struct {
		name string
		tool string
		args map[string]interface{}
	}{
		{
			"Tabby Standard",
			"tabby_completion",
			map[string]interface{}{
				"segments": map[string]interface{}{"prefix": "test"},
			},
		},
		{
			"Warp Echo",
			"warp_action",
			map[string]interface{}{
				"type":   "RequestCommandOutput",
				"params": map[string]interface{}{"command": "echo e2e"},
			},
		},
		{
			"Wave Read",
			"wave_action",
			map[string]interface{}{
				"type":   "readfile",
				"params": map[string]interface{}{"path": "go.mod"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			resp, err := h.ExecuteTool(ctx, tc.tool, tc.args)
			duration := time.Since(start)

			if err != nil && tc.tool != "tabby_completion" { // tabby fails without models, which is expected
				t.Errorf("failed end-to-end %s: %v", tc.name, err)
			}

			fmt.Printf("End-to-End %s took %v\n", tc.name, duration)
			if resp == "" && tc.tool != "tabby_completion" {
				t.Errorf("empty response for %s", tc.name)
			}
		})
	}
}
