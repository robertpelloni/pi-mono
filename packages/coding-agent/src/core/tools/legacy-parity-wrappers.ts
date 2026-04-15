import { createBashToolDefinition } from "./bash.js";
import { createEditToolDefinition } from "./edit.js";
import { createGrepToolDefinition } from "./grep.js";
import { createReadToolDefinition } from "./read.js";

// Legacy Claude/Codex tool schemas often use 'replace' instead of 'edit',
// or slightly different argument names like 'file' instead of 'path'.
// This file exports wrapper tools to ensure 100% parity with various IDE models.

export const replaceToolDefinition = (cwd: string) => {
	const def = createEditToolDefinition(cwd);
	return {
		...def,
		name: "replace",
		description:
			"Alias for edit tool to maintain exact parity with Claude/Codex internal schemas. Replaces text in a file.",
	};
};

export const searchToolDefinition = (cwd: string) => {
	const def = createGrepToolDefinition(cwd);
	return {
		...def,
		name: "search",
		description:
			"Alias for grep tool to maintain exact parity with Copilot/Gemini internal schemas. Searches text across files.",
	};
};

export const readFileToolDefinition = (cwd: string) => {
	const def = createReadToolDefinition(cwd);
	return {
		...def,
		name: "read_file", // common alias
		description: "Reads the contents of a file.",
	};
};

export const executeToolDefinition = (cwd: string) => {
	const def = createBashToolDefinition(cwd);
	return {
		...def,
		name: "execute_command", // common alias
		description: "Executes a shell command.",
	};
};
