import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

export default defineConfig(() => {
    const inDocker = process.env.VITE_IN_DOCKER === "true";
    const apiTarget = inDocker ? "http://backend:8080" : "http://localhost:8080";

    return {
        plugins: [vue()],
        server: {
            host: true,
            port: 5173,
            strictPort: true,
            proxy: {
                "/api": {
                    target: apiTarget,
                    changeOrigin: true,
                    rewrite: (path) => path.replace(/^\/api/, ""),
                },
            },
        },
    };
});
