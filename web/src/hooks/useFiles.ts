const API_BASE = import.meta.env.VITE_API_URL ?? "";

export interface FileEntry {
  name: string;
  type: "file" | "directory";
  size: number;
  modified: string;
}

export function useFiles() {
  async function listDir(
    runId: string,
    path: string
  ): Promise<{ entries: FileEntry[] }> {
    const res = await fetch(
      `${API_BASE}/api/v1/runs/${runId}/files?path=${encodeURIComponent(path)}`
    );
    if (!res.ok) throw new Error(`listDir failed: ${res.status}`);
    return res.json();
  }

  async function readFile(runId: string, path: string): Promise<string> {
    const res = await fetch(
      `${API_BASE}/api/v1/runs/${runId}/files/content?path=${encodeURIComponent(path)}`
    );
    if (!res.ok) throw new Error(`readFile failed: ${res.status}`);
    return res.text();
  }

  return { listDir, readFile };
}
