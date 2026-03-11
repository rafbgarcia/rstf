#!/usr/bin/env node

const { existsSync, readFileSync } = require("node:fs");
const { execFileSync, spawn } = require("node:child_process");
const path = require("node:path");

function currentBinaryName() {
  const platformMap = {
    darwin: "darwin",
    linux: "linux",
    win32: "windows",
  };
  const archMap = {
    x64: "amd64",
    arm64: "arm64",
  };

  const goos = platformMap[process.platform];
  const goarch = archMap[process.arch];
  if (!goos || !goarch) {
    throw new Error(`Unsupported platform: ${process.platform}/${process.arch}`);
  }

  return `rstf-${goos}-${goarch}${process.platform === "win32" ? ".exe" : ""}`;
}

function ensureBinary(binaryPath, packageDir) {
  if (existsSync(binaryPath)) {
    return;
  }

  const metadataPath = path.join(packageDir, "dist", "metadata.json");
  if (existsSync(metadataPath)) {
    const metadata = JSON.parse(readFileSync(metadataPath, "utf8"));
    if (!metadata.repoRoot) {
      throw new Error("Missing packaged rstf binary. Reinstall @rstf/cli.");
    }

    execFileSync("go", ["build", "-o", binaryPath, "./cmd/rstf"], {
      cwd: metadata.repoRoot,
      stdio: "inherit",
    });
    return;
  }

  const sourceRepoRoot = path.resolve(packageDir, "..", "..");
  if (existsSync(path.join(sourceRepoRoot, "cmd", "rstf"))) {
    execFileSync("go", ["build", "-o", binaryPath, "./cmd/rstf"], {
      cwd: sourceRepoRoot,
      stdio: "inherit",
    });
    return;
  }

  throw new Error("Missing packaged rstf binary. Reinstall @rstf/cli.");
}

function main() {
  const packageDir = path.resolve(__dirname, "..");
  const binaryPath = path.join(packageDir, "dist", currentBinaryName());
  ensureBinary(binaryPath, packageDir);

  const child = spawn(binaryPath, process.argv.slice(2), {
    stdio: "inherit",
  });

  child.on("exit", (code, signal) => {
    if (signal) {
      process.kill(process.pid, signal);
      return;
    }
    process.exit(code ?? 0);
  });

  child.on("error", (error) => {
    console.error(error.message);
    process.exit(1);
  });
}

main();
