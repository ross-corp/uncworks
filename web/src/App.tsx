import { Router, Route } from "@solidjs/router";
import { AOTClient } from "../../packages/shared/src/grpc/client";
import { createAgentStore } from "../../packages/shared/src/store/agent-store";
import RunListPage from "./pages/RunListPage";
import RunDetailPage from "./pages/RunDetailPage";

const API_BASE_URL = import.meta.env.VITE_API_URL ?? "";
const client = new AOTClient({ baseUrl: API_BASE_URL });
const store = createAgentStore();

export default function App() {
  return (
    <main style={{ "max-width": "1100px", margin: "0 auto", padding: "16px" }}>
      <h1 style={{ "margin-bottom": "16px" }}>AOT Dashboard</h1>
      <Router>
        <Route path="/" component={() => <RunListPage client={client} store={store} />} />
        <Route path="/runs/:id" component={() => <RunDetailPage client={client} store={store} />} />
      </Router>
    </main>
  );
}
