#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");
const { spawnSync } = require("node:child_process");

const repoRoot = path.resolve(__dirname, "..");
const cliPackagePath = path.join(repoRoot, "packages", "cli", "package.json");
const createPackagePath = path.join(repoRoot, "packages", "create-rstf", "package.json");
const releaseGoPath = path.join(repoRoot, "internal", "release", "release.go");
const originalContents = new Map();
let releaseCommitCreated = false;

function usage(exitCode = 0) {
  const out = exitCode === 0 ? process.stdout : process.stderr;
  out.write(
    [
      "Usage: node scripts/release-tag.cjs <version> [--push] [--skip-verify]",
      "",
      "Examples:",
      "  node scripts/release-tag.cjs 0.1.0-alpha.4",
      "  node scripts/release-tag.cjs 0.1.0-alpha.4 --push",
      "",
      "Behavior:",
      "  - requires a clean git worktree",
      "  - updates all release-version sources together",
      "  - verifies the release locally unless --skip-verify is set",
      "  - creates commit 'release: v<version>' and annotated tag 'v<version>'",
      "  - pushes commit and tag only when --push is set",
      "",
    ].join("\n"),
  );
  process.exit(exitCode);
}

function fail(message) {
  console.error(`release-tag: ${message}`);
  process.exit(1);
}

function run(command, args, { cwd = repoRoot, stdio = "pipe", allowFailure = false } = {}) {
  const result = spawnSync(command, args, {
    cwd,
    stdio,
    encoding: "utf8",
  });

  if (result.error) {
    throw result.error;
  }

  if (!allowFailure && result.status !== 0) {
    const stderr = typeof result.stderr === "string" ? result.stderr.trim() : "";
    const stdout = typeof result.stdout === "string" ? result.stdout.trim() : "";
    throw new Error(stderr || stdout || `${command} exited with status ${result.status}`);
  }

  return result;
}

function readJSON(filePath) {
  return JSON.parse(fs.readFileSync(filePath, "utf8"));
}

function writeJSON(filePath, data) {
  fs.writeFileSync(filePath, `${JSON.stringify(data, null, 2)}\n`);
}

function snapshotFile(filePath) {
  if (!originalContents.has(filePath)) {
    originalContents.set(filePath, fs.readFileSync(filePath, "utf8"));
  }
}

function restoreSnapshots() {
  for (const [filePath, contents] of originalContents.entries()) {
    fs.writeFileSync(filePath, contents);
  }
}

function ensureCleanWorktree() {
  const result = run("git", ["status", "--short"]);
  if (result.stdout.trim() !== "") {
    fail("git worktree is not clean; commit or stash changes before cutting a release");
  }
}

function ensureTagDoesNotExist(tag) {
  const local = run("git", ["rev-parse", "-q", "--verify", `refs/tags/${tag}`], {
    allowFailure: true,
  });
  if (local.status === 0) {
    fail(`git tag ${tag} already exists locally`);
  }

  const remote = run("git", ["ls-remote", "--tags", "origin", `refs/tags/${tag}`], {
    allowFailure: true,
  });
  if (remote.status === 0 && remote.stdout.trim() !== "") {
    fail(`git tag ${tag} already exists on origin`);
  }
}

function ensurePublishVersionAvailable(packageName, version) {
  const result = run("npm", ["view", `${packageName}@${version}`, "version", "--json"], {
    allowFailure: true,
  });

  if (result.status === 0) {
    fail(`${packageName}@${version} is already published on npm`);
  }

  const combined = `${result.stdout || ""}\n${result.stderr || ""}`;
  if (!combined.includes("E404")) {
    throw new Error(`failed to verify npm availability for ${packageName}@${version}: ${combined.trim()}`);
  }
}

function updateReleaseGo(contents, version) {
  const moduleVersion = `v${version}`;
  let next = contents.replace(/^(\s*Version\s*=\s*)"[^"]+"/m, `$1"${version}"`);
  next = next.replace(/^(\s*ModuleVersion\s*=\s*)"[^"]+"/m, `$1"${moduleVersion}"`);
  if (next === contents) {
    throw new Error(`no changes applied to ${releaseGoPath}`);
  }
  return next;
}

function verifyVersionShape(version) {
  const semverPattern = /^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/;
  if (!semverPattern.test(version)) {
    fail(`version must look like semver, got '${version}'`);
  }
}

function currentBranchName() {
  const result = run("git", ["rev-parse", "--abbrev-ref", "HEAD"]);
  const branch = result.stdout.trim();
  if (!branch || branch === "HEAD") {
    fail("cannot push from detached HEAD");
  }
  return branch;
}

