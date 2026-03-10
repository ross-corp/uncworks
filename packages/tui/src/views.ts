import type { RenderNode } from "./renderer";

export interface AgentRunView {
  id: string;
  name: string;
  phase: string;
  backend: string;
  prompt: string;
}

const phaseSymbols: Record<string, string> = {
  Pending: "○",
  Running: "●",
  WaitingForInput: "◎",
  Succeeded: "✓",
  Failed: "✗",
  Cancelled: "⊘",
};

const phaseColors: Record<string, string> = {
  Pending: "yellow",
  Running: "blue",
  WaitingForInput: "magenta",
  Succeeded: "green",
  Failed: "red",
  Cancelled: "gray",
};

/** Build the header render node. */
export function headerView(): RenderNode {
  return {
    type: "box",
    children: [
      {
        type: "text",
        content: "═══ AOT Dashboard ═══",
        style: { bold: true, color: "cyan" },
      },
      { type: "text", content: "" },
    ],
  };
}

/** Build the agent run list render node. */
export function agentRunListView(
  runs: AgentRunView[],
  selectedIndex: number
): RenderNode {
  if (runs.length === 0) {
    return {
      type: "box",
      children: [
        { type: "text", content: "No agent runs", style: { color: "gray" } },
      ],
    };
  }

  return {
    type: "box",
    children: runs.map((run, i) => ({
      type: "text" as const,
      content: `${i === selectedIndex ? "▸" : " "} ${phaseSymbols[run.phase] || "?"} ${run.name} [${run.phase}] - ${run.prompt.slice(0, 50)}`,
      style: {
        bold: i === selectedIndex,
        color: phaseColors[run.phase] || "white",
      },
    })),
  };
}

/** Build the detail view render node. */
export function agentRunDetailView(run: AgentRunView | null): RenderNode {
  if (!run) {
    return {
      type: "box",
      children: [
        {
          type: "text",
          content: "Select an agent run (↑/↓, Enter)",
          style: { color: "gray" },
        },
      ],
    };
  }

  return {
    type: "box",
    children: [
      { type: "text", content: `Agent: ${run.name}`, style: { bold: true } },
      {
        type: "text",
        content: `Phase: ${phaseSymbols[run.phase] || "?"} ${run.phase}`,
        style: { color: phaseColors[run.phase] },
      },
      { type: "text", content: `Backend: ${run.backend}` },
      { type: "text", content: `Prompt: ${run.prompt}` },
    ],
  };
}

/** Build the full dashboard layout. */
export function dashboardView(
  runs: AgentRunView[],
  selectedIndex: number,
  selectedRun: AgentRunView | null
): RenderNode {
  return {
    type: "box",
    children: [
      headerView(),
      agentRunListView(runs, selectedIndex),
      { type: "text", content: "" },
      { type: "text", content: "─── Detail ───", style: { color: "gray" } },
      agentRunDetailView(selectedRun),
      { type: "text", content: "" },
      {
        type: "text",
        content: "q: quit | ↑/↓: navigate | Enter: select",
        style: { color: "gray" },
      },
    ],
  };
}
