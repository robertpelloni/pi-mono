# Recovery Guidance for Pi Agent

This guide provides a concise recovery plan and actionable instructions for restoring the Pi Agent when it enters a FAILED state.

## Recovery Plan

1. **Validate Session State**: Query the system to confirm the specific session ID is indeed in a FAILED state and capture any associated error logs from `~/.pi/sessions/<id>/logs` or the server output.
2. **Identify Root Cause**: Review recent activity logs and progress entries for patterns such as:
   - Tool integration failures (e.g., missing system dependencies like `ripgrep`).
   - Resource limit exhaustion (OOM, disk space).
   - LLM provider API timeouts or rate limits.
3. **Apply Immediate Fix**: Restart the agent process or re-initialize the failed tool integration. For the Go backend, this typically involves killing the process and restarting the server or CLI.
4. **Monitor**: Set a short-term health check (e.g., 5-minute interval via `GET /api/health`) to ensure the session remains in a RUNNING state.

## Operational Instruction

To verify and recover a failed agent service:
1. Verify the state:
   ```bash
   # Check logs for the specific session
   tail -f server.log | grep "ERROR"
   ```
2. Execute a restart:
   ```bash
   # Kill existing server and restart
   pkill pi-agent
   ./pi-agent server --port 8080 &
   ```
3. Re-submit the pending task and confirm success by checking if the session status returns to RUNNING in the UI or via `/api/sessions`.