function updateVersions(version) {
  snapshotFile(cliPackagePath);
  snapshotFile(createPackagePath);
  snapshotFile(releaseGoPath);

  const cliPackage = readJSON(cliPackagePath);
  const createPackage = readJSON(createPackagePath);

  if (cliPackage.version === version) {
    fail(`packages/cli is already at version ${version}`);
  }

  cliPackage.version = version;
  cliPackage.rstf.goVersion = `v${version}`;
  writeJSON(cliPackagePath, cliPackage);

  createPackage.version = version;
  createPackage.dependencies["@rstf/cli"] = version;
  writeJSON(createPackagePath, createPackage);

  const releaseGo = fs.readFileSync(releaseGoPath, "utf8");
  fs.writeFileSync(releaseGoPath, updateReleaseGo(releaseGo, version));
}

function verifyRelease(version) {
  const tag = `v${version}`;
  const cliPackage = readJSON(cliPackagePath);
  const createPackage = readJSON(createPackagePath);
  const goVersion = run("go", ["run", "./cmd/rstf", "version"]).stdout.trim();

  if (cliPackage.rstf.goVersion !== tag) {
    fail(`packages/cli rstf.goVersion is ${cliPackage.rstf.goVersion}; expected ${tag}`);
  }
  if (cliPackage.version !== version) {
    fail(`packages/cli version is ${cliPackage.version}; expected ${version}`);
  }
  if (createPackage.version !== version) {
    fail(`packages/create-rstf version is ${createPackage.version}; expected ${version}`);
  }
  if (createPackage.dependencies["@rstf/cli"] !== version) {
    fail(`create-rstf depends on @rstf/cli ${createPackage.dependencies["@rstf/cli"]}; expected ${version}`);
  }
  if (goVersion !== version) {
    fail(`go CLI version is ${goVersion}; expected ${version}`);
  }

  run("node", ["--test", "packages/cli/scripts/install.test.js"], { stdio: "inherit" });
  run("npm", ["pack", "--dry-run"], {
    cwd: path.join(repoRoot, "packages", "cli"),
    stdio: "inherit",
  });
  run("npm", ["pack", "--dry-run"], {
    cwd: path.join(repoRoot, "packages", "create-rstf"),
    stdio: "inherit",
  });
  run("go", ["test", "./..."], { stdio: "inherit" });
}

function createReleaseCommit(version) {
  const tag = `v${version}`;
  run("git", ["add", releaseGoPath, cliPackagePath, createPackagePath], { stdio: "inherit" });
  run("git", ["commit", "-m", `release: ${tag}`], { stdio: "inherit" });
  run("git", ["tag", "-a", tag, "-m", `release ${tag}`], { stdio: "inherit" });
  releaseCommitCreated = true;
}

function pushRelease(version) {
  const tag = `v${version}`;
  const branch = currentBranchName();
  run("git", ["push", "origin", branch], { stdio: "inherit" });
  run("git", ["push", "origin", tag], { stdio: "inherit" });
}

function main() {
  const args = process.argv.slice(2);
  if (args.length === 0 || args.includes("--help") || args.includes("-h")) {
    usage(0);
  }

  const version = args.find((arg) => !arg.startsWith("-"));
  const flags = new Set(args.filter((arg) => arg.startsWith("-")));
  if (!version) {
    usage(1);
  }

  const unsupportedFlags = [...flags].filter((flag) => !["--push", "--skip-verify"].includes(flag));
  if (unsupportedFlags.length > 0) {
    fail(`unsupported flags: ${unsupportedFlags.join(", ")}`);
  }

  verifyVersionShape(version);
  ensureCleanWorktree();
  ensureTagDoesNotExist(`v${version}`);
  ensurePublishVersionAvailable("@rstf/cli", version);
  ensurePublishVersionAvailable("create-rstf", version);
  updateVersions(version);

  if (!flags.has("--skip-verify")) {
    verifyRelease(version);
  }

  createReleaseCommit(version);

  if (flags.has("--push")) {
    pushRelease(version);
  }

  console.log(`release-tag: prepared release v${version}`);
  if (!flags.has("--push")) {
    console.log("release-tag: commit and tag created locally; push when ready");
  }
}

try {
  main();
} catch (error) {
  if (!releaseCommitCreated && originalContents.size > 0) {
    restoreSnapshots();
  }
  fail(error instanceof Error ? error.message : String(error));
}
