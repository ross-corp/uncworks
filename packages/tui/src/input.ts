/**
 * Input parser — translates raw terminal key sequences into named actions.
 */

export type InputAction =
  | { type: "up" }
  | { type: "down" }
  | { type: "enter" }
  | { type: "escape" }
  | { type: "quit" }
  | { type: "backspace" }
  | { type: "char"; char: string };

/** Parse a raw key string into a named action. */
export function parseInput(key: string): InputAction | null {
  switch (key) {
    case "\x1b[A": // Arrow Up
      return { type: "up" };
    case "\x1b[B": // Arrow Down
      return { type: "down" };
    case "\r": // Enter
    case "\n":
      return { type: "enter" };
    case "\x1b": // Escape (standalone)
      return { type: "escape" };
    case "q":
    case "Q":
      return { type: "quit" };
    case "\x7f": // Backspace (most terminals)
    case "\b": // Backspace (alternative)
      return { type: "backspace" };
    default:
      // Printable ASCII characters (space through tilde)
      if (key.length === 1 && key >= " " && key <= "~") {
        return { type: "char", char: key };
      }
      return null;
  }
}
