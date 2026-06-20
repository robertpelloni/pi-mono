import type { AgentTool } from "@mariozechner/pi-agent-core";
import { Text } from "@mariozechner/pi-tui";
import {
	handleAmpDiff,
	handleAmpReview,
	handleClineAskFollowup,
	handleClineBrowserAction,
	handleClineExecuteCommand,
	handleClineListCodeDefinitionNames,
	handleClineWriteToFile,
	handleGeminiReplace,
	handleGeminiRunShellCommand,
	handleHermesMemory,
	handleOpenCodeApplyPatch,
	handleOpenCodeMultiEdit,
	handleOpenInterpreterComputerUse,
} from "./clean-room-handlers.js";
import {
	ampDiffSchema,
	ampReviewSchema,
	clineAskFollowupSchema,
	clineBrowserActionSchema,
	clineExecuteCommandSchema,
	clineListCodeDefinitionNamesSchema,
	clineWriteToFileSchema,
	geminiReplaceSchema,
	geminiRunShellCommandSchema,
	hermesMemorySchema,
	openCodeApplyPatchSchema,
	openCodeMultiEditSchema,
	openInterpreterComputerUseSchema,
} from "./clean-room-schemas.js";
import { wrapToolDefinition } from "./tool-definition-wrapper.js";

export function createOpenInterpreterComputerUseTool(): AgentTool<typeof openInterpreterComputerUseSchema> {
	return wrapToolDefinition({
		name: "computer",
		label: "computer",
		description: "Interact with the primary monitor's screen, keyboard, and mouse (Open Interpreter format).",
		promptSnippet: "Use computer",
		promptGuidelines: [],
		parameters: openInterpreterComputerUseSchema,
		async execute(_toolCallId, args, _signal, _onUpdate, _ctx) {
			const output = await handleOpenInterpreterComputerUse(args);
			return { content: [{ type: "text", text: output }], details: {} };
		},
		renderCall(_args, theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			text.setText(`${theme.fg("toolTitle", theme.bold("computer"))} (${args.action})`);
			return text;
		},
		renderResult(_result, _options, _theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			text.setText("Computer action executed.");
			return text;
		},
	});
}

export function createAmpDiffTool(): AgentTool<typeof ampDiffSchema> {
	return wrapToolDefinition({
		name: "amp_diff",
		label: "amp_diff",
		description: "Review and stage changes directly in Amp.",
		promptSnippet: "Use amp_diff",
		promptGuidelines: [],
		parameters: ampDiffSchema,
		async execute(_toolCallId, args, _signal, _onUpdate, _ctx) {
			const output = await handleAmpDiff(args);
			return { content: [{ type: "text", text: output }], details: {} };
		},
		renderCall(_args, theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			text.setText(`${theme.fg("toolTitle", theme.bold("amp_diff"))}`);
			return text;
		},
		renderResult(_result, _options, _theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			return text;
		},
	});
}

export function createAmpReviewTool(): AgentTool<typeof ampReviewSchema> {
	return wrapToolDefinition({
		name: "amp_review",
		label: "amp_review",
		description: "Claude Opus 4.8 makes tighter changes and checks its own work.",
		promptSnippet: "Use amp_review",
		promptGuidelines: [],
		parameters: ampReviewSchema,
		async execute(_toolCallId, args, _signal, _onUpdate, _ctx) {
			const output = await handleAmpReview(args);
			return { content: [{ type: "text", text: output }], details: {} };
		},
		renderCall(_args, theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			text.setText(`${theme.fg("toolTitle", theme.bold("amp_review"))}`);
			return text;
		},
		renderResult(_result, _options, _theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			return text;
		},
	});
}

