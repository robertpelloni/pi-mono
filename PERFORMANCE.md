# Pi Agent Performance Metrics

This document records the performance characteristics of the Pi Go LLM Harness and its assimilated tool handlers.

## Baseline Benchmarks (v0.97.0)

Executed on standard dev sandbox environment.

### Tool Execution Latency
| Tool | Operation | Latency (Avg) |
| :--- | :--- | :--- |
| `warp_action` | Echo Command | ~3.5ms |
| `wave_action` | Read File (`go.mod`) | ~41µs |
| `tabby_completion` | Standard Request | ~175µs |
| `repo_map` | 100 Files (full scan) | ~1.8s |

### Throughput & Concurrency
- **Concurrent Workers**: 10
- **Total Requests**: 1,000
- **Successful Completions**: 100%
- **System Stability**: Stable, no race conditions detected in model registry or harness routing.

## Resource Recommendations

Based on the observed low-latency tool routing:
- **CPU**: 1-2 Cores (Go's scheduler efficiently manages the concurrent tool loop).
- **RAM**: 512MB - 1GB (Depending on the number of concurrent agent sessions).
- **Network**: Low latency to frontier LLM providers is the primary bottleneck, as the internal harness overhead is negligible (<5ms).
