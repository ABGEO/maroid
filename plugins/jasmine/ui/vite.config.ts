import { federation } from "@module-federation/vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [
    federation({
      dts: true,
      dev: { disableDynamicRemoteTypeHints: true, remoteHmr: true },
      name: "dev.maroid.jasmine",
      filename: "remoteEntry.js",
      manifest: true,
      publicPath: "auto",
      exposes: {
        "./entry": "./src/entry.ts",
      },
      remotes: {},
      shared: [],
    }),
    svelte(),
  ],
});