export function createHermesMemoryTool(): AgentTool<typeof hermesMemorySchema> {
	return wrapToolDefinition({
		name: "memory",
		label: "memory",
		description: "Save important information to persistent memory that survives across sessions.",
		promptSnippet: "Use memory",
		promptGuidelines: [],
		parameters: hermesMemorySchema,
		async execute(_toolCallId, args, _signal, _onUpdate, _ctx) {
			const output = await handleHermesMemory(args);
			return { content: [{ type: "text", text: output }], details: {} };
		},
		renderCall(_args, theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			text.setText(`${theme.fg("toolTitle", theme.bold("memory"))} (${args.key})`);
			return text;
		},
		renderResult(_result, _options, _theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			text.setText("Memory saved.");
			return text;
		},
	});
}

export function createClineExecuteCommandTool(): AgentTool<typeof clineExecuteCommandSchema> {
	return wrapToolDefinition({
		name: "execute_command",
		label: "execute_command",
		description: "Execute a CLI command on the system.",
		promptSnippet: "Use execute_command",
		promptGuidelines: [],
		parameters: clineExecuteCommandSchema,
		async execute(_toolCallId, args, _signal, _onUpdate, _ctx) {
			const output = await handleClineExecuteCommand(args);
			return { content: [{ type: "text", text: output }], details: {} };
		},
		renderCall(_args, theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			text.setText(`${theme.fg("toolTitle", theme.bold("execute_command"))} (${args.command})`);
			return text;
		},
		renderResult(_result, _options, _theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			text.setText("Command executed.");
			return text;
		},
	});
}

export function createClineWriteToFileTool(): AgentTool<typeof clineWriteToFileSchema> {
	return wrapToolDefinition({
		name: "write_to_file",
		label: "write_to_file",
		description: "Request to write content to a file at the specified path.",
		promptSnippet: "Use write_to_file",
		promptGuidelines: [],
		parameters: clineWriteToFileSchema,
		async execute(_toolCallId, args, _signal, _onUpdate, _ctx) {
			const output = await handleClineWriteToFile(args);
			return { content: [{ type: "text", text: output }], details: {} };
		},
		renderCall(_args, theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			text.setText(`${theme.fg("toolTitle", theme.bold("write_to_file"))} (${args.path})`);
			return text;
		},
		renderResult(_result, _options, _theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			text.setText("File written.");
			return text;
		},
	});
}

export function createClineAskFollowupTool(): AgentTool<typeof clineAskFollowupSchema> {
	return wrapToolDefinition({
		name: "ask_followup_question",
		label: "ask_followup_question",
		description: "Ask the user a question.",
		promptSnippet: "Use ask_followup_question",
		promptGuidelines: [],
		parameters: clineAskFollowupSchema,
		async execute(_toolCallId, args, _signal, _onUpdate, _ctx) {
			const output = await handleClineAskFollowup(args);
			return { content: [{ type: "text", text: output }], details: {} };
		},
		renderCall(_args, theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			text.setText(`${theme.fg("toolTitle", theme.bold("ask_followup_question"))}`);
			return text;
		},
		renderResult(_result, _options, _theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			text.setText("Question asked.");
			return text;
		},
	});
}

export function createClineListCodeDefinitionNamesTool(): AgentTool<typeof clineListCodeDefinitionNamesSchema> {
	return wrapToolDefinition({
		name: "list_code_definition_names",
		label: "list_code_definition_names",
		description: "List definition names (classes, functions, methods) used in source code files.",
		promptSnippet: "Use list_code_definition_names",
		promptGuidelines: [],
		parameters: clineListCodeDefinitionNamesSchema,
		async execute(_toolCallId, args, _signal, _onUpdate, _ctx) {
			const output = await handleClineListCodeDefinitionNames(args);
			return { content: [{ type: "text", text: output }], details: {} };
		},
		renderCall(_args, theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			text.setText(`${theme.fg("toolTitle", theme.bold("list_code_definition_names"))} (${args.path})`);
			return text;
		},
		renderResult(_result, _options, _theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			text.setText("Listed definitions.");
			return text;
		},
	});
}

