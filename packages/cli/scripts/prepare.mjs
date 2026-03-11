import { execFileSync } from "node:child_process";
import { mkdirSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

function currentBinaryName() {
  const platformMap = new Map([
    ["darwin", "darwin"],
    ["linux", "linux"],
    ["win32", "windows"],
  ]);
  const archMap = new Map([
    ["x64", "amd64"],
    ["arm64", "arm64"],
  ]);

  const goos = platformMap.get(process.platform);
  const goarch = archMap.get(process.arch);
  if (!goos || !goarch) {
    throw new Error(`Unsupported platform: ${process.platform}/${process.arch}`);
  }

  return {
    goos,
    goarch,
    fileName: `rstf-${goos}-${goarch}${process.platform === "win32" ? ".exe" : ""}`,
  };
}

const packageDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const repoRoot = path.resolve(packageDir, "..", "..");
const distDir = path.join(packageDir, "dist");
const binary = currentBinaryName();
const outputPath = path.join(distDir, binary.fileName);

mkdirSync(distDir, { recursive: true });

execFileSync("go", ["build", "-o", outputPath, "./cmd/rstf"], {
  cwd: repoRoot,
  stdio: "inherit",
});

writeFileSync(
  path.join(distDir, "metadata.json"),
  JSON.stringify(
    {
      binary: binary.fileName,
      goarch: binary.goarch,
      goos: binary.goos,
      repoRoot,
    },
    null,
    2,
  ) + "\n",
);
