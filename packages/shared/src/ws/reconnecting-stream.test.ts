import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { ReconnectingStream } from "./reconnecting-stream.js";

describe("ReconnectingStream", () => {
  it("calculates exponential backoff with jitter", () => {
    const stream = new ReconnectingStream({
      baseDelay: 1000,
      maxDelay: 30000,
      maxJitter: 0, // disable jitter for deterministic test
    });

    // attempt 0: 1000 * 2^0 = 1000
    assert.equal(stream.getBackoffDelay(), 1000);
  });

  it("caps backoff at maxDelay", () => {
    const stream = new ReconnectingStream({
      baseDelay: 1000,
      maxDelay: 5000,
      maxJitter: 0,
    });

    // The delay should never exceed maxDelay
    const delay = stream.getBackoffDelay();
    assert.ok(delay <= 5000, `delay ${delay} exceeds maxDelay`);
  });

  it("tracks subscriptions for restoration", () => {
    const stream = new ReconnectingStream();

    stream.subscribe("run-1");
    stream.subscribe("run-2");
    assert.deepEqual(stream.getSubscriptions().sort(), ["run-1", "run-2"]);

    stream.unsubscribe("run-1");
    assert.deepEqual(stream.getSubscriptions(), ["run-2"]);
  });

  it("restores subscriptions on reconnect", async () => {
    const restored: string[] = [];
    let connectCount = 0;

    const stream = new ReconnectingStream({ maxRetries: 3 });

    stream.setConnectFn(async () => {
      connectCount++;
    });

    stream.setSubscribeFn((id: string) => {
      restored.push(id);
    });

    stream.subscribe("run-1");
    stream.subscribe("run-2");

    await stream.connect();

    assert.equal(connectCount, 1);
    assert.deepEqual(restored.sort(), ["run-1", "run-2"]);

    stream.disconnect();
  });

  it("resets backoff on message received", () => {
    const stream = new ReconnectingStream({ maxJitter: 0 });

    stream.messageReceived("data");

    // After message received, attempt resets to 0
    assert.equal(stream.getBackoffDelay(), 1000);
  });

  it("emits connection_failed after max retries", async () => {
    let failedCalled = false;

    const stream = new ReconnectingStream({
      baseDelay: 10,
      maxDelay: 10,
      maxJitter: 0,
      maxRetries: 2,
    });

    stream.onFailed(() => {
      failedCalled = true;
    });

    let attempts = 0;
    stream.setConnectFn(async () => {
      attempts++;
      throw new Error("connection refused");
    });

    await stream.connect();

    // Wait for retries
    await new Promise((resolve) => setTimeout(resolve, 200));

    assert.ok(failedCalled, "connection_failed should have been emitted");
    assert.ok(attempts >= 2, `expected at least 2 attempts, got ${attempts}`);
    assert.equal(stream.getState(), "failed");

    stream.disconnect();
  });

  it("stops reconnecting on disconnect", async () => {
    let attempts = 0;

    const stream = new ReconnectingStream({
      baseDelay: 50,
      maxJitter: 0,
      maxRetries: 10,
    });

    stream.setConnectFn(async () => {
      attempts++;
      throw new Error("fail");
    });

    await stream.connect();
    await new Promise((resolve) => setTimeout(resolve, 100));
    stream.disconnect();

    const attemptsAtDisconnect = attempts;
    await new Promise((resolve) => setTimeout(resolve, 200));

    assert.equal(attempts, attemptsAtDisconnect, "no more attempts after disconnect");
    assert.equal(stream.getState(), "disconnected");
  });
});
