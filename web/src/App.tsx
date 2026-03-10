import { createSignal } from "solid-js";

export default function App() {
  const [count, setCount] = createSignal(0);

  return (
    <main>
      <h1 data-testid="title">AOT Dashboard</h1>
      <p data-testid="status">Status: Ready</p>
      <button data-testid="counter" onClick={() => setCount((c) => c + 1)}>
        Count: {count()}
      </button>
    </main>
  );
}
