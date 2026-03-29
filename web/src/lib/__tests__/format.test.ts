// web/src/lib/__tests__/format.test.ts
// Tests for format.ts: aggregatePhase, formatRelative, formatAge.
import { describe, it, expect, vi, afterEach } from "vitest";
import { aggregatePhase, formatRelative, formatAge } from "../format";
import type { AgentRunPhase } from "../../types/agent-run";

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------
function run(phase: AgentRunPhase) {
  return { status: { phase } };
}

// ---------------------------------------------------------------------------
// aggregatePhase
// ---------------------------------------------------------------------------
describe("aggregatePhase", () => {
  it("returns pending for an empty array", () => {
    expect(aggregatePhase([])).toBe("pending");
  });

  it("returns the phase when all runs share the same phase", () => {
    expect(aggregatePhase([run("failed"), run("failed")])).toBe("failed");
    expect(aggregatePhase([run("cancelled"), run("cancelled")])).toBe("cancelled");
    expect(aggregatePhase([run("succeeded"), run("succeeded")])).toBe("succeeded");
  });

  it("running beats all other phases", () => {
    expect(aggregatePhase([run("running"), run("succeeded")])).toBe("running");
    expect(aggregatePhase([run("running"), run("failed")])).toBe("running");
    expect(aggregatePhase([run("running"), run("cancelled")])).toBe("running");
    expect(aggregatePhase([run("running"), run("pending")])).toBe("running");
  });

  it("waiting_for_input beats pending / succeeded / failed / cancelled", () => {
    expect(aggregatePhase([run("waiting_for_input"), run("succeeded")])).toBe("waiting_for_input");
    expect(aggregatePhase([run("waiting_for_input"), run("failed")])).toBe("waiting_for_input");
    expect(aggregatePhase([run("waiting_for_input"), run("pending")])).toBe("waiting_for_input");
  });

  it("running beats waiting_for_input", () => {
    expect(aggregatePhase([run("running"), run("waiting_for_input")])).toBe("running");
  });

  it("pending beats succeeded (in-flight group)", () => {
    expect(aggregatePhase([run("pending"), run("succeeded")])).toBe("pending");
  });

  it("succeeded wins over failed/cancelled mix", () => {
    expect(aggregatePhase([run("succeeded"), run("failed")])).toBe("succeeded");
    expect(aggregatePhase([run("succeeded"), run("cancelled")])).toBe("succeeded");
  });

  it("failed requires all runs to be failed", () => {
    expect(aggregatePhase([run("failed"), run("cancelled")])).toBe("pending");
  });

  it("cancelled requires all runs to be cancelled", () => {
    expect(aggregatePhase([run("cancelled"), run("failed")])).toBe("pending");
  });

  it("single run returns its own phase", () => {
    const phases: AgentRunPhase[] = ["pending", "running", "waiting_for_input", "succeeded", "failed", "cancelled"];
    for (const p of phases) {
      expect(aggregatePhase([run(p)])).toBe(p);
    }
  });
});

// ---------------------------------------------------------------------------
// formatRelative
// ---------------------------------------------------------------------------
describe("formatRelative", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('returns "" for an empty string', () => {
    expect(formatRelative("")).toBe("");
  });

  it('returns "" for an invalid ISO string', () => {
    expect(formatRelative("not-a-date")).toBe("");
    expect(formatRelative("2099-99-99")).toBe("");
  });

  it('returns "overdue" when the timestamp is in the past', () => {
    const past = new Date(Date.now() - 60_000).toISOString();
    expect(formatRelative(past)).toBe("overdue");
  });

  it('returns "in Xs" for < 60 seconds in the future', () => {
    const future = new Date(Date.now() + 30_000).toISOString();
    expect(formatRelative(future)).toBe("in 30s");
  });

  it('returns "in Xm" for < 60 minutes in the future', () => {
    const future = new Date(Date.now() + 5 * 60_000).toISOString();
    expect(formatRelative(future)).toBe("in 5m");
  });

  it('returns "in Xh" for < 24 hours in the future', () => {
    const future = new Date(Date.now() + 3 * 3600_000).toISOString();
    expect(formatRelative(future)).toBe("in 3h");
  });

  it('returns "in Xd" for >= 24 hours in the future', () => {
    const future = new Date(Date.now() + 2 * 86400_000).toISOString();
    expect(formatRelative(future)).toBe("in 2d");
  });
});

// ---------------------------------------------------------------------------
// formatAge
// ---------------------------------------------------------------------------
describe("formatAge", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('returns "" for an empty string', () => {
    expect(formatAge("")).toBe("");
  });

  it('returns "" for an invalid ISO string', () => {
    expect(formatAge("not-a-date")).toBe("");
    expect(formatAge("2099-99-99")).toBe("");
  });

  it('returns "0s" for a future timestamp (clock skew)', () => {
    const future = new Date(Date.now() + 60_000).toISOString();
    expect(formatAge(future)).toBe("0s");
  });

  it('returns "Xs" for < 60 seconds old', () => {
    const past = new Date(Date.now() - 45_000).toISOString();
    expect(formatAge(past)).toBe("45s");
  });

  it('returns "Xm" for < 60 minutes old', () => {
    const past = new Date(Date.now() - 7 * 60_000).toISOString();
    expect(formatAge(past)).toBe("7m");
  });

  it('returns "Xh" for < 24 hours old', () => {
    const past = new Date(Date.now() - 4 * 3600_000).toISOString();
    expect(formatAge(past)).toBe("4h");
  });

  it('returns "Xd" for >= 24 hours old', () => {
    const past = new Date(Date.now() - 3 * 86400_000).toISOString();
    expect(formatAge(past)).toBe("3d");
  });
});
