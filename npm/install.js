#!/usr/bin/env node
"use strict";

const https = require("https");
const http = require("http");
const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");

const pkg = require("./package.json");
const version = pkg.version;

const PLATFORM_MAP = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const ARCH_MAP = {
  x64: "amd64",
  arm64: "arm64",
};

function getPlatform() {
  const platform = PLATFORM_MAP[process.platform];
  if (!platform) {
    throw new Error(`Unsupported platform: ${process.platform}`);
  }
  return platform;
}

function getArch() {
  const arch = ARCH_MAP[process.arch];
  if (!arch) {
    throw new Error(`Unsupported architecture: ${process.arch}`);
  }
  return arch;
}

function getExtension() {
  return process.platform === "win32" ? ".exe" : "";
}

function download(url) {
  return new Promise((resolve, reject) => {
    const client = url.startsWith("https") ? https : http;
    client
      .get(url, { headers: { "User-Agent": "ccbox-npm-install" } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          return download(res.headers.location).then(resolve, reject);
        }
        if (res.statusCode !== 200) {
          return reject(new Error(`Download failed: HTTP ${res.statusCode} for ${url}`));
        }
        const chunks = [];
        res.on("data", (chunk) => chunks.push(chunk));
        res.on("end", () => resolve(Buffer.concat(chunks)));
        res.on("error", reject);
      })
      .on("error", reject);
  });
}

async function install() {
  const platform = getPlatform();
  const arch = getArch();
  const ext = getExtension();

  const binName = `ccbox-${platform}-${arch}${ext}`;
  const url = `https://github.com/ccdevkit/ccbox/releases/download/v${version}/${binName}`;

  const binDir = path.join(__dirname, "bin");
  const binPath = path.join(binDir, `ccbox${ext}`);

  console.log(`Downloading ccbox v${version} for ${platform}/${arch}...`);

  const data = await download(url);

  if (!fs.existsSync(binDir)) {
    fs.mkdirSync(binDir, { recursive: true });
  }

  fs.writeFileSync(binPath, data);

  if (process.platform !== "win32") {
    fs.chmodSync(binPath, 0o755);
  }

  console.log(`Installed ccbox to ${binPath}`);
}

install().catch((err) => {
  console.error(`Failed to install ccbox: ${err.message}`);
  process.exit(1);
});
