import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import AppNew from "./AppNew";
import ErrorBoundary from "./components/ErrorBoundary";
import { ToastProvider } from "./components/Toast";
import "./index.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ErrorBoundary>
      <ToastProvider>
        <AppNew />
      </ToastProvider>
    </ErrorBoundary>
  </StrictMode>
);
