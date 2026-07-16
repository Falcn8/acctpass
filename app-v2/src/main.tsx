import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "@fontsource/dm-mono/latin-400.css";
import "@fontsource/dm-mono/latin-500.css";
import "@fontsource/manrope/latin-400.css";
import "@fontsource/manrope/latin-700.css";
import App from "./App";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
