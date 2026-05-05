// web/src/mocks/handlers.ts — Default happy-path MSW handlers for integration tests.
// Import `server` and call `server.use(...)` in individual tests to override for
// error cases or alternate data shapes.
import { http, HttpResponse } from "msw";
import {
  projectFixture,
  rawAgentRunFixture,
} from "./fixtures";

const BASE = "";

export const defaultHandlers = [
  // ── Projects ────────────────────────────────────────────────────────────────
  http.get(`${BASE}/api/v1/projects`, () =>
    HttpResponse.json([projectFixture(), projectFixture({ name: "second-project", displayName: "Second Project" })])
  ),

  http.post(`${BASE}/api/v1/projects`, () =>
    HttpResponse.json({ name: "new-project" }, { status: 201 })
  ),

  http.get(`${BASE}/api/v1/projects/:name`, ({ params }) =>
    HttpResponse.json(projectFixture({ name: String(params.name) }))
  ),

  http.put(`${BASE}/api/v1/projects/:name`, () =>
    HttpResponse.json({})
  ),

  http.delete(`${BASE}/api/v1/projects/:name`, () =>
    new HttpResponse(null, { status: 204 })
  ),

  // ── Runs ─────────────────────────────────────────────────────────────────────
  http.get(`${BASE}/api/v1/runs`, () =>
    HttpResponse.json([
      rawAgentRunFixture({ id: "run-001", name: "ar-run-001", spec: { ...rawAgentRunFixture().spec, projectRef: "my-project" } }),
      rawAgentRunFixture({ id: "run-002", name: "ar-run-002", status: { ...rawAgentRunFixture().status, phase: "Failed" }, spec: { ...rawAgentRunFixture().spec, projectRef: "my-project" } }),
    ])
  ),

  http.get(`${BASE}/api/v1/runs/:name`, ({ params }) =>
    HttpResponse.json(rawAgentRunFixture({ name: String(params.name) }))
  ),

  http.post(`${BASE}/api/v1/runs`, () =>
    HttpResponse.json(rawAgentRunFixture({ id: "new-run", name: "ar-new-run" }), { status: 201 })
  ),

  http.delete(`${BASE}/api/v1/runs/:name`, () =>
    new HttpResponse(null, { status: 204 })
  ),

  // ── Chains (Kubernetes-style { metadata, spec }) ──────────────────────────────
  http.get(`${BASE}/api/v1/chains`, () =>
    HttpResponse.json([{
      metadata: { name: "my-chain", creationTimestamp: "2026-01-01T00:00:00Z" },
      spec: { displayName: "My Chain", steps: [{ name: "step-1", templateRef: "my-template" }], projectRef: "my-project" },
    }])
  ),

  http.get(`${BASE}/api/v1/chains/:name`, ({ params }) =>
    HttpResponse.json({
      metadata: { name: String(params.name), creationTimestamp: "2026-01-01T00:00:00Z" },
      spec: { displayName: "My Chain", steps: [{ name: "step-1", templateRef: "my-template" }] },
    })
  ),

  http.post(`${BASE}/api/v1/chains`, () =>
    HttpResponse.json({ metadata: { name: "new-chain" }, spec: { steps: [] } }, { status: 201 })
  ),

  // ── Chain Runs ────────────────────────────────────────────────────────────────
  http.get(`${BASE}/api/v1/chainruns`, () =>
    HttpResponse.json([{
      metadata: { name: "cr-001", creationTimestamp: "2026-01-01T00:00:00Z" },
      spec: { chainRef: "my-chain" },
      status: { phase: "Succeeded" },
    }])
  ),

  http.get(`${BASE}/api/v1/chainruns/:name`, ({ params }) =>
    HttpResponse.json({
      metadata: { name: params.name, creationTimestamp: "2026-01-01T00:00:00Z" },
      spec: { chainRef: "my-chain" },
      status: { phase: "Succeeded" },
    })
  ),

  // ── Templates (Kubernetes-style) ──────────────────────────────────────────────
  http.get(`${BASE}/api/v1/templates`, () =>
    HttpResponse.json([{
      metadata: { name: "my-template", creationTimestamp: "2026-01-01T00:00:00Z" },
      spec: { displayName: "My Template", projectRef: "my-project", prompt: "Run the tests" },
    }])
  ),

  http.get(`${BASE}/api/v1/templates/:name`, ({ params }) =>
    HttpResponse.json({
      metadata: { name: String(params.name), creationTimestamp: "2026-01-01T00:00:00Z" },
      spec: { displayName: "My Template", projectRef: "my-project", prompt: "Run the tests" },
    })
  ),

  http.post(`${BASE}/api/v1/templates`, () =>
    HttpResponse.json({ metadata: { name: "new-template" }, spec: {} }, { status: 201 })
  ),

  // ── Schedules (Kubernetes-style) ──────────────────────────────────────────────
  http.get(`${BASE}/api/v1/schedules`, () =>
    HttpResponse.json([{
      metadata: { name: "my-schedule", creationTimestamp: "2026-01-01T00:00:00Z" },
      spec: { displayName: "My Schedule", cron: "0 * * * *", chainRef: "my-chain", concurrencyPolicy: "Allow", suspend: false },
    }])
  ),

  http.get(`${BASE}/api/v1/schedules/:name`, ({ params }) =>
    HttpResponse.json({
      metadata: { name: String(params.name), creationTimestamp: "2026-01-01T00:00:00Z" },
      spec: { displayName: "My Schedule", cron: "0 * * * *", chainRef: "my-chain", concurrencyPolicy: "Allow", suspend: false },
    })
  ),

  http.post(`${BASE}/api/v1/schedules`, () =>
    HttpResponse.json({ metadata: { name: "new-schedule" }, spec: {} }, { status: 201 })
  ),

  http.put(`${BASE}/api/v1/schedules/:name`, () =>
    HttpResponse.json({})
  ),

  // ── Features ──────────────────────────────────────────────────────────────────
  http.get(`${BASE}/api/v1/features`, () =>
    HttpResponse.json([{ name: "my-feature", displayName: "My Feature", runCount: 2 }])
  ),

  http.get(`${BASE}/api/v1/features/:name`, ({ params }) =>
    HttpResponse.json({ name: params.name, displayName: "My Feature", runCount: 2 })
  ),

  // ── Chat / Copilot ────────────────────────────────────────────────────────────
  http.post(`${BASE}/api/v1/chat/stream`, () => {
    const encoder = new TextEncoder();
    const stream = new ReadableStream({
      start(controller) {
        controller.enqueue(encoder.encode('data: {"choices":[{"delta":{"content":"Hello"}}]}\n\n'));
        controller.enqueue(encoder.encode('data: {"choices":[{"delta":{"content":" world"}}]}\n\n'));
        controller.enqueue(encoder.encode('data: {"choices":[{"delta":{"content":"!"}}]}\n\n'));
        controller.enqueue(encoder.encode("data: [DONE]\n\n"));
        controller.close();
      },
    });
    return new HttpResponse(stream, {
      headers: { "Content-Type": "text/event-stream", "Cache-Control": "no-cache" },
    });
  }),

  // ── Health ────────────────────────────────────────────────────────────────────
  http.get(`${BASE}/api/v1/health`, () =>
    HttpResponse.json({ status: "ok" })
  ),
];
