#!/usr/bin/env node

const { spawnSync } = require("node:child_process");

const cliEntry = require.resolve("@rstf/cli/bin/rstf.js");
const result = spawnSync(process.execPath, [cliEntry, "init", ...process.argv.slice(2)], {
  cwd: process.cwd(),
  env: process.env,
  stdio: "inherit",
});

process.exit(result.status === null ? 1 : result.status);
