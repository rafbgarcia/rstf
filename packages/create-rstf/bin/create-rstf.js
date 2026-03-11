#!/usr/bin/env node

const { spawnSync } = require("node:child_process");
const path = require("node:path");
const readline = require("node:readline/promises");
const { stdin, stdout, stderr, exit, argv, env } = require("node:process");
const { createRequire } = require("node:module");

const requireFromHere = createRequire(__filename);

function resolveCLIEntry() {
  try {
    return requireFromHere.resolve("@rstf/cli/bin/rstf.js");
  } catch {
    return path.resolve(__dirname, "..", "..", "cli", "bin", "rstf.js");
  }
}

function detectPackageManager() {
  const agent = env.npm_config_user_agent || "";
  if (agent.startsWith("pnpm/")) {
    return "pnpm";
  }
  if (agent.startsWith("bun/")) {
    return "bun";
  }
  if (agent.startsWith("yarn/")) {
    return "yarn";
  }
  return "npm";
}

async function resolveNameAndArgs(rawArgs) {
  const passthrough = [];
  let name = "";

  for (let i = 0; i < rawArgs.length; i++) {
    const arg = rawArgs[i];
    if (!name && !arg.startsWith("-")) {
      name = arg;
      continue;
    }
    passthrough.push(arg);
  }

  if (name) {
    return { name, passthrough };
  }

  if (!stdout.isTTY) {
    throw new Error("project name is required");
  }

  const rl = readline.createInterface({ input: stdin, output: stdout });
  try {
    const answer = (await rl.question("Project name: ")).trim();
    if (!answer) {
      throw new Error("project name is required");
    }
    return { name: answer, passthrough };
  } finally {
    rl.close();
  }
}

async function main() {
  const { name, passthrough } = await resolveNameAndArgs(argv.slice(2));
  const cliEntry = resolveCLIEntry();
  const result = spawnSync(process.execPath, [cliEntry, "init", name, ...passthrough], {
    stdio: "inherit",
  });

  if (result.status !== 0) {
    exit(result.status ?? 1);
  }

  const packageManager = detectPackageManager();
  const devCommand = packageManager === "npm" ? "npm run dev" : `${packageManager} dev`;
  stderr.write(`\n  Next ............. cd ${name} && ${devCommand}\n`);
}

main().catch((error) => {
  stderr.write(`${error.message}\n`);
  exit(1);
});
