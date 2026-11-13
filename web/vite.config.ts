import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 5173,
    strictPort: true,
    // Same-origin requests work both locally and through Compose's service DNS.
    proxy: { "/api": process.env.API_PROXY_TARGET ?? "http://localhost:8080" },
  },
});
