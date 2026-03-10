import { routes } from "@rstf/routes";

routes.url("get-vs-ssr");
routes.url("users.$id", { id: 1 });

// @ts-expect-error missing path params
routes.url("users.$id");

// @ts-expect-error static routes do not accept params
routes.url("get-vs-ssr", { id: 1 });

// @ts-expect-error unknown route ids should fail at compile time
routes.url("users.$missing", { missing: 1 });
