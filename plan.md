1. Use `run_in_bash_session` to write a comprehensive summary into `MEMORY.md` to document the learned architecture, patterns, and decisions.
2. Use `run_in_bash_session` to update `HANDOFF.md` to indicate the compilation of the project memory summary.
3. Verify the updates to `MEMORY.md` and `HANDOFF.md` using `run_in_bash_session` with `cat`.
4. Run tests using `npm run test --workspaces --if-present` and `go test $(go list ./... | grep -v 'submodules')`.
5. Use `run_in_bash_session` to run `git add MEMORY.md HANDOFF.md` and `git commit` to commit the documentation updates.
6. Complete pre-commit steps to ensure proper testing, verification, review, and reflection are done.
7. Submit the task with `submit` tool using `branch_name`, `commit_message`, `title`, and `description`.
