// web/src/AppKeybindingsWrapper.tsx — Thin wrapper that mounts KeybindingsProvider
// (which needs a Router ancestor for useNavigate) and renders the WhichKeyPopup portal.
import type { ReactNode } from "react";
import { KeybindingsProvider, useKeybindings } from "./contexts/KeybindingsContext";
import WhichKeyPopup from "./components/WhichKeyPopup";

function WhichKeyPortal() {
  const { whichKeyVisible, chordPrefix, whichKeyEntries, dismissWhichKey } =
    useKeybindings();

  return (
    <WhichKeyPopup
      visible={whichKeyVisible}
      prefix={chordPrefix}
      entries={whichKeyEntries}
      onDismiss={dismissWhichKey}
    />
  );
}

/**
 * Must be rendered inside BrowserRouter so KeybindingsProvider can call useNavigate.
 * Also renders the WhichKeyPopup portal as a sibling to the main tree.
 */
export default function AppKeybindingsWrapper({ children }: { children: ReactNode }) {
  return (
    // SettingsProvider is mounted at the AppNew root; KeybindingsProvider reads
    // settings via useSettings(). KEYBINDINGS_DEFAULTS are used until settings load.
    <KeybindingsProvider>
      {children}
      <WhichKeyPortal />
    </KeybindingsProvider>
  );
}
