import { routes } from "@rstf/routes";

routes["get-vs-ssr"].url();
routes["users._id"].url({ id: "1" });

// @ts-expect-error missing path params
routes["users._id"].url({});
// @ts-expect-error missing path params
routes["users._id"].url();

// @ts-expect-error unknown param
routes["users._id"].url({ id: "1", notARouteParam: "2" });

// @ts-expect-error params must be strings
routes["users._id"].url({ id: 1 });

// @ts-expect-error static routes do not accept params
routes["get-vs-ssr"].url({ id: "1" });

// @ts-expect-error unknown route ids should fail at compile time
routes["users._missing"].url({ missing: "1" });
