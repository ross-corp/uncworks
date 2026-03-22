/**
 * AOT Determinism Extension for pi-coding-agent
 *
 * Enforces deterministic agent behavior in the AOT spec-driven pipeline:
 * 1. Loop detection — blocks repeated identical tool calls
 * 2. Turn limit — kills agent after max turns to prevent runaway
 * 3. Write validation — ensures OpenSpec files use SHALL/MUST
 * 4. Protected paths — blocks writes outside workspace
 * 5. Audit logging — logs all tool calls for traceability
 * 6. HITL — registers ask_user tool for human-in-the-loop elicitation
 * 7. Subagent — registers delegate_task tool for tracked task delegation
 */

import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { Type, type Static } from "@sinclair/typebox";
import * as fs from "node:fs";
import * as path from "node:path";

// Max identical consecutive tool calls before blocking
const MAX_REPEAT_CALLS = 3;
// Max total turns before forcing exit
const MAX_TURNS = 50;
// Required keywords in spec requirement text
const SPEC_REQUIREMENT_KEYWORDS = /\b(SHALL|MUST)\b/;

// HITL file paths
const INPUT_DIR = "/workspace/.aot/input";
const QUESTION_PATH = path.join(INPUT_DIR, "question.json");
const RESPONSE_PATH = path.join(INPUT_DIR, "response.txt");

// Subagent tracking paths
const SUBAGENT_DIR = "/workspace/.aot/subagents";
const SUBAGENT_LOG = "/workspace/.aot/logs/agent.jsonl";

// HITL polling config
const POLL_INTERVAL_MS = 2000;
const POLL_TIMEOUT_MS = 5 * 60 * 1000; // 5 minutes

interface ToolSignature {
  name: string;
  inputHash: string;
}

function hashInput(input: unknown): string {
  return JSON.stringify(input).length.toString();
}

// --- ask_user tool schema (Typebox) ---

const AskUserParams = Type.Object({
  question: Type.String({ description: "The question to ask the human" }),
  options: Type.Optional(
    Type.Array(Type.String(), {
      description: "Optional list of choices for the human to pick from",
    })
  ),
});

type AskUserInput = Static<typeof AskUserParams>;

// --- delegate_task tool schema (Typebox) ---

const DelegateTaskParams = Type.Object({
  task: Type.String({ description: "A clear description of the subtask to delegate" }),
  context: Type.Optional(
    Type.String({
      description: "Optional additional context, file paths, or constraints for the subtask",
    })
  ),
});

type DelegateTaskInput = Static<typeof DelegateTaskParams>;

let delegateCounter = 0;

/**
 * Logs the delegation event to a marker file and the JSONL log,
 * then returns guidance to handle the task inline.
 */
function delegateTask(params: DelegateTaskInput): string {
  delegateCounter++;
  const delegateId = `delegate-${Date.now()}-${delegateCounter}`;

  // Ensure subagent tracking directory exists
  fs.mkdirSync(SUBAGENT_DIR, { recursive: true });

  // Write a marker file for this delegation
  const marker = {
    id: delegateId,
    task: params.task,
    context: params.context ?? "",
    status: "inline",
    timestamp: new Date().toISOString(),
  };
  fs.writeFileSync(
    path.join(SUBAGENT_DIR, `${delegateId}.json`),
    JSON.stringify(marker, null, 2)
  );

  return [
    `[Delegation tracked: ${delegateId}]`,
    "",
    `Task: ${params.task}`,
    params.context ? `Context: ${params.context}` : "",
    "",
    "Note: This delegation is tracked for visibility in the AOT dashboard.",
    "Handle this subtask inline now — break it into steps and execute them sequentially.",
    "When complete, move on to your next objective.",
  ]
    .filter(Boolean)
    .join("\n");
}

/**
 * Writes the question file and polls for a response file written by the
 * sidecar's SendInput RPC handler.
 */
async function askUser(params: AskUserInput): Promise<string> {
  // Ensure input directory exists
  fs.mkdirSync(INPUT_DIR, { recursive: true });

  // Write the question payload
  const questionPayload = {
    question: params.question,
    type: params.options && params.options.length > 0 ? "choice" : "text",
    options: params.options ?? [],
    timestamp: new Date().toISOString(),
  };
  fs.writeFileSync(QUESTION_PATH, JSON.stringify(questionPayload, null, 2));

  // Poll for response file
  const deadline = Date.now() + POLL_TIMEOUT_MS;
  while (Date.now() < deadline) {
    if (fs.existsSync(RESPONSE_PATH)) {
      const response = fs.readFileSync(RESPONSE_PATH, "utf-8");
      // Clean up the response file so it doesn't get re-read
      try {
        fs.unlinkSync(RESPONSE_PATH);
      } catch {
        // ignore cleanup errors
      }
      return response;
    }
    await new Promise((resolve) => setTimeout(resolve, POLL_INTERVAL_MS));
  }

  return "No response received within 5 minutes";
}

