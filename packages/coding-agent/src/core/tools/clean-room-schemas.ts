import { z } from "zod";

/**
 * EXACT Parity Tool Schemas
 *
 * Models trained internally (Claude Code, Copilot, Cursor, etc.) expect VERY specific parameter names.
 * We must route all of these specific aliases to our core unified implementations.
 */

// --- READ ALIASES ---

export const claudeCodeReadSchema = z.object({
	file_path: z.string().describe("The absolute or relative path to the file to read."),
	offset: z.number().optional().describe("Optional line number to start reading from."),
	limit: z.number().optional().describe("Optional number of lines to read."),
});

export const copilotReadSchema = z.object({
	uri: z.string().describe("The URI or path of the file to read"),
});

export const aideReadSchema = z.object({
	filename: z.string().describe("The path to the file"),
});

// --- BASH ALIASES ---

export const claudeCodeBashSchema = z.object({
	command: z.string().describe("The bash command to run."),
});

export const aiderRunCommandSchema = z.object({
	cmd: z.string().describe("The command to run in the terminal"),
});

export const geminiShellSchema = z.object({
	script: z.string().describe("The shell script or command to execute"),
});

// --- GREP ALIASES ---

export const claudeCodeGrepSchema = z.object({
	pattern: z.string().describe("The regex pattern to search for"),
	path: z.string().optional().describe("The directory or file path to search in"),
	include: z.string().optional().describe("File glob to include"),
	exclude: z.string().optional().describe("File glob to exclude"),
});

export const openCodeSearchSchema = z.object({
	query: z.string().describe("The search query"),
	directory: z.string().optional().describe("The directory to search"),
});


// --- HERMES AGENT & II-AGENT PARITY SCHEMAS ---

export const hermesPatchSchema = z.object({
	file_path: z.string().describe("Path to the file to edit."),
	find: z.string().describe("The exact string or regex to find."),
	replace: z.string().describe("The replacement string."),
});

export const hermesSearchFilesSchema = z.object({
	target: z.string().describe("Either 'content' or 'name'."),
	query: z.string().describe("The search query or regex."),
	path: z.string().optional().describe("The directory to search in."),
});

export const hermesTerminalSchema = z.object({
	command: z.string().describe("The command string to execute"),
	background: z.boolean().optional().describe("Run in background?"),
});

export const hermesBrowserNavigateSchema = z.object({
	url: z.string(),
});

export const hermesBrowserClickSchema = z.object({
	ref_id: z.string(),
});

export const hermesBrowserTypeSchema = z.object({
	ref_id: z.string(),
	text: z.string(),
});

export const hermesClarifySchema = z.object({
	question: z.string(),
	choices: z.array(z.string()).optional(),
});

export const hermesExecuteCodeSchema = z.object({
	code: z.string(),
});

export const hermesCronjobSchema = z.object({
	action: z.enum(["create", "list", "update", "pause", "resume", "run", "remove"]),
	schedule: z.string().optional(),
	command: z.string().optional(),
});

export const hermesDelegateTaskSchema = z.object({
	task: z.string(),
	context: z.string().optional(),
});

export const hermesWriteFileSchema = z.object({
	file_path: z.string(),
	content: z.string(),
});

export const hermesHACallServiceSchema = z.object({
	domain: z.string(),
	service: z.string(),
	entity_id: z.string().optional(),
});

export const hermesMemorySchema = z.object({
	key: z.string(),
	value: z.string(),
});

export const hermesMOASchema = z.object({
	prompt: z.string(),
});

export const hermesSessionSearchSchema = z.object({
	query: z.string(),
});

export const hermesSkillManageSchema = z.object({
	action: z.enum(["create", "update", "delete"]),
	name: z.string(),
	content: z.string().optional(),
});

export const hermesWebSearchSchema = z.object({
	query: z.string(),
});
