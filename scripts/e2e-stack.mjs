#!/usr/bin/env node
// Runs the E2E stack without Docker, for environments without it (e.g. cloud
// agents): builds the backend and the frontend bundle, starts the backend on
// 127.0.0.1:8081 with a fresh database, and serves the bundle on :7300 with
// /api, /health and /carddav proxied, like docker/nginx.conf does.
//
// Usage: node scripts/e2e-stack.mjs [-- <command to run against the stack>]
// Without a command it keeps the stack running until Ctrl+C.
// Set E2E_SKIP_BUILD=1 to reuse the previous builds.
import { spawn, spawnSync } from 'node:child_process'
import { createReadStream, existsSync, mkdtempSync, rmSync, statSync } from 'node:fs'
import http from 'node:http'
import { tmpdir } from 'node:os'
import { extname, join, normalize, resolve } from 'node:path'

const root = resolve(import.meta.dirname, '..')
const backendDir = join(root, 'backend')
const frontendDir = join(root, 'frontend')
const buildDir = join(frontendDir, 'build')
const BACKEND_PORT = 8081
const APP_PORT = 7300
const PROXIED = ['/api/', '/health', '/carddav/', '/.well-known/carddav']
const TYPES = {
  '.html': 'text/html',
  '.js': 'text/javascript',
  '.css': 'text/css',
  '.json': 'application/json',
  '.svg': 'image/svg+xml',
  '.png': 'image/png',
  '.ico': 'image/x-icon',
  '.txt': 'text/plain',
  '.map': 'application/json',
  '.woff2': 'font/woff2',
}

function run(cmd, args, opts) {
  const r = spawnSync(cmd, args, { stdio: 'inherit', ...opts })
  if (r.status !== 0) {
    console.error(`e2e-stack: ${cmd} ${args.join(' ')} failed`)
    process.exit(1)
  }
}

if (process.env.E2E_SKIP_BUILD !== '1') {
  run('go', ['build', '-o', 'meerkat', '.'], { cwd: backendDir })
  run('corepack', ['yarn', 'build'], {
    cwd: frontendDir,
    env: { ...process.env, REACT_APP_API_URL: '' },
  })
}

const dataDir = mkdtempSync(join(tmpdir(), 'meerkat-e2e-'))
const backend = spawn(join(backendDir, 'meerkat'), [], {
  cwd: backendDir,
  stdio: ['ignore', 'inherit', 'inherit'],
  env: {
    ...process.env,
    PORT: String(BACKEND_PORT),
    SQLITE_DB_PATH: join(dataDir, 'test.db'),
    PROFILE_PHOTO_DIR: join(dataDir, 'photos'),
    JWT_SECRET_KEY: 'test-secret-key-for-e2e-testing-minimum-32-chars',
    FRONTEND_URL: `http://localhost:${APP_PORT}`,
    DISABLE_REGISTRATION: 'false',
    COOKIE_SECURE: 'false',
    LOG_LEVEL: process.env.LOG_LEVEL ?? 'warn',
  },
})

const server = http.createServer((req, res) => {
  if (PROXIED.some(p => req.url.startsWith(p))) {
    const upstream = http.request(
      { host: '127.0.0.1', port: BACKEND_PORT, path: req.url, method: req.method, headers: req.headers },
      up => {
        res.writeHead(up.statusCode, up.headers)
        up.pipe(res)
      }
    )
    upstream.on('error', () => {
      res.writeHead(502)
      res.end()
    })
    req.pipe(upstream)
    return
  }
  // Static file, else SPA fallback to index.html.
  const path = normalize(decodeURIComponent(req.url.split('?')[0])).replace(/^(\.\.[/\\])+/, '')
  let file = join(buildDir, path)
  if (!file.startsWith(buildDir) || !existsSync(file) || statSync(file).isDirectory()) {
    file = join(buildDir, 'index.html')
  }
  res.writeHead(200, { 'Content-Type': TYPES[extname(file)] ?? 'application/octet-stream' })
  createReadStream(file).pipe(res)
})

let exiting = false
function shutdown(code) {
  if (exiting) return
  exiting = true
  server.close()
  backend.kill('SIGTERM')
  rmSync(dataDir, { recursive: true, force: true })
  process.exit(code)
}
process.on('SIGINT', () => shutdown(130))
process.on('SIGTERM', () => shutdown(143))
backend.on('exit', code => {
  if (!exiting) {
    console.error(`e2e-stack: backend exited (${code})`)
    shutdown(1)
  }
})

async function waitHealthy() {
  for (let i = 0; i < 60; i++) {
    try {
      const r = await fetch(`http://127.0.0.1:${APP_PORT}/health`)
      if (r.ok) return
    } catch {}
    await new Promise(r => setTimeout(r, 500))
  }
  console.error('e2e-stack: backend not healthy after 30s')
  shutdown(1)
}

server.listen(APP_PORT, async () => {
  await waitHealthy()
  const sep = process.argv.indexOf('--')
  const cmd = sep === -1 ? [] : process.argv.slice(sep + 1)
  if (cmd.length === 0) {
    console.log(`e2e-stack: app on http://localhost:${APP_PORT} (Ctrl+C to stop)`)
    return
  }
  const child = spawn(cmd[0], cmd.slice(1), { cwd: frontendDir, stdio: 'inherit' })
  child.on('exit', code => shutdown(code ?? 1))
})
