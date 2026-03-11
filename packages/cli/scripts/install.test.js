const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");

const {
  binaryAssetName,
  checksumAssetName,
  installBinary,
  parseChecksums,
  sha256Hex,
  vendorBinaryPath,
} = require("./install");

test("parseChecksums reads sha256sum output", () => {
  const checksums = parseChecksums([
    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  rstf-darwin-arm64",
    "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb *rstf-linux-amd64",
    "",
  ].join("\n"));

  assert.equal(checksums.get("rstf-darwin-arm64"), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa");
  assert.equal(checksums.get("rstf-linux-amd64"), "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb");
});

test("installBinary downloads and verifies the checksum before writing", async () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "rstf-cli-install-"));
  const packageDir = path.join(tempDir, "pkg");
  const scriptsDir = path.join(packageDir, "scripts");
  fs.mkdirSync(scriptsDir, { recursive: true });

  const binaryData = Buffer.from("rstf-test-binary");
  const assetName = binaryAssetName("linux", "amd64");
  const checksums = `${sha256Hex(binaryData)}  ${assetName}\n`;
  const requested = [];

  const fetchImpl = async (url) => {
    requested.push(url);
    if (url.endsWith(`/${checksumAssetName()}`)) {
      return {
        ok: true,
        status: 200,
        statusText: "OK",
        arrayBuffer: async () => Buffer.from(checksums),
      };
    }
    if (url.endsWith(`/${assetName}`)) {
      return {
        ok: true,
        status: 200,
        statusText: "OK",
        arrayBuffer: async () => binaryData,
      };
    }
    throw new Error(`unexpected url ${url}`);
  };

  const installedPath = await installBinary({
    pkg: {
      rstf: {
        repo: "rafbgarcia/rstf",
        goVersion: "v0.1.0-alpha.1",
      },
    },
    env: {},
    fetchImpl,
    platform: "linux",
    arch: "x64",
    scriptsDir,
  });

  assert.deepEqual(requested, [
    "https://github.com/rafbgarcia/rstf/releases/download/v0.1.0-alpha.1/rstf-checksums.txt",
    "https://github.com/rafbgarcia/rstf/releases/download/v0.1.0-alpha.1/rstf-linux-amd64",
  ]);
  assert.equal(installedPath, vendorBinaryPath("linux", "x64", scriptsDir));
  assert.equal(fs.readFileSync(installedPath, "utf8"), "rstf-test-binary");
});

test("installBinary rejects a checksum mismatch", async () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "rstf-cli-install-"));
  const packageDir = path.join(tempDir, "pkg");
  const scriptsDir = path.join(packageDir, "scripts");
  fs.mkdirSync(scriptsDir, { recursive: true });

  const assetName = binaryAssetName("darwin", "arm64");

  await assert.rejects(
    installBinary({
      pkg: {
        rstf: {
          repo: "rafbgarcia/rstf",
          goVersion: "v0.1.0-alpha.1",
        },
      },
      env: {},
      fetchImpl: async (url) => ({
        ok: true,
        status: 200,
        statusText: "OK",
        arrayBuffer: async () =>
          Buffer.from(url.endsWith(`/${checksumAssetName()}`) ? `aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  ${assetName}\n` : "actual-binary"),
      }),
      platform: "darwin",
      arch: "arm64",
      scriptsDir,
    }),
    /checksum mismatch/,
  );
});
