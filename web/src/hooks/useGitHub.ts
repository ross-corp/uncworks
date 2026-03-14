const API_BASE_URL = import.meta.env.VITE_API_URL ?? "";

export function useGitHub() {
  async function pullSpec(
    repo: string,
    path: string,
  ): Promise<{ content: string; sha: string }> {
    const params = new URLSearchParams({ repo, path });
    const res = await fetch(
      `${API_BASE_URL}/api/v1/specs/pull?${params.toString()}`,
    );
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`Failed to pull spec: ${text}`);
    }
    return res.json();
  }

  async function pushSpec(
    repo: string,
    path: string,
    content: string,
    message: string,
  ): Promise<{ sha: string }> {
    const res = await fetch(`${API_BASE_URL}/api/v1/specs/push`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ repo, path, content, message }),
    });
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`Failed to push spec: ${text}`);
    }
    return res.json();
  }

  return { pullSpec, pushSpec };
}
