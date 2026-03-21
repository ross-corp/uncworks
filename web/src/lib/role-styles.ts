export const ROLE_STYLES = {
  manage: {
    label: "manage",
    text: "text-[hsl(var(--role-manage))]",
    bg: "bg-[hsl(var(--role-manage)/0.1)]",
    border: "border-[hsl(var(--role-manage)/0.4)]",
    dot: "bg-[hsl(var(--role-manage))]",
  },
  implement: {
    label: "implement",
    text: "text-[hsl(var(--role-implement))]",
    bg: "bg-[hsl(var(--role-implement)/0.1)]",
    border: "border-[hsl(var(--role-implement)/0.4)]",
    dot: "bg-[hsl(var(--role-implement))]",
  },
  system: {
    label: "system",
    text: "text-[hsl(var(--role-system))]",
    bg: "bg-[hsl(var(--role-system)/0.1)]",
    border: "border-[hsl(var(--role-system)/0.4)]",
    dot: "bg-[hsl(var(--role-system))]",
  },
  user: {
    label: "user",
    text: "text-[hsl(var(--role-user))]",
    bg: "bg-[hsl(var(--role-user)/0.1)]",
    border: "border-[hsl(var(--role-user)/0.4)]",
    dot: "bg-[hsl(var(--role-user))]",
  },
  delegate: {
    label: "delegate",
    text: "text-[hsl(var(--role-delegate))]",
    bg: "bg-[hsl(var(--role-delegate)/0.1)]",
    border: "border-[hsl(var(--role-delegate)/0.4)]",
    dot: "bg-[hsl(var(--role-delegate))]",
  },
  error: {
    label: "error",
    text: "text-[hsl(var(--role-error))]",
    bg: "bg-[hsl(var(--role-error)/0.1)]",
    border: "border-[hsl(var(--role-error)/0.4)]",
    dot: "bg-[hsl(var(--role-error))]",
  },
} as const;

export type RoleName = keyof typeof ROLE_STYLES;

/** Extract the role from a span name like "manage.thought" → "manage" */
export function roleFromSpanName(name: string): RoleName {
  const prefix = name.split(".")[0];
  if (prefix in ROLE_STYLES) return prefix as RoleName;
  return "system"; // fallback
}
