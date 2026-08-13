// web/src/lib/__tests__/role-styles.test.ts
// Tests for role-styles.ts: roleFromSpanName, displaySpanName.
import { describe, it, expect } from "vitest";
import { roleFromSpanName, displaySpanName, ROLE_STYLES } from "../role-styles";
import type { RoleName } from "../role-styles";

// ---------------------------------------------------------------------------
// roleFromSpanName
// ---------------------------------------------------------------------------
describe("roleFromSpanName", () => {
  it('returns "system" for an empty string', () => {
    expect(roleFromSpanName("")).toBe("system");
  });

  it("returns a known role when prefix matches a ROLE_STYLES key", () => {
    const knownRoles = Object.keys(ROLE_STYLES) as RoleName[];
    for (const role of knownRoles) {
      expect(roleFromSpanName(`${role}.thought`)).toBe(role);
      expect(roleFromSpanName(role)).toBe(role);
    }
  });

  it("resolves legacy alias unc → manage", () => {
    expect(roleFromSpanName("unc.tool")).toBe("manage");
    expect(roleFromSpanName("unc")).toBe("manage");
  });

  it("resolves legacy alias neph → implement", () => {
    expect(roleFromSpanName("neph.thought")).toBe("implement");
    expect(roleFromSpanName("neph")).toBe("implement");
  });

  it("resolves legacy alias impl → implement", () => {
    expect(roleFromSpanName("impl.execute")).toBe("implement");
  });

  it('falls back to "system" for an unknown prefix', () => {
    expect(roleFromSpanName("unknown.span")).toBe("system");
    expect(roleFromSpanName("random")).toBe("system");
  });

  it("uses the first dot-separated segment only", () => {
    expect(roleFromSpanName("manage.sub.nested")).toBe("manage");
    expect(roleFromSpanName("unc.sub.nested")).toBe("manage");
  });
});

// ---------------------------------------------------------------------------
// displaySpanName
// ---------------------------------------------------------------------------
describe("displaySpanName", () => {
  it('returns "" for an empty string', () => {
    expect(displaySpanName("")).toBe("");
  });

  it("rewrites unc. prefix to manage.", () => {
    expect(displaySpanName("unc.tool")).toBe("manage.tool");
    expect(displaySpanName("unc.thought")).toBe("manage.thought");
  });

  it("rewrites neph. prefix to implement.", () => {
    expect(displaySpanName("neph.execute")).toBe("implement.execute");
  });

  it("rewrites impl. prefix to implement.", () => {
    expect(displaySpanName("impl.execute")).toBe("implement.execute");
  });

  it("leaves modern names unchanged", () => {
    expect(displaySpanName("manage.tool")).toBe("manage.tool");
    expect(displaySpanName("implement.thought")).toBe("implement.thought");
    expect(displaySpanName("system.lifecycle")).toBe("system.lifecycle");
    expect(displaySpanName("user.input")).toBe("user.input");
  });

  it("does not rewrite a prefix that only partially matches", () => {
    // "uncle.foo" should NOT be rewritten because the prefix is "uncle", not "unc"
    expect(displaySpanName("uncle.foo")).toBe("uncle.foo");
  });
});

// ---------------------------------------------------------------------------
// Label vocabulary
//
// This replaces the e2e assertion that "at least one of implement, manage, or
// system is visible", which needed a live run and passed on whichever label
// happened to appear. The label set is a pure mapping, so it belongs here.
// ---------------------------------------------------------------------------
describe("ROLE_STYLES labels", () => {
  it("labels every role with its full semantic name", () => {
    for (const [role, style] of Object.entries(ROLE_STYLES)) {
      expect(style.label).toBe(role);
    }
  });

  it("uses no abbreviation or product codename", () => {
    const banned = ["neph", "unc", "impl", "mgr", "sys"];
    for (const style of Object.values(ROLE_STYLES)) {
      expect(banned).not.toContain(style.label);
    }
  });

  it("covers every role the activity feed can emit", () => {
    for (const role of ["manage", "implement", "system", "user", "delegate"]) {
      expect(ROLE_STYLES).toHaveProperty(role);
    }
  });
});
