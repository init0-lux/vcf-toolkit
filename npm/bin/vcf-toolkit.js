#!/usr/bin/env node
const fs = require('fs');
const path = require('path');
const os = require('os');
const https = require('https');
const { spawnSync } = require('child_process');
const tar = require('tar');
const AdmZip = require('adm-zip');

const pkg = require('../package.json');

const PLATFORM_MAP = { darwin: 'darwin', linux: 'linux', win32: 'windows' };
const ARCH_MAP = { x64: 'amd64', arm64: 'arm64' };

const platform = PLATFORM_MAP[process.platform];
const arch = ARCH_MAP[process.arch];
if (!platform || !arch) {
  console.error(`Unsupported platform/arch: ${process.platform}/${process.arch}`);
  process.exit(1);
}

const version = process.env.VCF_TOOLKIT_VERSION || pkg.version;
const tag = `v${version}`;
const isWindows = platform === 'windows';
const ext = isWindows ? 'zip' : 'tar.gz';
const filename = `vcf-toolkit_${version}_${platform}_${arch}.${ext}`;

const baseURL =
  process.env.VCF_TOOLKIT_DIST_URL ||
  'https://github.com/init0-lux/vcf-toolkit/releases/download';

const url = `${baseURL}/${tag}/${filename}`;

const binDir = path.join(__dirname, '..', 'bin', version, `${platform}-${arch}`);
const binName = isWindows ? 'vcf-toolkit.exe' : 'vcf-toolkit';
const binPath = path.join(binDir, binName);

function ensureDir(dir) {
  fs.mkdirSync(dir, { recursive: true });
}

function download(urlToGet, dest, redirects = 0) {
  return new Promise((resolve, reject) => {
    https
      .get(urlToGet, (res) => {
        if (
          res.statusCode &&
          res.statusCode >= 300 &&
          res.statusCode < 400 &&
          res.headers.location
        ) {
          if (redirects > 5) {
            reject(new Error('Too many redirects while downloading release asset'));
            return;
          }
          resolve(download(res.headers.location, dest, redirects + 1));
          return;
        }

        if (res.statusCode !== 200) {
          reject(
            new Error(
              `Download failed (${res.statusCode}) from ${urlToGet}`
            )
          );
          return;
        }

        const file = fs.createWriteStream(dest);
        res.pipe(file);
        file.on('finish', () => file.close(resolve));
        file.on('error', reject);
      })
      .on('error', reject);
  });
}

async function extract(archivePath, destDir) {
  if (isWindows) {
    const zip = new AdmZip(archivePath);
    zip.extractAllTo(destDir, true);
    return;
  }

  await tar.x({ file: archivePath, cwd: destDir });
}

async function ensureBinary() {
  if (fs.existsSync(binPath)) {
    return;
  }

  ensureDir(binDir);

  // Download and extract outside binDir, then rename the binary into place so
  // an interrupted first run never leaves a truncated binary behind.
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'vcf-toolkit-'));
  try {
    const archivePath = path.join(tmpDir, filename);
    await download(url, archivePath);
    await extract(archivePath, tmpDir);

    const staged = `${binPath}.${process.pid}.tmp`;
    fs.copyFileSync(path.join(tmpDir, binName), staged);
    if (!isWindows) {
      fs.chmodSync(staged, 0o755);
    }
    fs.renameSync(staged, binPath);
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  }
}

async function run() {
  await ensureBinary();

  const result = spawnSync(binPath, process.argv.slice(2), {
    stdio: 'inherit',
  });

  if (result.error) {
    console.error(result.error);
    process.exit(1);
  }

  process.exit(result.status ?? 1);
}

run().catch((err) => {
  console.error(`vcf-toolkit: ${err.message}`);
  process.exit(1);
});
