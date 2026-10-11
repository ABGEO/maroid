import { defineRoute } from "@maroid/plugin-sdk";

export const routes = {
  "/attention": defineRoute(() => import("./pages/attention/Attention.svelte")),
  "/plants": defineRoute(() => import("./pages/plants/Plants.svelte")),
  "/environments": defineRoute(
    () => import("./pages/environments/Environments.svelte"),
  ),
  "/maintenance": defineRoute(
    () => import("./pages/maintenance/Maintenance.svelte"),
  ),
};
