import { defineRoute } from "@maroid/plugin-sdk";

export const routes = {
  "/account": defineRoute(() => import("./pages/account/Account.svelte")),
  "/transactions": defineRoute(
    () => import("./pages/transactions/Transactions.svelte"),
  ),
};
