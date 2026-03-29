import { useState, useEffect } from "react";
import { useFiles } from "../hooks/useFiles";
import FileTree from "./FileTree";
import FilePreview from "./FilePreview";
import type { AgentRunPhase } from "../types/agent-run";

export default function FileExplorer({ runId, phase }: { runId: string; phase?: AgentRunPhase }) {
  const { readFile } = useFiles();
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [fileContent, setFileContent] = useState<string | null>(null);
  const [loadingFile, setLoadingFile] = useState(false);
  const [fileError, setFileError] = useState<string | null>(null);

  // Reset file selection when navigating to a different run
  useEffect(() => {
    setSelectedPath(null);
    setFileContent(null);
    setFileError(null);
    setLoadingFile(false);
  }, [runId]);

  async function handleSelectFile(path: string) {
    setSelectedPath(path);
    setFileContent(null);
    setFileError(null);
    setLoadingFile(true);
    try {
      const content = await readFile(runId, path);
      setFileContent(content);
    } catch (err) {
      setFileError((err as Error).message);
    } finally {
      setLoadingFile(false);
    }
  }

  return (
    <div className="flex h-full">
      {/* File tree */}
      <div className="w-[250px] shrink-0 overflow-y-auto overscroll-none border-r border-border bg-background">
        <FileTree runId={runId} phase={phase} onSelectFile={handleSelectFile} />
      </div>

      {/* File preview */}
      <div className="flex-1 overflow-hidden bg-card">
        {loadingFile ? (
          <div className="flex h-full items-center justify-center text-sm text-muted-foreground/60">
            Loading file...
          </div>
        ) : fileError ? (
          <div className="flex h-full items-center justify-center text-sm text-destructive">
            {fileError}
          </div>
        ) : selectedPath && fileContent !== null ? (
          <FilePreview path={selectedPath} content={fileContent} />
        ) : (
          <div className="flex h-full items-center justify-center text-sm text-muted-foreground/60">
            Select a file to preview
          </div>
        )}
      </div>
    </div>
  );
}