export function createClineBrowserActionTool(): AgentTool<typeof clineBrowserActionSchema> {
	return wrapToolDefinition({
		name: "browser_action",
		label: "browser_action",
		description: "Interact with a Puppeteer-controlled browser.",
		promptSnippet: "Use browser_action",
		promptGuidelines: [],
		parameters: clineBrowserActionSchema,
		async execute(_toolCallId, args, _signal, _onUpdate, _ctx) {
			const output = await handleClineBrowserAction(args);
			return { content: [{ type: "text", text: output }], details: {} };
		},
		renderCall(_args, theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			text.setText(`${theme.fg("toolTitle", theme.bold("browser_action"))} (${args.action})`);
			return text;
		},
		renderResult(_result, _options, _theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			text.setText("Browser action complete.");
			return text;
		},
	});
}

export function createOpenCodeApplyPatchTool(): AgentTool<typeof openCodeApplyPatchSchema> {
	return wrapToolDefinition({
		name: "apply_patch",
		label: "apply_patch",
		description: "Apply a patch to files. The patch format is similar to unified diffs.",
		promptSnippet: "Use apply_patch",
		promptGuidelines: [],
		parameters: openCodeApplyPatchSchema,
		async execute(_toolCallId, args, _signal, _onUpdate, _ctx) {
			const output = await handleOpenCodeApplyPatch(args);
			return { content: [{ type: "text", text: output }], details: {} };
		},
		renderCall(_args, theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			text.setText(`${theme.fg("toolTitle", theme.bold("apply_patch"))}`);
			return text;
		},
		renderResult(_result, _options, _theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			return text;
		},
	});
}

export function createOpenCodeMultiEditTool(): AgentTool<typeof openCodeMultiEditSchema> {
	return wrapToolDefinition({
		name: "multiedit",
		label: "multiedit",
		description: "Apply multiple find-and-replace edits to a file.",
		promptSnippet: "Use multiedit",
		promptGuidelines: [],
		parameters: openCodeMultiEditSchema,
		async execute(_toolCallId, args, _signal, _onUpdate, _ctx) {
			const output = await handleOpenCodeMultiEdit(args);
			return { content: [{ type: "text", text: output }], details: {} };
		},
		renderCall(_args, theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			text.setText(`${theme.fg("toolTitle", theme.bold("multiedit"))}`);
			return text;
		},
		renderResult(_result, _options, _theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			return text;
		},
	});
}

export function createGeminiRunShellCommandTool(): AgentTool<typeof geminiRunShellCommandSchema> {
	return wrapToolDefinition({
		name: "run_shell_command",
		label: "run_shell_command",
		description: "Executes a shell command or script.",
		promptSnippet: "Use run_shell_command",
		promptGuidelines: [],
		parameters: geminiRunShellCommandSchema,
		async execute(_toolCallId, args, _signal, _onUpdate, _ctx) {
			const output = await handleGeminiRunShellCommand(args);
			return { content: [{ type: "text", text: output }], details: {} };
		},
		renderCall(_args, theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			text.setText(`${theme.fg("toolTitle", theme.bold("run_shell_command"))}`);
			return text;
		},
		renderResult(_result, _options, _theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			return text;
		},
	});
}

export function createGeminiReplaceTool(): AgentTool<typeof geminiReplaceSchema> {
	return wrapToolDefinition({
		name: "replace",
		label: "replace",
		description: "Replaces text within a file.",
		promptSnippet: "Use replace",
		promptGuidelines: [],
		parameters: geminiReplaceSchema,
		async execute(_toolCallId, args, _signal, _onUpdate, _ctx) {
			const output = await handleGeminiReplace(args);
			return { content: [{ type: "text", text: output }], details: {} };
		},
		renderCall(_args, theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			text.setText(`${theme.fg("toolTitle", theme.bold("replace"))}`);
			return text;
		},
		renderResult(_result, _options, _theme, context) {
			const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
			return text;
		},
	});
}
