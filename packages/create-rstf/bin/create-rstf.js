#!/usr/bin/env node

const { spawnSync } = require("node:child_process");
const { readFileSync } = require("node:fs");
const path = require("node:path");

const pkg = JSON.parse(readFileSync(path.join(__dirname, "..", "package.json"), "utf8"));
const localBinary = process.env.RSTF_CLI_LOCAL_BINARY;
const localSource = process.env.RSTF_CLI_LOCAL_SOURCE;

if (localBinary) {
  const result = spawnSync(path.resolve(localBinary), ["init", ...process.argv.slice(2)], {
    cwd: process.cwd(),
    env: process.env,
    stdio: "inherit",
  });

  process.exit(result.status === null ? 1 : result.status);
}

const cliTarget = localSource
  ? path.join(path.resolve(localSource), "cmd", "rstf")
  : `${pkg.rstf.goPackage}@${pkg.rstf.goVersion}`;

const result = spawnSync("go", ["run", cliTarget, "init", ...process.argv.slice(2)], {
  cwd: process.cwd(),
  env: process.env,
  stdio: "inherit",
});

if (result.error && result.error.code === "ENOENT") {
  console.error("create-rstf: Go 1.24 is required and `go` was not found on PATH.");
  process.exit(1);
}

process.exit(result.status === null ? 1 : result.status);
