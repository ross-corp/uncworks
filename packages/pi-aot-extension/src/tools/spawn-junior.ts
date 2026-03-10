import type { ToolDefinition, ToolResult } from "../extension";

/** Creates the spawn_junior tool for multi-agent collaboration. */
export function SpawnJuniorTool(
  onSpawn: (task: string, context: string) => Promise<string>
): ToolDefinition {
  return {
    name: "spawn_junior",
    description:
      "Spawn a junior agent to handle a sub-task. Returns the junior agent's run ID.",
    parameters: {
      type: "object",
      properties: {
        task: {
          type: "string",
          description: "The task description for the junior agent",
        },
        context: {
          type: "string",
          description: "Additional context from the senior agent",
        },
      },
      required: ["task"],
    },
    execute: async (params): Promise<ToolResult> => {
      const task = params.task as string;
      const context = (params.context as string) || "";

      if (!task) {
        return { success: false, output: "", error: "task is required" };
      }

      try {
        const juniorRunId = await onSpawn(task, context);
        return { success: true, output: juniorRunId };
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        return { success: false, output: "", error: message };
      }
    },
  };
}
