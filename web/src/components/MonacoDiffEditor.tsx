import { useRef } from "react";
import { DiffEditor } from "@monaco-editor/react";

// ============================================================
// MonacoDiffEditor — wrapper around @monaco-editor/react's
// DiffEditor with MU-TH-UR dark theme configuration.
// Loaded lazily from DiffViewer.tsx via React.lazy().
// ============================================================

/** Custom MU-TH-UR dark theme definition for Monaco. */
function defineTheme(monaco: typeof import("monaco-editor")) {
  monaco.editor.defineTheme("muthr-dark", {
    base: "vs-dark",
    inherit: true,
    rules: [
      { token: "", foreground: "00ff41" },
      { token: "comment", foreground: "1a3a1a", fontStyle: "italic" },
      { token: "keyword", foreground: "00ff41" },
      { token: "string", foreground: "66ff66" },
      { token: "number", foreground: "33cc33" },
      { token: "type", foreground: "00cc33" },
    ],
    colors: {
      "editor.background": "#0a0a0a",
      "editor.foreground": "#00ff41",
      "editor.lineHighlightBackground": "#1a3a1a20",
      "editor.selectionBackground": "#00ff4130",
      "editorLineNumber.foreground": "#1a3a1a",
      "editorLineNumber.activeForeground": "#00ff41",
      "editorGutter.background": "#0a0a0a",
      "diffEditor.insertedTextBackground": "#00ff4115",
      "diffEditor.removedTextBackground": "#ff660015",
      "diffEditor.insertedLineBackground": "#00ff4110",
      "diffEditor.removedLineBackground": "#ff660010",
      "scrollbarSlider.background": "#1a3a1a80",
      "scrollbarSlider.hoverBackground": "#00ff4140",
      "scrollbarSlider.activeBackground": "#00ff4160",
    },
  });
}

export default function MonacoDiffEditor({
  original,
  modified,
  language,
  inlineMode,
}: {
  original: string;
  modified: string;
  language: string;
  inlineMode: boolean;
}) {
  const themeSet = useRef(false);

  function handleBeforeMount(monaco: typeof import("monaco-editor")) {
    if (!themeSet.current) {
      defineTheme(monaco);
      themeSet.current = true;
    }
  }

  return (
    <DiffEditor
      original={original}
      modified={modified}
      language={language}
      theme="muthr-dark"
      beforeMount={handleBeforeMount}
      options={{
        readOnly: true,
        renderSideBySide: !inlineMode,
        minimap: { enabled: false },
        fontSize: 12,
        fontFamily: "var(--muthr-font)",
        scrollBeyondLastLine: false,
        lineNumbers: "on",
        glyphMargin: false,
        folding: true,
        wordWrap: "off",
        automaticLayout: true,
      }}
      height="100%"
    />
  );
}
