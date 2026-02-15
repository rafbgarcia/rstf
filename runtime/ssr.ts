import { createElement, Fragment } from "react";
import { renderToString } from "react-dom/server";

// Parse --project-root from CLI args
function parseProjectRoot(): string {
  const args = Bun.argv;
  const idx = args.indexOf("--project-root");
  if (idx === -1 || idx + 1 >= args.length) {
    console.error("Usage: bun run ssr.ts --project-root <path>");
    process.exit(1);
  }
  return args[idx + 1];
}

const projectRoot = parseProjectRoot();

// Module cache: component path -> module
const moduleCache = new Map<string, any>();

// Cache of generated modules for __setServerData calls
const generatedModuleCache = new Map<string, any>();

async function loadComponent(componentPath: string): Promise<Function> {
  const fullPath = `${projectRoot}/${componentPath}`;

  let mod = moduleCache.get(fullPath);
  if (!mod) {
    try {
      mod = await import(fullPath);
    } catch (e: any) {
      throw new Error(`Component not found: ${componentPath}`);
    }
    moduleCache.set(fullPath, mod);
  }

  if (typeof mod.View !== "function") {
    throw new Error(`Component does not export View: ${componentPath}`);
  }

  return mod.View;
}

// Resolves generated modules for all serverData keys (async phase).
// Returns pairs of [module, data] ready for synchronous __setServerData calls.
async function resolveGeneratedModules(
  serverData: Record<string, Record<string, any>>
): Promise<Array<[any, Record<string, any>]>> {
  const entries = Object.entries(serverData);
  const results: Array<[any, Record<string, any>]> = [];

  await Promise.all(
    entries.map(async ([componentPath, data]) => {
      let mod = generatedModuleCache.get(componentPath);
      if (!mod) {
        const genPath = `${projectRoot}/.rstf/generated/${componentPath}.ts`;
        try {
          mod = await import(genPath);
          generatedModuleCache.set(componentPath, mod);
        } catch {
          return; // No generated module — component has no .go file
        }
      }
      if (typeof mod.__setServerData === "function") {
        results.push([mod, data]);
      }
    })
  );

  return results;
}

interface RenderRequest {
  component: string;
  layout: string;
  serverData?: Record<string, Record<string, any>>;
}

const server = Bun.serve({
  port: 0,
  routes: {
    "/render": {
      POST: async (req) => {
        try {
          const body = (await req.json()) as RenderRequest;

          // Async phase: resolve all imports before entering the synchronous block.
          const generatedModules = body.serverData
            ? await resolveGeneratedModules(body.serverData)
            : [];
          const Layout = await loadComponent(body.layout);
          const Route = await loadComponent(body.component);

          // Synchronous phase: set data + render. No await here — cannot be
          // interrupted by another request on the single-threaded event loop.
          for (const [mod, data] of generatedModules) {
            mod.__setServerData(data);
          }
          const html = renderToString(
            createElement(Layout, null, createElement(Route))
          );

          return Response.json({ html });
        } catch (e: any) {
          return Response.json({ error: e.message }, { status: 500 });
        }
      },
    },
    "/invalidate": {
      POST: () => {
        moduleCache.clear();
        generatedModuleCache.clear();
        return Response.json({ ok: true });
      },
    },
  },
});

// Print port to stdout so the Go process can read it
console.log(server.port);
