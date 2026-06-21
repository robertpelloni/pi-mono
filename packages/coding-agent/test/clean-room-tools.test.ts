import { describe, expect, it } from "vitest";
import {
	handleAmpDiff,
	handleAmpReview,
	handleAuggieAsk,
	handleAuggieSearch,
	handleGeminiReplace,
	handleGeminiRunShellCommand,
	handleOpenCodeApplyPatch,
	handleOpenCodeMultiEdit,
} from "../src/core/tools/clean-room-handlers.js";

describe("Clean Room Tools Parity", () => {
	it("handleOpenCodeApplyPatch should return simulated success message", async () => {
		const res = await handleOpenCodeApplyPatch({ patchText: "test" });
		expect(res).toBe("Simulated apply_patch logic in legacy TS layer.");
	});

	it("handleOpenCodeMultiEdit should return simulated success message", async () => {
		const res = await handleOpenCodeMultiEdit({ params: { filePath: "test.ts", edits: [] } });
		expect(res).toBe("Simulated multiedit logic in legacy TS layer.");
	});

	it("handleGeminiRunShellCommand should return simulated success message", async () => {
		const res = await handleGeminiRunShellCommand({ command: "test" });
		expect(res).toBe("Simulated shell command: test");
	});

	it("handleGeminiReplace should return simulated success message", async () => {
		const res = await handleGeminiReplace({ file_path: "test.ts", old_string: "foo", new_string: "bar" });
		expect(res).toBe("Simulated replace in test.ts");
	});

	it("handleAmpDiff should return simulated success message", async () => {
		const res = await handleAmpDiff({ file_path: "test.ts" });
		expect(res).toBe("Amp Code: Reviewed and staged changes for test.ts.");
	});

	it("handleAmpReview should return simulated success message", async () => {
		const res = await handleAmpReview({ diff_id: "123" });
		expect(res).toBe("Amp Code: Smart mode review checked its own work for diff 123.");
	});

	it("handleAuggieSearch should return simulated success message", async () => {
		const res = await handleAuggieSearch({ query: "test-query" });
		expect(res).toBe('Auggie CLI: Indexed and searched context for query: "test-query"');
	});

	it("handleAuggieAsk should return simulated success message", async () => {
		const res = await handleAuggieAsk({ contextQuery: "context", question: "question" });
		expect(res).toBe('Auggie CLI: Searched context for "context" and asked: "question"');
	});
});
