import { defineRoute } from "@maroid/plugin-sdk";

export const routes = {
  "/plants": defineRoute(() => import("./pages/plants/Plants.svelte")),
  "/environments": defineRoute(
    () => import("./pages/environments/Environments.svelte"),
  ),
};
