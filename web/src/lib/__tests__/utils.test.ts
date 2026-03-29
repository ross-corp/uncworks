// web/src/lib/__tests__/utils.test.ts
// Tests for utils.ts: cn (clsx + tailwind-merge wrapper).
import { describe, it, expect } from "vitest";
import { cn } from "../utils";

describe("cn", () => {
  it("returns an empty string for no arguments", () => {
    expect(cn()).toBe("");
  });

  it("returns a single class unchanged", () => {
    expect(cn("foo")).toBe("foo");
  });

  it("joins multiple class strings", () => {
    expect(cn("foo", "bar")).toBe("foo bar");
  });

  it("ignores undefined and null values", () => {
    expect(cn("foo", undefined, null, "bar")).toBe("foo bar");
  });

  it("ignores falsy conditional values", () => {
    expect(cn("foo", false && "bar", "baz")).toBe("foo baz");
  });

  it("includes truthy conditional values", () => {
    expect(cn("foo", true && "bar")).toBe("foo bar");
  });

  it("merges conflicting tailwind classes (last wins)", () => {
    // tailwind-merge resolves p-2 vs p-4 — last value wins
    expect(cn("p-2", "p-4")).toBe("p-4");
  });

  it("merges conflicting tailwind text color classes", () => {
    expect(cn("text-red-500", "text-blue-500")).toBe("text-blue-500");
  });

  it("supports object syntax from clsx", () => {
    expect(cn({ foo: true, bar: false, baz: true })).toBe("foo baz");
  });

  it("supports array syntax from clsx", () => {
    expect(cn(["foo", "bar"])).toBe("foo bar");
  });
});
