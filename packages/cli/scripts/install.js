#!/usr/bin/env node

const { chmodSync, existsSync, mkdirSync, readFileSync, writeFileSync } = require("node:fs");
const path = require("node:path");

const supportedPlatforms = new Map([
  ["darwin", "darwin"],
  ["linux", "linux"],
]);

const supportedArchitectures = new Map([
  ["arm64", "arm64"],
  ["x64", "amd64"],
]);

async function main() {
  if (process.env.RSTF_CLI_SKIP_INSTALL === "1" || process.env.RSTF_CLI_LOCAL_BINARY || process.env.RSTF_CLI_LOCAL_SOURCE) {
    return;
  }

  const pkg = JSON.parse(readFileSync(path.join(__dirname, "..", "package.json"), "utf8"));
  const platform = supportedPlatforms.get(process.platform);
  const arch = supportedArchitectures.get(process.arch);
  if (!platform || !arch) {
    throw new Error(`unsupported platform ${process.platform}/${process.arch}; rstf currently supports macOS and Linux on x64 and arm64`);
  }

  const assetName = `rstf-${platform}-${arch}`;
  const baseURL =
    process.env.RSTF_CLI_DOWNLOAD_BASE_URL ||
    `https://github.com/${pkg.rstf.repo}/releases/download/${pkg.rstf.goVersion}`;
  const binaryURL = `${baseURL}/${assetName}`;
  const vendorDir = path.join(__dirname, "vendor", process.platform, process.arch);
  const binaryPath = path.join(vendorDir, "rstf");

  if (existsSync(binaryPath)) {
    return;
  }

  mkdirSync(vendorDir, { recursive: true });

  const response = await fetch(binaryURL);
  if (!response.ok) {
    throw new Error(`download failed from ${binaryURL}: ${response.status} ${response.statusText}`);
  }

  const data = Buffer.from(await response.arrayBuffer());
  writeFileSync(binaryPath, data);
  chmodSync(binaryPath, 0o755);
}

main().catch((error) => {
  console.error(`rstf: failed to install CLI binary: ${error.message}`);
  process.exit(1);
});
