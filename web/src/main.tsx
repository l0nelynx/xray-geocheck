import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import { PrivacyProvider } from "./lib/privacy";
import "./index.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <PrivacyProvider>
      <App />
    </PrivacyProvider>
  </StrictMode>,
);
