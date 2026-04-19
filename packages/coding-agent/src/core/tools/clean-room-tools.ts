import { createBashToolDefinition } from "./bash.js";
import { wrapToolDefinition } from "./tool-definition-wrapper.js";
import { Text } from "@mariozechner/pi-tui";
import type { AgentTool } from "@mariozechner/pi-agent-core";

import { gooseDeveloperShellSchema, gooseFinalOutputSchema } from "./clean-room-schemas.js";

export function createGooseShellTool(cwd: string): AgentTool<typeof gooseDeveloperShellSchema> {
    const baseDef = createBashToolDefinition(cwd);

    return wrapToolDefinition({
        name: "developer__shell",
        label: "developer__shell",
        description: "Run shell commands (Goose format)",
        promptSnippet: "Run a shell command",
        promptGuidelines: [],
        parameters: gooseDeveloperShellSchema,
        async execute(toolCallId, args, signal, onUpdate, ctx) {
            // Note: Goose schema optionally passes 'timeout_secs' which we can ignore for basic parity or implement
            return baseDef.execute(toolCallId, { command: args.command }, signal, onUpdate, ctx);
        },
        renderCall(args, theme, context) {
            const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
            text.setText(`${theme.fg("toolTitle", theme.bold("developer__shell"))} ${args.command}`);
            return text;
        },
        renderResult(result, options, theme, context) {
            return baseDef.renderResult!(result as any, options, theme, context);
        }
    });
}

export function createGooseFinalOutputTool(): AgentTool<typeof gooseFinalOutputSchema> {
    return wrapToolDefinition({
        name: "recipe__final_output",
        label: "recipe__final_output",
        description: "Output the final result for the user (Goose format)",
        promptSnippet: "Output final result",
        promptGuidelines: ["Call this tool when you have finished your task completely."],
        parameters: gooseFinalOutputSchema,
        async execute(toolCallId, args, signal, onUpdate, ctx) {
            return {
                content: [{ type: "text", text: args.message }],
                details: { finalOutput: args.message }
            };
        },
        renderCall(args, theme, context) {
            const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
            text.setText(`${theme.fg("toolTitle", theme.bold("recipe__final_output"))}`);
            return text;
        },
        renderResult(result, options, theme, context) {
            const text = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
            text.setText("Task marked complete.");
            return text;
        }
    });
}
