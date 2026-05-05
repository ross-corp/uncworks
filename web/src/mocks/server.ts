// web/src/mocks/server.ts — MSW Node server for Vitest integration tests.
// Import and call setupServer in each test file, or wire into vitest setup.
import { setupServer } from "msw/node";
import { defaultHandlers } from "./handlers";

export const server = setupServer(...defaultHandlers);
