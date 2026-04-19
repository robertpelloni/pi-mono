import { wrapToolDefinition } from "./tool-definition-wrapper.js";
import type { AgentTool } from "@mariozechner/pi-agent-core";
import { Text } from "@mariozechner/pi-tui";

import { openInterpreterComputerUseSchema } from "./clean-room-schemas.js";
import { handleOpenInterpreterComputerUse } from "./clean-room-handlers.js";

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
