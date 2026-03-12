import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { createAppState, handleAction } from "./state.js";

const mockRuns = [
  { id: "ar-1", name: "fix-auth", phase: "Running", backend: "Pod", prompt: "Fix auth" },
  { id: "ar-2", name: "add-tests", phase: "WaitingForInput", backend: "Pod", prompt: "Add tests" },
  { id: "ar-3", name: "deploy", phase: "Succeeded", backend: "Pod", prompt: "Deploy" },
];

describe("State Management", () => {
  it("initializes with empty state", () => {
    const state = createAppState();
    assert.deepEqual(state.runs(), []);
    assert.equal(state.selectedIndex(), 0);
    assert.equal(state.viewMode(), "list");
    assert.equal(state.inputBuffer(), "");
    assert.equal(state.error(), null);
  });

  it("moves selection down", () => {
    const state = createAppState();
    state.setRuns(mockRuns);
    handleAction(state, { type: "down" });
    assert.equal(state.selectedIndex(), 1);
  });

  it("moves selection up", () => {
    const state = createAppState();
    state.setRuns(mockRuns);
    state.setSelectedIndex(2);
    handleAction(state, { type: "up" });
    assert.equal(state.selectedIndex(), 1);
  });

  it("clamps at top", () => {
    const state = createAppState();
    state.setRuns(mockRuns);
    handleAction(state, { type: "up" });
    assert.equal(state.selectedIndex(), 0);
  });

  it("clamps at bottom", () => {
    const state = createAppState();
    state.setRuns(mockRuns);
    state.setSelectedIndex(2);
    handleAction(state, { type: "down" });
    assert.equal(state.selectedIndex(), 2);
  });

  it("quit returns true", () => {
    const state = createAppState();
    const shouldQuit = handleAction(state, { type: "quit" });
    assert.equal(shouldQuit, true);
  });

  it("enter on Running goes to detail mode", () => {
    const state = createAppState();
    state.setRuns(mockRuns);
    state.setSelectedIndex(0); // Running
    handleAction(state, { type: "enter" });
    assert.equal(state.viewMode(), "detail");
  });

  it("enter on WaitingForInput goes to input mode", () => {
    const state = createAppState();
    state.setRuns(mockRuns);
    state.setSelectedIndex(1); // WaitingForInput
    handleAction(state, { type: "enter" });
    assert.equal(state.viewMode(), "input");
    assert.equal(state.inputBuffer(), "");
  });

  it("escape from detail returns to list", () => {
    const state = createAppState();
    state.setViewMode("detail");
    handleAction(state, { type: "escape" });
    assert.equal(state.viewMode(), "list");
  });

  it("escape from input returns to detail and clears buffer", () => {
    const state = createAppState();
    state.setViewMode("input");
    state.setInputBuffer("hello");
    handleAction(state, { type: "escape" });
    assert.equal(state.viewMode(), "detail");
    assert.equal(state.inputBuffer(), "");
  });

  it("typing in input mode appends to buffer", () => {
    const state = createAppState();
    state.setViewMode("input");
    handleAction(state, { type: "char", char: "h" });
    handleAction(state, { type: "char", char: "i" });
    assert.equal(state.inputBuffer(), "hi");
  });

  it("backspace in input mode removes last char", () => {
    const state = createAppState();
    state.setViewMode("input");
    state.setInputBuffer("hello");
    handleAction(state, { type: "backspace" });
    assert.equal(state.inputBuffer(), "hell");
  });

  it("backspace on empty buffer does nothing", () => {
    const state = createAppState();
    state.setViewMode("input");
    handleAction(state, { type: "backspace" });
    assert.equal(state.inputBuffer(), "");
  });

  it("quit is ignored in input mode", () => {
    const state = createAppState();
    state.setViewMode("input");
    const shouldQuit = handleAction(state, { type: "quit" });
    assert.equal(shouldQuit, false);
  });

  it("arrows are ignored in input mode", () => {
    const state = createAppState();
    state.setRuns(mockRuns);
    state.setViewMode("input");
    handleAction(state, { type: "down" });
    assert.equal(state.selectedIndex(), 0); // unchanged
  });
});
