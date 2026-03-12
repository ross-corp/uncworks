import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { parseInput } from "./input.js";

describe("Input Parser", () => {
  it("parses arrow up", () => {
    const action = parseInput("\x1b[A");
    assert.deepEqual(action, { type: "up" });
  });

  it("parses arrow down", () => {
    const action = parseInput("\x1b[B");
    assert.deepEqual(action, { type: "down" });
  });

  it("parses enter (\\r)", () => {
    const action = parseInput("\r");
    assert.deepEqual(action, { type: "enter" });
  });

  it("parses enter (\\n)", () => {
    const action = parseInput("\n");
    assert.deepEqual(action, { type: "enter" });
  });

  it("parses escape", () => {
    const action = parseInput("\x1b");
    assert.deepEqual(action, { type: "escape" });
  });

  it("parses quit (q)", () => {
    const action = parseInput("q");
    assert.deepEqual(action, { type: "quit" });
  });

  it("parses quit (Q)", () => {
    const action = parseInput("Q");
    assert.deepEqual(action, { type: "quit" });
  });

  it("parses backspace (0x7f)", () => {
    const action = parseInput("\x7f");
    assert.deepEqual(action, { type: "backspace" });
  });

  it("parses backspace (0x08)", () => {
    const action = parseInput("\b");
    assert.deepEqual(action, { type: "backspace" });
  });

  it("parses printable characters", () => {
    const action = parseInput("a");
    assert.deepEqual(action, { type: "char", char: "a" });
  });

  it("parses space as char", () => {
    const action = parseInput(" ");
    assert.deepEqual(action, { type: "char", char: " " });
  });

  it("returns null for unknown sequences", () => {
    const action = parseInput("\x1b[C"); // Arrow right — not handled
    assert.equal(action, null);
  });

  it("returns null for multi-byte unknown input", () => {
    const action = parseInput("\x1b[1;5A"); // Ctrl+Up
    assert.equal(action, null);
  });
});
