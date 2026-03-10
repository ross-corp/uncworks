import type { ToolDefinition, ToolResult } from "../extension";

/** Creates the /ask_human tool for HITL workflows. */
export function AskHumanTool(
  onAsk: (question: string) => Promise<string>
): ToolDefinition {
  return {
    name: "ask_human",
    description:
      "Pause the agent and ask a human a question. Use when clarification is needed.",
    parameters: {
      type: "object",
      properties: {
        question: {
          type: "string",
          description: "The question to ask the human",
        },
      },
      required: ["question"],
    },
    execute: async (params): Promise<ToolResult> => {
      const question = params.question as string;
      if (!question) {
        return { success: false, output: "", error: "question is required" };
      }

      try {
        const answer = await onAsk(question);
        return { success: true, output: answer };
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        return { success: false, output: "", error: message };
      }
    },
  };
}
