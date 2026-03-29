// web/src/contexts/KeybindingsContext.tsx — Global keybinding provider with
// chord state machine (IDLE / CHORD_PENDING) and action dispatch.
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { useNavigate } from "react-router-dom";
import type { ActionID, KeybindingsConfig, KeyBinding } from "../lib/keybindings/types";
import { KEYBINDINGS_DEFAULTS, ACTION_DESCRIPTIONS } from "../lib/keybindings/types";
import {
  resolveEffectiveBindings,
  resolveChordPrefixes,
  isTextInput,
  normaliseKey,
} from "../lib/keybindings/resolve";
import { useModal } from "./ModalContext";
import { useSettings } from "../hooks/useSettings";

// ── Types ─────────────────────────────────────────────────────────────────────

type ChordState = "IDLE" | "CHORD_PENDING";

export interface WhichKeyEntry {
  key: string;
  action: ActionID;
  description: string;
}

interface KeybindingsContextValue {
  /** Programmatically dispatch an action. */
  dispatch: (action: ActionID) => void;
  /** Current chord prefix being waited on, or null. */
  chordPrefix: string | null;
  /** Whether the which-key popup should be visible. */
  whichKeyVisible: boolean;
  /** Entries to display in the which-key popup. */
  whichKeyEntries: WhichKeyEntry[];
  /** Dismiss the which-key popup and reset chord state. */
  dismissWhichKey: () => void;
  /** Effective bindings (for Settings UI). */
  effectiveBindings: KeyBinding[];
}

const KeybindingsContext = createContext<KeybindingsContextValue>({
  dispatch: () => {},
  chordPrefix: null,
  whichKeyVisible: false,
  whichKeyEntries: [],
  dismissWhichKey: () => {},
  effectiveBindings: [],
});

// ── Provider ──────────────────────────────────────────────────────────────────

export function KeybindingsProvider({ children }: { children: ReactNode }) {
  const navigate = useNavigate();
  const { closeTop } = useModal();
  const { settings } = useSettings();

  // Merge keybinding config from settings with defaults
  const cfg: KeybindingsConfig = {
    ...KEYBINDINGS_DEFAULTS,
    ...settings.keybindings,
  };

  const effectiveBindings = resolveEffectiveBindings(cfg);
  const chordPrefixes = resolveChordPrefixes(effectiveBindings);
  const delayMs = cfg.whichKeyDelayMs > 0 ? cfg.whichKeyDelayMs : 500;

  // ── State ──────────────────────────────────────────────────────────────────

  const [_chordState, setChordState] = useState<ChordState>("IDLE");
  const [chordPrefix, setChordPrefix] = useState<string | null>(null);
  const [whichKeyVisible, setWhichKeyVisible] = useState(false);
  const [whichKeyEntries, setWhichKeyEntries] = useState<WhichKeyEntry[]>([]);

  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Keep mutable refs to avoid stale closure captures inside the keydown handler.
  const chordStateRef = useRef<ChordState>("IDLE");
  const chordPrefixRef = useRef<string | null>(null);

  function syncChordState(state: ChordState) {
    chordStateRef.current = state;
    setChordState(state);
  }
  function syncChordPrefix(prefix: string | null) {
    chordPrefixRef.current = prefix;
    setChordPrefix(prefix);
  }

  // ── Dispatch ───────────────────────────────────────────────────────────────

  const dispatch = useCallback((action: ActionID) => {
    switch (action) {
      case "nav.runs":
        navigate("/");
        break;
      case "nav.projects":
        navigate("/projects");
        break;
      case "nav.settings":
        navigate("/settings");
        break;
      case "run.new":
        navigate("/new");
        break;
      case "ui.modal.close":
        closeTop();
        break;
      case "ui.search.focus": {
        // Try to focus a search input anywhere in the current view.
        const el = document.querySelector<HTMLElement>(
          'input[type="search"], input[placeholder*="earch"], input[placeholder*="ilter"]'
        );
        el?.focus();
        break;
      }
      case "system.reload":
        window.location.reload();
        break;
      default:
        break;
    }
  }, [navigate, closeTop]);

  // ── Chord helpers ──────────────────────────────────────────────────────────

  function clearTimer() {
    if (timerRef.current !== null) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }

  function resetToIdle() {
    clearTimer();
    syncChordState("IDLE");
    syncChordPrefix(null);
    setWhichKeyVisible(false);
    setWhichKeyEntries([]);
  }

  function buildWhichKeyEntries(prefix: string): WhichKeyEntry[] {
    return effectiveBindings
      .filter(({ keys }) => {
        const tokens = keys.split(" ").map((t) => t.trim()).filter(Boolean);
        return tokens.length >= 2 && tokens[0] === prefix;
      })
      .map(({ keys, action }) => {
        const tokens = keys.split(" ").map((t) => t.trim()).filter(Boolean);
        return {
          key: tokens.slice(1).join(" "),
          action,
          description: ACTION_DESCRIPTIONS[action] ?? action,
        };
      });
  }

  // ── Global keydown handler ─────────────────────────────────────────────────

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      // Do not intercept when user is typing in a text field.
      if (isTextInput(e.target)) return;

      const token = normaliseKey(e);
      if (!token) return; // pure modifier key

      if (chordStateRef.current === "IDLE") {
        // Check for full single-key match
        const fullMatch = effectiveBindings.find(
          ({ keys }) => keys.trim() === token
        );
        if (fullMatch) {
          e.preventDefault();
          dispatch(fullMatch.action);
          return;
        }

        // Check for chord prefix
        if (chordPrefixes.has(token)) {
          e.preventDefault();
          clearTimer();
          syncChordPrefix(token);
          syncChordState("CHORD_PENDING");

          // Smart single-match skip: if delayMs === 0 and only one completion,
          // dispatch immediately without waiting for a second key.
          const entries = buildWhichKeyEntries(token);
          if (entries.length === 1 && delayMs === 0) {
            resetToIdle();
            dispatch(entries[0].action);
            return;
          }

          // Start timer to show popup after delay
          timerRef.current = setTimeout(() => {
            const e2 = buildWhichKeyEntries(chordPrefixRef.current ?? token);
            setWhichKeyEntries(e2);
            setWhichKeyVisible(true);
          }, delayMs);
          return;
        }

        // Unrecognised key in IDLE — ignore.
      } else if (chordStateRef.current === "CHORD_PENDING") {
        e.preventDefault();
        clearTimer();
        const prefix = chordPrefixRef.current!;

        if (token === "Escape") {
          resetToIdle();
          return;
        }

        const sequence = `${prefix} ${token}`;
        const match = effectiveBindings.find(({ keys }) => keys.trim() === sequence);
        resetToIdle();
        if (match) {
          dispatch(match.action);
        }
        // Unrecognised chord second key → silently reset.
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [effectiveBindings, chordPrefixes, delayMs, dispatch]
  );

  useEffect(() => {
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  // Cleanup timer on unmount
  useEffect(() => () => clearTimer(), []);

  const dismissWhichKey = useCallback(() => {
    resetToIdle();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <KeybindingsContext.Provider
      value={{
        dispatch,
        chordPrefix,
        whichKeyVisible,
        whichKeyEntries,
        dismissWhichKey,
        effectiveBindings,
      }}
    >
      {children}
    </KeybindingsContext.Provider>
  );
}

/** Access the keybindings context. */
export function useKeybindings() {
  return useContext(KeybindingsContext);
}
