import { readFileSync, rmSync } from "node:fs";

import { STATE_FILE } from "./global-setup";

export default function globalTeardown() {
  try {
    const state = JSON.parse(readFileSync(STATE_FILE, "utf-8")) as {
      pids?: number[];
      dataDir?: string;
    };
    for (const pid of state.pids ?? []) {
      try {
        process.kill(pid);
      } catch {
        // already gone
      }
    }
    if (state.dataDir !== undefined) rmSync(state.dataDir, { recursive: true, force: true });
  } catch {
    // no state file: setup never completed
  }
  rmSync(STATE_FILE, { force: true });
}
