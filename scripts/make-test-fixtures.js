#!/usr/bin/env node
/*
  Create test fixture projects under dist/fixtures and run `pnpm i` in each
  (fallback to `npm i` if pnpm is unavailable) to generate sizeable node_modules
  trees for scanning/deletion tests.

  Usage:
    node scripts/make-test-fixtures.js            # create defaults and install
    node scripts/make-test-fixtures.js --count 3  # create N projects
    node scripts/make-test-fixtures.js --no-install  # only write package.jsons
    node scripts/make-test-fixtures.js --path ./custom-fixtures
*/
const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');

function ensureDir(p) {
  fs.mkdirSync(p, { recursive: true });
}

function writeJSON(p, obj) {
  fs.writeFileSync(p, JSON.stringify(obj, null, 2) + '\n', 'utf8');
}

function execCmd(cmd, args, options) {
  const res = spawnSync(cmd, args, { stdio: 'inherit', ...options });
  if (res.error) throw res.error;
  if (res.status !== 0) {
    throw new Error(`${cmd} ${args.join(' ')} exited with code ${res.status}`);
  }
}

function hasCmd(cmd) {
  try {
    const res = spawnSync(cmd, ['--version'], { stdio: 'ignore' });
    return res && res.status === 0;
  } catch (_) {
    return false;
  }
}

function parseArgs(argv) {
  const out = { count: 4, install: true, basePath: path.resolve(__dirname, '..', 'dist', 'fixtures') };
  for (let i = 2; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--count' || a === '-n') {
      out.count = parseInt(argv[++i], 10) || out.count;
    } else if (a === '--no-install') {
      out.install = false;
    } else if (a === '--path' || a === '-p') {
      out.basePath = path.resolve(argv[++i]);
    } else if (a === '--no-text') {
      // optional: skip creating the special text-app-* project
      out.noText = true;
    } else {
      console.warn(`Unknown arg: ${a}`);
    }
  }
  return out;
}

function pickRandom(objArray, min, max) {
  const n = Math.floor(Math.random() * (max - min + 1)) + min;
  const shuffled = [...objArray].sort(() => Math.random() - 0.5);
  return shuffled.slice(0, n);
}

function makePackageJSON(name) {
  // Base deps common to all fixtures
  const baseDeps = {
    react: '^18.2.0',
    'react-dom': '^18.2.0',
    typescript: '^5.4.0',
    vite: '^5.0.0',
    tailwindcss: '^3.4.0',
    '@types/react': '^18.2.0',
    '@types/react-dom': '^18.2.0'
  };

  const extraDepsPool = [
    ['lodash', '^4.17.21'],
    ['axios', '^1.6.8'],
    ['date-fns', '^3.6.0'],
    ['three', '^0.160.0'],
    ['rxjs', '^7.8.1'],
    ['chart.js', '^4.4.1'],
    ['d3', '^7.9.0'],
    ['zustand', '^4.5.4'],
    ['immer', '^10.1.1'],
    ['swr', '^2.2.5'],
    ['@mui/material', '^5.15.10'],
    ['@chakra-ui/react', '^2.8.2'],
    ['next', '^14.2.3'],
    ['@tanstack/react-query', '^5.51.0']
  ];

  const extraDevPool = [
    ['eslint', '^8.57.0'],
    ['prettier', '^3.3.3'],
    ['@typescript-eslint/eslint-plugin', '^7.0.0'],
    ['@typescript-eslint/parser', '^7.0.0'],
    ['eslint-config-prettier', '^9.1.0'],
    ['eslint-plugin-react', '^7.34.2']
  ];

  const deps = { ...baseDeps };
  // Pick 3-7 random extra deps to vary size
  for (const [name, ver] of pickRandom(extraDepsPool, 3, 7)) deps[name] = ver;
  const devDeps = {};
  for (const [name, ver] of pickRandom(extraDevPool, 1, 3)) devDeps[name] = ver;

  return {
    name,
    version: '0.0.0',
    private: true,
    scripts: {
      build: 'echo build',
      test: 'echo test'
    },
    dependencies: deps,
    devDependencies: devDeps
  };
}

(function main() {
  const opts = parseArgs(process.argv);
  ensureDir(opts.basePath);

  console.log(`Creating ${opts.count} fixture projects in ${opts.basePath}`);

  const created = [];
  for (let i = 1; i <= opts.count; i++) {
    const dir = path.join(opts.basePath, `react-app-${i}`);
    ensureDir(dir);
    const pkg = makePackageJSON(path.basename(dir));
    writeJSON(path.join(dir, 'package.json'), pkg);
    // Minimal files to resemble a project
    const srcDir = path.join(dir, 'src');
    ensureDir(srcDir);
    fs.writeFileSync(path.join(srcDir, 'index.tsx'), 'console.log("hello")\n');
    created.push(dir);
  }

  // Always create a special project whose folder name contains the word "text"
  // to help manual testing of the TUI search/filter feature.
  if (!opts.noText) {
    const rand = Math.random().toString(36).slice(2, 8);
    const dir = path.join(opts.basePath, `text-app-${rand}`);
    ensureDir(dir);
    // Use a stable project name; only the folder has a random suffix
    const pkg = makePackageJSON('text-app');
    writeJSON(path.join(dir, 'package.json'), pkg);
    const srcDir = path.join(dir, 'src');
    ensureDir(srcDir);
    fs.writeFileSync(
      path.join(srcDir, 'index.tsx'),
      `// Text app fixture for search testing\nconsole.log("text app ${rand}")\n`
    );
    created.push(dir);
    console.log(`Added special search test project: ${path.basename(dir)}`);
  }

  if (!opts.install) {
    console.log('Skipping npm install (--no-install provided).');
    return;
  }

  // Choose package manager (prefer pnpm, fallback to npm)
  const usePnpm = hasCmd('pnpm');
  const pm = usePnpm ? 'pnpm' : 'npm';
  console.log(`\nPackage manager: ${pm}${usePnpm ? '' : ' (pnpm not found, falling back)'}\n`);

  // Install sequentially to avoid overwhelming the network/disk
  for (const dir of created) {
    console.log(`\n==> Installing dependencies in ${dir}`);
    try {
      if (usePnpm) {
        execCmd('pnpm', ['install', '--ignore-scripts', '--reporter=silent'], {
          cwd: dir,
          env: { ...process.env }
        });
      } else {
        execCmd('npm', ['install', '--no-audit', '--no-fund', '--progress=false'], {
          cwd: dir,
          env: { ...process.env, npm_config_loglevel: 'warn' }
        });
      }
    } catch (e) {
      console.error(`${pm} install failed in ${dir}:`, e.message);
      console.error('You can rerun later:');
      if (usePnpm) {
        console.error(`  (cd ${dir} && pnpm i --ignore-scripts --reporter=silent)`);
      } else {
        console.error(`  (cd ${dir} && npm i --no-audit --no-fund --progress=false)`);
      }
    }
  }

  console.log('\nAll fixtures prepared.');
})();
