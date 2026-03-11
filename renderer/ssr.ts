import { createServer } from "node:http";
import { existsSync } from "node:fs";
import { lstat, mkdir, symlink } from "node:fs/promises";
import { createRequire } from "node:module";
import path from "node:path";
import { pathToFileURL } from "node:url";

function parseProjectRoot(): string {
  const args = process.argv;
  const idx = args.indexOf("--project-root");
  if (idx === -1 || idx + 1 >= args.length) {
    console.error("Usage: node --import tsx ssr.ts --project-root <path>");
    process.exit(1);
  }
  return path.resolve(args[idx + 1]);
}

const projectRoot = parseProjectRoot();
const nodeRequire = createRequire(import.meta.url);

const moduleCache = new Map<string, any>();
let cacheVersion = 0;
let createElement: any;
let renderToString: any;

async function ensureRSTFAlias(): Promise<void> {
  const scopeDir = path.join(projectRoot, "node_modules", "@rstf");
  const target = path.join(projectRoot, "rstf", "generated");

  await mkdir(path.dirname(scopeDir), { recursive: true });
  try {
    const st = await lstat(scopeDir);
    if (st.isSymbolicLink()) {
      return;
    }
    return;
  } catch {
    // Missing alias dir; create symlink.
  }

  const relTarget = path.relative(path.dirname(scopeDir), target);
  await symlink(relTarget, scopeDir, "dir");
}

function resolveComponentFile(componentPath: string): string {
  const base = path.join(projectRoot, componentPath);
  const candidates = [
    `${base}.tsx`,
    `${base}.ts`,
    `${base}.jsx`,
    `${base}.js`,
    path.join(base, "index.tsx"),
    path.join(base, "index.ts"),
    path.join(base, "index.jsx"),
    path.join(base, "index.js"),
  ];

  for (const filePath of candidates) {
    if (existsSync(filePath)) {
      return filePath;
    }
  }

  throw new Error(`Component not found: ${componentPath}`);
}

async function importVersioned(absPath: string): Promise<any> {
  const href = pathToFileURL(absPath).href;
  return import(`${href}?v=${cacheVersion}`);
}

async function loadComponent(componentPath: string): Promise<Function> {
  let mod = moduleCache.get(componentPath);
  if (!mod) {
    mod = await importVersioned(resolveComponentFile(componentPath));
    moduleCache.set(componentPath, mod);
  }

  if (typeof mod.View !== "function") {
    throw new Error(`Component does not export View: ${componentPath}`);
  }

  return mod.View;
}

interface RenderRequest {
  component: string;
  layout: string;
  ssrProps?: Record<string, Record<string, any>>;
}

function writeJSON(res: any, status: number, payload: unknown): void {
  const body = JSON.stringify(payload);
  res.statusCode = status;
  res.setHeader("Content-Type", "application/json");
  res.end(body);
}

const server = createServer(async (req, res) => {
  const method = req.method || "";
  const url = req.url || "";

  if (method === "POST" && url === "/invalidate") {
    moduleCache.clear();
    cacheVersion++;
    writeJSON(res, 200, { ok: true });
    return;
  }

  if (method === "POST" && url === "/render") {
    try {
      const chunks: Buffer[] = [];
      for await (const chunk of req) {
        chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
      }

      const body = JSON.parse(Buffer.concat(chunks).toString("utf8")) as RenderRequest;
      const ssrRuntimePath = path.join(projectRoot, "rstf", "generated", "ssr.ts");
      const { SSRDataProvider } = await importVersioned(ssrRuntimePath);
      const Layout = await loadComponent(body.layout);
      const Route = await loadComponent(body.component);

      const html = renderToString(
        createElement(
          SSRDataProvider,
          { data: body.ssrProps ?? {} },
          createElement(Layout, null, createElement(Route))
        )
      );
      writeJSON(res, 200, { html });
      return;
    } catch (e: any) {
      writeJSON(res, 500, { error: e?.message || String(e) });
      return;
    }
  }

  writeJSON(res, 404, { error: "not found" });
});

async function main(): Promise<void> {
  await ensureRSTFAlias();

  const reactPath = nodeRequire.resolve("react", { paths: [projectRoot] });
  const reactDomServerPath = nodeRequire.resolve("react-dom/server", {
    paths: [projectRoot],
  });
  ({ createElement } = await import(pathToFileURL(reactPath).href));
  ({ renderToString } = await import(pathToFileURL(reactDomServerPath).href));

  server.listen(0, "127.0.0.1", () => {
    const addr = server.address();
    if (!addr || typeof addr === "string") {
      console.error("failed to bind sidecar port");
      process.exit(1);
    }
    console.log(addr.port);
  });
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});

process.on("SIGINT", () => {
  server.closeIdleConnections?.();
  server.closeAllConnections?.();

  const forceExit = setTimeout(() => {
    process.exit(0);
  }, 1500);
  forceExit.unref();

  server.close(() => {
    clearTimeout(forceExit);
    process.exit(0);
  });
});