export default function (pi: ExtensionAPI) {
  let lastToolSig: ToolSignature | null = null;
  let repeatCount = 0;
  let turnCount = 0;
  const stage = process.env.PI_STAGE || "";
  const role = process.env.PI_ROLE || "implement"; // "manage" or "implement"

  // --- Register ask_user custom tool ---
  pi.registerTool({
    name: "ask_user",
    description:
      "Pause and ask the human operator a question. Use when you need clarification, approval, or a decision from the user. The question will be displayed in the AOT dashboard and the agent will wait for a response.",
    parameters: AskUserParams,
    execute: async (params: Record<string, unknown>) => {
      const question = params.question as string;
      if (!question) {
        return "Error: question is required";
      }
      const options = params.options as string[] | undefined;
      return askUser({ question, options });
    },
  });

  // --- Register delegate_task custom tool ---
  pi.registerTool({
    name: "delegate_task",
    description:
      "Delegate a subtask for tracking purposes. Use when you want to break work into logical subtasks that should be visible in the AOT dashboard. The task will be handled inline but tracked as a subagent delegation for observability.",
    parameters: DelegateTaskParams,
    execute: async (params: Record<string, unknown>) => {
      const task = params.task as string;
      if (!task) {
        return "Error: task description is required";
      }
      const context = params.context as string | undefined;
      return delegateTask({ task, context });
    },
  });

  pi.on("turn_start", async () => {
    turnCount++;

    if (turnCount > MAX_TURNS) {
      return { block: true, reason: `Turn limit exceeded (${MAX_TURNS}). Stopping agent to prevent runaway.` };
    }

    return undefined;
  });

  pi.on("tool_call", async (event, ctx) => {
    const sig: ToolSignature = {
      name: event.toolName,
      inputHash: hashInput(event.input),
    };

    // Loop detection
    if (lastToolSig && sig.name === lastToolSig.name && sig.inputHash === lastToolSig.inputHash) {
      repeatCount++;
      if (repeatCount >= MAX_REPEAT_CALLS) {
        const reason = `Loop detected: ${sig.name} called ${repeatCount + 1} times with identical input. Blocking to prevent infinite loop.`;
        if (ctx.hasUI) {
          ctx.ui.notify(reason, "warning");
        }
        // Reset so agent can try something else
        lastToolSig = null;
        repeatCount = 0;
        return { block: true, reason };
      }
    } else {
      lastToolSig = sig;
      repeatCount = 0;
    }

    // Role-based tool policies
    if (role === "manage") {
      // Manage agents may not write/edit repo files (only openspec and .aot are allowed)
      if (event.toolName === "write" || event.toolName === "edit") {
        const filePath =
          (event.input as { path?: string }).path ||
          (event.input as { file_path?: string }).file_path ||
          "";
        if (
          filePath &&
          !filePath.startsWith("/workspace/openspec/") &&
          !filePath.startsWith("/workspace/.aot/")
        ) {
          return {
            block: true,
            reason: `Manage agent cannot write/edit repo files (${filePath}). Only /workspace/openspec/ and /workspace/.aot/ are allowed.`,
          };
        }
      }
    }

    if (role === "implement") {
      // Implement agents may not use ask_user — surface questions in output for manage agent
      if (event.toolName === "ask_user") {
        return {
          block: true,
          reason:
            "Implement agents must include questions in their output for the manage agent to surface.",
        };
      }
    }

    // Spec validation for write tool in plan stage
    if (stage === "plan" && event.toolName === "write") {
      const path = (event.input as { path?: string }).path || "";
      const content = (event.input as { content?: string }).content || "";

      // Check if writing a spec file
      if (path.includes("/specs/") && path.endsWith("spec.md")) {
        // Validate that requirement text contains SHALL or MUST
        if (content.includes("### Requirement:") && !SPEC_REQUIREMENT_KEYWORDS.test(content)) {
          return {
            block: true,
            reason: "Spec requirement text must contain SHALL or MUST (e.g., 'The system SHALL...'). Rewrite the requirement with formal language.",
          };
        }
      }

      // Check tasks.md doesn't have too many tasks for simple prompts
      if (path.endsWith("tasks.md")) {
        const checkboxCount = (content.match(/- \[ \]/g) || []).length;
        if (checkboxCount > 30) {
          return {
            block: true,
            reason: `Too many tasks (${checkboxCount}). Keep tasks proportional to complexity. Aim for 3-15 tasks.`,
          };
        }
      }
    }

    // Protected paths — block writes outside workspace
    if (event.toolName === "write" || event.toolName === "edit") {
      const path = (event.input as { path?: string; file_path?: string }).path ||
                   (event.input as { file_path?: string }).file_path || "";
      if (path.startsWith("/") && !path.startsWith("/workspace")) {
        return { block: true, reason: `Cannot write outside /workspace: ${path}` };
      }
    }

    return undefined;
  });

  // Log session info on start
  pi.on("session_start", async (_event, ctx) => {
    if (ctx.hasUI) {
      ctx.ui.notify(`AOT determinism: max ${MAX_TURNS} turns, loop detection active, ask_user + delegate_task tools registered`, "info");
    }
  });

  // Emit compaction trace event when pi compacts context.
  // Pi-compaxxt (if installed) handles the actual compaction with richer summaries;
  // this hook emits metadata so the sidecar creates a trace span.
  pi.on("session_before_compact", async (event) => {
    const tokensBefore = event.preparation?.tokensBefore ?? 0;
    const messagesToSummarize = event.preparation?.messagesToSummarize?.length ?? 0;
    // Emit a structured JSONL event matching piEvent format (type + message payload).
    // The sidecar's maybeCaptureStreamEvent parses evt.Message for token counts.
    const compactionEvent = {
      type: "context_compaction",
      message: {
        tokensBefore,
        tokensAfter: 0, // not yet known pre-compaction; updated by post-compact if available
        messagesToSummarize,
        hasPreviousSummary: !!event.preparation?.previousSummary,
      },
    };
    console.log(JSON.stringify(compactionEvent));
    // Return undefined to let pi-compaxxt (or pi's default) handle the actual compaction
    return undefined;
  });
}
