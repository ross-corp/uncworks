/**
 * Terminal renderer abstraction.
 * This module provides a rendering layer that will integrate with OpenTUI
 * (Zig + Yoga Flexbox) when available, falling back to a simple ANSI renderer.
 */

export interface Box {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface RenderNode {
  type: "box" | "text";
  content?: string;
  style?: {
    bold?: boolean;
    color?: string;
    bg?: string;
  };
  children?: RenderNode[];
  box?: Box;
}

const ANSI = {
  reset: "\x1b[0m",
  bold: "\x1b[1m",
  colors: {
    red: "\x1b[31m",
    green: "\x1b[32m",
    yellow: "\x1b[33m",
    blue: "\x1b[34m",
    magenta: "\x1b[35m",
    cyan: "\x1b[36m",
    white: "\x1b[37m",
    gray: "\x1b[90m",
  } as Record<string, string>,
  clear: "\x1b[2J\x1b[H",
  moveTo: (x: number, y: number) => `\x1b[${y};${x}H`,
};

/** Render a tree of nodes to an ANSI string. */
export function renderToString(node: RenderNode): string {
  const lines: string[] = [];
  flattenNode(node, lines, 0);
  return lines.join("\n");
}

function flattenNode(node: RenderNode, lines: string[], indent: number): void {
  if (node.type === "text" && node.content) {
    let line = " ".repeat(indent);
    if (node.style?.bold) line += ANSI.bold;
    if (node.style?.color && ANSI.colors[node.style.color]) {
      line += ANSI.colors[node.style.color];
    }
    line += node.content;
    if (node.style?.bold || node.style?.color) line += ANSI.reset;
    lines.push(line);
  }

  if (node.children) {
    for (const child of node.children) {
      flattenNode(child, lines, indent + (node.type === "box" ? 2 : 0));
    }
  }
}

/** Write a rendered tree to the terminal with screen clear. */
export function renderToTerminal(node: RenderNode, stream: NodeJS.WriteStream = process.stdout): void {
  stream.write(ANSI.clear);
  stream.write(renderToString(node));
  stream.write("\n");
}
