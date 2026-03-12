/**
 * Terminal runtime — raw mode management, screen lifecycle, and graceful exit.
 *
 * Enters raw mode on start, hides cursor, clears screen.
 * Restores terminal state on exit (q, Ctrl-C, SIGINT, SIGTERM).
 */

const CURSOR_HIDE = "\x1b[?25l";
const CURSOR_SHOW = "\x1b[?25h";
const SCREEN_CLEAR = "\x1b[2J\x1b[H";

export interface RuntimeOptions {
  stdin?: NodeJS.ReadStream;
  stdout?: NodeJS.WriteStream;
}

export class Runtime {
  private stdin: NodeJS.ReadStream;
  private stdout: NodeJS.WriteStream;
  private running = false;
  private wasRaw = false;
  private onKeyHandler: ((key: string, raw: Buffer) => void) | null = null;
  private cleanedUp = false;

  private boundCleanup = () => this.cleanup();

  constructor(options: RuntimeOptions = {}) {
    this.stdin = options.stdin ?? process.stdin;
    this.stdout = options.stdout ?? process.stdout;
  }

  /** Register a handler for keypress data. */
  onKey(handler: (key: string, raw: Buffer) => void): void {
    this.onKeyHandler = handler;
  }

  /** Enter raw mode, hide cursor, clear screen. */
  start(): void {
    if (this.running) return;
    this.running = true;
    this.cleanedUp = false;

    // Save raw mode state and enable it
    this.wasRaw = this.stdin.isRaw ?? false;
    if (this.stdin.isTTY) {
      this.stdin.setRawMode(true);
    }
    this.stdin.resume();
    this.stdin.setEncoding("utf8");

    // Hide cursor and clear screen
    this.stdout.write(CURSOR_HIDE);
    this.stdout.write(SCREEN_CLEAR);

    // Listen for input
    this.stdin.on("data", this.handleData);

    // Graceful exit handlers
    process.on("SIGINT", this.boundCleanup);
    process.on("SIGTERM", this.boundCleanup);
  }

  /** Restore terminal state and stop. */
  stop(): void {
    this.cleanup();
  }

  /** Whether the runtime is currently active. */
  isRunning(): boolean {
    return this.running;
  }

  /** Write raw content to stdout. */
  write(content: string): void {
    this.stdout.write(content);
  }

  /** Clear screen. */
  clearScreen(): void {
    this.stdout.write(SCREEN_CLEAR);
  }

  private handleData = (data: string | Buffer): void => {
    const raw = Buffer.isBuffer(data) ? data : Buffer.from(data, "utf8");
    const str = raw.toString("utf8");

    // Ctrl-C: always exit
    if (str === "\x03") {
      this.cleanup();
      return;
    }

    if (this.onKeyHandler) {
      this.onKeyHandler(str, raw);
    }
  };

  private cleanup(): void {
    if (this.cleanedUp) return;
    this.cleanedUp = true;
    this.running = false;

    this.stdin.removeListener("data", this.handleData);
    process.removeListener("SIGINT", this.boundCleanup);
    process.removeListener("SIGTERM", this.boundCleanup);

    // Restore terminal state
    if (this.stdin.isTTY) {
      this.stdin.setRawMode(this.wasRaw);
    }
    this.stdin.pause();

    // Show cursor and clear screen
    this.stdout.write(SCREEN_CLEAR);
    this.stdout.write(CURSOR_SHOW);
  }
}
