import { wrapToolDefinition } from "./tool-definition-wrapper.js";
import type { AgentTool } from "@mariozechner/pi-agent-core";
import { Text } from "@mariozechner/pi-tui";

import { openInterpreterComputerUseSchema } from "./clean-room-schemas.js";
import { handleOpenInterpreterComputerUse, handleHermesMemory, handleAiderRunCommand, handleAiderReplaceLines, handleClineExecuteCommand, handleClineWriteToFile, handleClineAskFollowup } from "./clean-room-handlers.js";
import { hermesMemorySchema, aiderRunCommandSchema, aiderReplaceLinesSchema, clineExecuteCommandSchema, clineWriteToFileSchema, clineAskFollowupSchema } from "./clean-room-schemas.js";

export function createOpenInterpreterComputerUseTool(): AgentTool<typeof openInterpreterComputerUseSchema> {
    return wrapToolDefinition({
        name: "computer",
        label: "computer",
        description: "Interact with the primary monitor's screen, keyboard, and mouse (Open Interpreter format).",
        promptSnippet: "Use computer",
        promptGuidelines: [],
        parameters: openInterpreterComputerUseSchema,
        async execute(toolCallId, args, signal, onUpdate, ctx) {
            const output = await handleOpenInterpreterComputerUse(args);
            return { content: [{ type: "text", text: output }], details: {} };
        },
        renderCall(args, theme, context) {
            const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
            text.setText(`${theme.fg("toolTitle", theme.bold("computer"))} (${args.action})`);
            return text;
        },
        renderResult(result, options, theme, context) {
            const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
            text.setText("Computer action executed.");
            return text;
        }
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
        async execute(toolCallId, args, signal, onUpdate, ctx) {
            const output = await handleHermesMemory(args);
            return { content: [{ type: "text", text: output }], details: {} };
        },
        renderCall(args, theme, context) {
            const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
            text.setText(`${theme.fg("toolTitle", theme.bold("memory"))} (${args.key})`);
            return text;
        },
        renderResult(result, options, theme, context) {
            const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
            text.setText("Memory saved.");
            return text;
        }
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
        async execute(toolCallId, args, signal, onUpdate, ctx) {
            const output = await handleClineExecuteCommand(args);
            return { content: [{ type: "text", text: output }], details: {} };
        },
        renderCall(args, theme, context) {
            const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
            text.setText(`${theme.fg("toolTitle", theme.bold("execute_command"))} (${args.command})`);
            return text;
        },
        renderResult(result, options, theme, context) {
            const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
            text.setText("Command executed.");
            return text;
        }
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
        async execute(toolCallId, args, signal, onUpdate, ctx) {
            const output = await handleClineWriteToFile(args);
            return { content: [{ type: "text", text: output }], details: {} };
        },
        renderCall(args, theme, context) {
            const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
            text.setText(`${theme.fg("toolTitle", theme.bold("write_to_file"))} (${args.path})`);
            return text;
        },
        renderResult(result, options, theme, context) {
            const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
            text.setText("File written.");
            return text;
        }
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
        async execute(toolCallId, args, signal, onUpdate, ctx) {
            const output = await handleClineAskFollowup(args);
            return { content: [{ type: "text", text: output }], details: {} };
        },
        renderCall(args, theme, context) {
            const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
            text.setText(`${theme.fg("toolTitle", theme.bold("ask_followup_question"))}`);
            return text;
        },
        renderResult(result, options, theme, context) {
            const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
            text.setText("Question asked.");
            return text;
        }
    });
}
