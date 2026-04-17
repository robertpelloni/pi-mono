import { Text } from "@mariozechner/pi-tui";
import type { AgentTool } from "@mariozechner/pi-agent-core";
import { createBashToolDefinition } from "./bash.js";
import { wrapToolDefinition } from "./tool-definition-wrapper.js";

import { aiderReplaceLinesSchema, aiderRunCommandSchema } from "./clean-room-schemas.js";
import { handleAiderReplaceLines } from "./clean-room-handlers.js";

export function createAiderRunCommandTool(cwd: string): AgentTool<typeof aiderRunCommandSchema> {
    const baseDef = createBashToolDefinition(cwd);
    return wrapToolDefinition({
        name: "run_command",
        label: "run_command",
        description: "Run a command in the terminal (Aider format).",
        promptSnippet: "Run a command",
        promptGuidelines: [],
        parameters: aiderRunCommandSchema,
        async execute(toolCallId, args, signal, onUpdate, ctx) {
            // Map cmd -> command
            return baseDef.execute(toolCallId, { command: args.cmd }, signal, onUpdate, ctx);
        },
        renderCall(args, theme, context) {
            const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
            text.setText(`${theme.fg("toolTitle", theme.bold("run_command"))} ${args.cmd}`);
            return text;
        },
        renderResult(result, options, theme, context) {
            return baseDef.renderResult!(result as any, options, theme, context);
        }
    });
}

export function createAiderReplaceLinesTool(): AgentTool<typeof aiderReplaceLinesSchema> {
    return wrapToolDefinition({
        name: "replace_lines",
        label: "replace_lines",
        description: "Create or update one or more files (Aider format).",
        promptSnippet: "Replace lines in a file",
        promptGuidelines: [],
        parameters: aiderReplaceLinesSchema,
        async execute(toolCallId, args, signal, onUpdate, ctx) {
            const output = await handleAiderReplaceLines(args);
            return { content: [{ type: "text", text: output }], details: {} };
        },
        renderCall(args, theme, context) {
            const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
            const files = args.edits.map((e: any) => e.path).join(", ");
            text.setText(`${theme.fg("toolTitle", theme.bold("replace_lines"))} in ${files}`);
            return text;
        },
        renderResult(result, options, theme, context) {
            const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
            text.setText("Lines replaced.");
            return text;
        }
    });
}

import { openInterpreterExecuteSchema } from "./clean-room-schemas.js";
import { handleOpenInterpreterExecute } from "./clean-room-handlers.js";

export function createOpenInterpreterExecuteTool(): AgentTool<typeof openInterpreterExecuteSchema> {
    return wrapToolDefinition({
        name: "execute_code",
        label: "execute_code",
        description: "Execute code in a specific language (Open Interpreter format).",
        promptSnippet: "Execute code",
        promptGuidelines: [],
        parameters: openInterpreterExecuteSchema,
        async execute(toolCallId, args, signal, onUpdate, ctx) {
            const output = await handleOpenInterpreterExecute(args);
            return { content: [{ type: "text", text: output }], details: {} };
        },
        renderCall(args, theme, context) {
            const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
            text.setText(`${theme.fg("toolTitle", theme.bold("execute_code"))} (${args.language})`);
            return text;
        },
        renderResult(result, options, theme, context) {
            const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
            text.setText("Code executed.");
            return text;
        }
    });
}
