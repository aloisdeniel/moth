import { spawn } from "node:child_process";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));

const BASE_URL = "http://127.0.0.1:8990";
export const STATE_FILE = join(tmpdir(), "moth-e2e-state.json");

// Launches the built moth binary on a fresh data directory and captures
// the one-time setup token it prints, so the test can walk the true
// first-run flow.
export default async function globalSetup() {
  const bin = process.env.MOTH_BIN ?? resolve(here, "../../../bin/moth");
  const dataDir = mkdtempSync(join(tmpdir(), "moth-e2e-"));

  const child = spawn(bin, ["serve"], {
    env: {
      ...process.env,
      MOTH_ADDR: "127.0.0.1:8990",
      MOTH_BASE_URL: BASE_URL,
      MOTH_DATA_DIR: dataDir,
    },
    stdio: ["ignore", "pipe", "pipe"],
    detached: false,
  });

  let output = "";
  child.stderr.on("data", (chunk: Buffer) => (output += chunk.toString()));
  child.stdout.on("data", (chunk: Buffer) => (output += chunk.toString()));

  // Wait for the server to answer and the setup token to appear.
  const deadline = Date.now() + 15_000;
  let token = "";
  for (;;) {
    const m = /\/admin\?setup=([a-z0-9]+)/.exec(output);
    if (m) {
      token = m[1];
      try {
        const resp = await fetch(`${BASE_URL}/healthz`);
        if (resp.ok) break;
      } catch {
        // not up yet
      }
    }
    if (Date.now() > deadline) {
      child.kill();
      throw new Error(`moth did not start; output so far:\n${output}`);
    }
    await new Promise((r) => setTimeout(r, 100));
  }

  writeFileSync(STATE_FILE, JSON.stringify({ pid: child.pid, setupToken: token }));
  // Keep the process handle alive independently of this Node process.
  child.unref();
}
