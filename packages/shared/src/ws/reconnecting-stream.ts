/**
 * ReconnectingStream provides automatic reconnection with exponential
 * backoff, jitter, and subscription restoration for streaming connections.
 *
 * Transport-agnostic — wraps any async connect/subscribe function
 * (ConnectRPC server-streaming, WebSocket, SSE, etc.).
 */

export interface ReconnectingStreamOptions {
  /** Base delay in ms for exponential backoff (default: 1000). */
  baseDelay?: number;
  /** Maximum delay in ms (default: 30000). */
  maxDelay?: number;
  /** Maximum jitter in ms added to each delay (default: 1000). */
  maxJitter?: number;
  /** Maximum consecutive reconnection attempts before giving up (default: 10). */
  maxRetries?: number;
}

export type ConnectionState = "connecting" | "connected" | "disconnected" | "failed";

export class ReconnectingStream {
  private baseDelay: number;
  private maxDelay: number;
  private maxJitter: number;
  private maxRetries: number;

  private attempt = 0;
  private state: ConnectionState = "disconnected";
  private subscriptions = new Set<string>();
  private abortController: AbortController | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;

  private connectFn: ((signal: AbortSignal) => Promise<void>) | null = null;
  private subscribeFn: ((id: string) => void) | null = null;

  private onStateChange: (state: ConnectionState) => void = () => {};
  private onConnectionFailed: () => void = () => {};
  private onMessage: (data: unknown) => void = () => {};

  constructor(options: ReconnectingStreamOptions = {}) {
    this.baseDelay = options.baseDelay ?? 1000;
    this.maxDelay = options.maxDelay ?? 30000;
    this.maxJitter = options.maxJitter ?? 1000;
    this.maxRetries = options.maxRetries ?? 10;
  }

  /** Set the connect function that establishes the streaming connection. */
  setConnectFn(fn: (signal: AbortSignal) => Promise<void>): void {
    this.connectFn = fn;
  }

  /** Set the function called to re-subscribe to a topic after reconnect. */
  setSubscribeFn(fn: (id: string) => void): void {
    this.subscribeFn = fn;
  }

  /** Register a handler for connection state changes. */
  onStateChanged(handler: (state: ConnectionState) => void): void {
    this.onStateChange = handler;
  }

  /** Register a handler for when max retries are exceeded. */
  onFailed(handler: () => void): void {
    this.onConnectionFailed = handler;
  }

  /** Register a handler for incoming messages. */
  onMessageReceived(handler: (data: unknown) => void): void {
    this.onMessage = handler;
  }

  /** Track a subscription for restoration on reconnect. */
  subscribe(id: string): void {
    this.subscriptions.add(id);
  }

  /** Remove a tracked subscription. */
  unsubscribe(id: string): void {
    this.subscriptions.delete(id);
  }

  /** Get current active subscriptions. */
  getSubscriptions(): string[] {
    return [...this.subscriptions];
  }

  /** Signal that a message was received (resets backoff). */
  messageReceived(data?: unknown): void {
    this.attempt = 0;
    if (this.onMessage) {
      this.onMessage(data);
    }
  }

  /** Start the connection. */
  async connect(): Promise<void> {
    if (!this.connectFn) throw new Error("connectFn not set");
    this.setState("connecting");
    this.abortController = new AbortController();

    try {
      await this.connectFn(this.abortController.signal);
      this.setState("connected");
      this.attempt = 0;
      this.restoreSubscriptions();
    } catch {
      if (this.abortController?.signal.aborted) return;
      this.handleDisconnect();
    }
  }

  /** Disconnect and stop reconnection attempts. */
  disconnect(): void {
    this.clearReconnectTimer();
    this.abortController?.abort();
    this.abortController = null;
    this.attempt = 0;
    this.setState("disconnected");
  }

  /** Get the current connection state. */
  getState(): ConnectionState {
    return this.state;
  }

  /** Calculate the delay for the current attempt (exposed for testing). */
  getBackoffDelay(): number {
    const exponential = this.baseDelay * Math.pow(2, this.attempt);
    const jitter = Math.random() * this.maxJitter;
    return Math.min(exponential + jitter, this.maxDelay);
  }

  private setState(newState: ConnectionState): void {
    this.state = newState;
    this.onStateChange(newState);
  }

  private handleDisconnect(): void {
    this.attempt++;
    if (this.attempt > this.maxRetries) {
      this.setState("failed");
      this.onConnectionFailed();
      return;
    }

    this.setState("disconnected");
    const delay = this.getBackoffDelay();
    this.reconnectTimer = setTimeout(() => {
      this.connect();
    }, delay);
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  private restoreSubscriptions(): void {
    if (!this.subscribeFn) return;
    for (const id of this.subscriptions) {
      this.subscribeFn(id);
    }
  }
}
