import { readFileSync, rmSync } from "node:fs";

import { STATE_FILE } from "./global-setup";

export default function globalTeardown() {
  try {
    const state = JSON.parse(readFileSync(STATE_FILE, "utf-8")) as { pid?: number };
    if (state.pid) process.kill(state.pid);
  } catch {
    // already gone
  }
  rmSync(STATE_FILE, { force: true });
}
