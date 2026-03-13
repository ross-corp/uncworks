import { createSignal, Show } from "solid-js";

interface HumanInputFormProps {
  onSubmit: (input: string) => Promise<void>;
}

export default function HumanInputForm(props: HumanInputFormProps) {
  const [input, setInput] = createSignal("");
  const [sending, setSending] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  async function handleSubmit(e: Event) {
    e.preventDefault();
    if (!input().trim() || sending()) return;

    setSending(true);
    setError(null);
    try {
      await props.onSubmit(input().trim());
      setInput("");
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSending(false);
    }
  }

  return (
    <div style={{ "margin-top": "16px", padding: "12px", background: "#f5f3ff", "border-radius": "6px", border: "1px solid #8b5cf6" }}>
      <div style={{ "font-weight": "bold", color: "#6d28d9", "margin-bottom": "8px" }}>Agent is waiting for input</div>
      <Show when={error()}>
        <div style={{ padding: "6px", background: "#fef2f2", "border-radius": "4px", "margin-bottom": "8px", color: "#991b1b", "font-size": "0.85em" }}>{error()}</div>
      </Show>
      <form onSubmit={handleSubmit} style={{ display: "flex", gap: "8px" }}>
        <input
          value={input()}
          onInput={(e) => setInput(e.currentTarget.value)}
          placeholder="Type your response..."
          disabled={sending()}
          style={{ flex: 1, padding: "6px 8px", border: "1px solid #d1d5db", "border-radius": "4px" }}
        />
        <button
          type="submit"
          disabled={sending() || !input().trim()}
          style={{ padding: "6px 16px", background: sending() ? "#9ca3af" : "#6d28d9", color: "white", border: "none", "border-radius": "4px", cursor: sending() ? "default" : "pointer" }}
        >
          {sending() ? "Sending..." : "Send"}
        </button>
      </form>
    </div>
  );
}
