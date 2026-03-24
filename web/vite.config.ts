import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "path";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes("node_modules")) {
            return undefined;
          }

          if (
            id.includes("/react/") ||
            id.includes("/react-dom/") ||
            id.includes("/scheduler/")
          ) {
            return "react-vendor";
          }

          if (id.includes("/react-router/")) {
            return "router-vendor";
          }

          if (id.includes("/@tanstack/react-query/") || id.includes("/pocketbase/")) {
            return "data-vendor";
          }

          if (id.includes("/recharts/")) {
            return "charts-vendor";
          }

          if (id.includes("/@xterm/")) {
            return "terminal-vendor";
          }

          if (
            id.includes("/lucide-react/") ||
            id.includes("/radix-ui/") ||
            id.includes("/cmdk/") ||
            id.includes("/sonner/")
          ) {
            return "ui-vendor";
          }

          return "vendor";
        },
      },
    },
  },
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8090",
        changeOrigin: true,
        ws: true,
      },
    },
  },
});
