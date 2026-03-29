// web/src/contexts/ModalContext.tsx — Thin modal stack for ui.modal.close dispatch.
// Modals register themselves on mount and deregister on unmount.
// KeybindingsContext calls closeTop() when the ui.modal.close action fires.
import {
  createContext,
  useCallback,
  useContext,
  useRef,
  type ReactNode,
} from "react";

export interface ModalHandle {
  /** Called when ui.modal.close fires and this modal is on top. */
  close: () => void;
}

interface ModalContextValue {
  /** Register a modal on mount. Returns a cleanup function to deregister. */
  register: (handle: ModalHandle) => () => void;
  /** Close the topmost registered modal, if any. */
  closeTop: () => void;
}

const ModalContext = createContext<ModalContextValue>({
  register: () => () => {},
  closeTop: () => {},
});

export function ModalProvider({ children }: { children: ReactNode }) {
  // Stack of modal handles; most-recently-registered is last (top of stack).
  const stackRef = useRef<ModalHandle[]>([]);

  const register = useCallback((handle: ModalHandle): (() => void) => {
    stackRef.current = [...stackRef.current, handle];
    return () => {
      stackRef.current = stackRef.current.filter((h) => h !== handle);
    };
  }, []);

  const closeTop = useCallback(() => {
    const stack = stackRef.current;
    if (stack.length === 0) return;
    stack[stack.length - 1].close();
  }, []);

  return (
    <ModalContext.Provider value={{ register, closeTop }}>
      {children}
    </ModalContext.Provider>
  );
}

/** Access the modal stack context from any component. */
export function useModal() {
  return useContext(ModalContext);
}
