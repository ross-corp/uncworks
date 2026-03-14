import { useState } from "react";
import { useFiles } from "../hooks/useFiles";
import FileTree from "./FileTree";
import FilePreview from "./FilePreview";

export default function FileExplorer({ runId }: { runId: string }) {
  const { readFile } = useFiles();
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [fileContent, setFileContent] = useState<string | null>(null);
  const [loadingFile, setLoadingFile] = useState(false);
  const [fileError, setFileError] = useState<string | null>(null);

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
      <div className="w-[250px] shrink-0 overflow-y-auto border-r border-edge bg-surface-0">
        <FileTree runId={runId} onSelectFile={handleSelectFile} />
      </div>

      {/* File preview */}
      <div className="flex-1 overflow-hidden bg-surface-1">
        {loadingFile ? (
          <div className="flex h-full items-center justify-center text-sm text-txt-tertiary">
            Loading file...
          </div>
        ) : fileError ? (
          <div className="flex h-full items-center justify-center text-sm text-red-400">
            {fileError}
          </div>
        ) : selectedPath && fileContent !== null ? (
          <FilePreview path={selectedPath} content={fileContent} />
        ) : (
          <div className="flex h-full items-center justify-center text-sm text-txt-tertiary">
            Select a file to preview
          </div>
        )}
      </div>
    </div>
  );
}
