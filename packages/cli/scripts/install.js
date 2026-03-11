#!/usr/bin/env node

const { chmodSync, existsSync, mkdirSync, readFileSync, writeFileSync } = require("node:fs");
const { createHash } = require("node:crypto");
const path = require("node:path");

const supportedPlatforms = new Map([
  ["darwin", "darwin"],
  ["linux", "linux"],
]);

const supportedArchitectures = new Map([
  ["arm64", "arm64"],
  ["x64", "amd64"],
]);

function currentPackage(packageDir = path.join(__dirname, "..")) {
  return JSON.parse(readFileSync(path.join(packageDir, "package.json"), "utf8"));
}

function normalizeTarget(platform = process.platform, arch = process.arch) {
  const goos = supportedPlatforms.get(platform);
  const goarch = supportedArchitectures.get(arch);
  if (!goos || !goarch) {
    throw new Error(`unsupported platform ${platform}/${arch}; rstf currently supports macOS and Linux on x64 and arm64`);
  }
  return { goos, goarch, platform, arch };
}

function releaseBaseURL(pkg, env = process.env) {
  return env.RSTF_CLI_DOWNLOAD_BASE_URL || `https://github.com/${pkg.rstf.repo}/releases/download/${pkg.rstf.goVersion}`;
}

function checksumAssetName() {
  return "rstf-checksums.txt";
}

function checksumSignatureAssetName() {
  return "rstf-checksums.txt.sig";
}

function checksumCertificateAssetName() {
  return "rstf-checksums.txt.pem";
}

function binaryAssetName(goos, goarch) {
  return `rstf-${goos}-${goarch}`;
}

function vendorBinaryPath(platform = process.platform, arch = process.arch, scriptsDir = __dirname) {
  return path.join(scriptsDir, "vendor", platform, arch, "rstf");
}

function sha256Hex(data) {
  return createHash("sha256").update(data).digest("hex");
}

function parseChecksums(contents) {
  const checksums = new Map();
  for (const line of contents.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed) {
      continue;
    }
    const match = trimmed.match(/^([a-f0-9]{64})\s+\*?(.+)$/i);
    if (!match) {
      throw new Error(`invalid checksum line: ${trimmed}`);
    }
    checksums.set(match[2], match[1].toLowerCase());
  }
  return checksums;
}

async function fetchBuffer(fetchImpl, url) {
  const response = await fetchImpl(url);
  if (!response.ok) {
    throw new Error(`download failed from ${url}: ${response.status} ${response.statusText}`);
  }
  return Buffer.from(await response.arrayBuffer());
}

async function installBinary({
  pkg = currentPackage(),
  env = process.env,
  fetchImpl = fetch,
  platform = process.platform,
  arch = process.arch,
  scriptsDir = __dirname,
} = {}) {
  if (env.RSTF_CLI_SKIP_INSTALL === "1" || env.RSTF_CLI_LOCAL_BINARY || env.RSTF_CLI_LOCAL_SOURCE) {
    return null;
  }

  const target = normalizeTarget(platform, arch);
  const assetName = binaryAssetName(target.goos, target.goarch);
  const baseURL = releaseBaseURL(pkg, env);
  const binaryURL = `${baseURL}/${assetName}`;
  const checksumsURL = `${baseURL}/${checksumAssetName()}`;
  const binaryPath = vendorBinaryPath(platform, arch, scriptsDir);

  if (existsSync(binaryPath)) {
    return binaryPath;
  }

  const vendorDir = path.dirname(binaryPath);
  mkdirSync(vendorDir, { recursive: true });

  const [checksumsData, binaryData] = await Promise.all([
    fetchBuffer(fetchImpl, checksumsURL),
    fetchBuffer(fetchImpl, binaryURL),
  ]);

  const checksums = parseChecksums(checksumsData.toString("utf8"));
  const expectedDigest = checksums.get(assetName);
  if (!expectedDigest) {
    throw new Error(`checksum missing for ${assetName} in ${checksumsURL}`);
  }

  const actualDigest = sha256Hex(binaryData);
  if (actualDigest !== expectedDigest) {
    throw new Error(`checksum mismatch for ${assetName}: expected ${expectedDigest}, got ${actualDigest}`);
  }

  writeFileSync(binaryPath, binaryData);
  chmodSync(binaryPath, 0o755);
  return binaryPath;
}

async function main() {
  await installBinary();
}

if (require.main === module) {
  main().catch((error) => {
    console.error(`rstf: failed to install CLI binary: ${error.message}`);
    process.exit(1);
  });
}

module.exports = {
  binaryAssetName,
  checksumAssetName,
  checksumCertificateAssetName,
  checksumSignatureAssetName,
  currentPackage,
  installBinary,
  normalizeTarget,
  parseChecksums,
  releaseBaseURL,
  sha256Hex,
  vendorBinaryPath,
};
