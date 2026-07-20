import type { Transport } from "@connectrpc/connect";

import { installFetchShim } from "./fetchShim";
import { createDemoTransport as createTransport } from "./transport";
import { registerAnalytics } from "./services/analytics";
import { registerContent } from "./services/content";
import { registerInstance } from "./services/instance";
import { registerMonetization } from "./services/monetization";
import { registerProjects } from "./services/projects";
import { registerUsers } from "./services/users";

export { DemoBadge } from "./DemoBadge";

// createDemoTransport wires every service's demo handlers and returns the
// in-browser transport. This is the demo bundle's only entry point: when
// `import.meta.env.VITE_DEMO` is unset the call site is dead code and the
// whole demo/ tree is dropped from the production build.
export function createDemoTransport(): Transport {
  installFetchShim();
  registerInstance();
  registerProjects();
  registerUsers();
  registerContent();
  registerMonetization();
  registerAnalytics();
  return createTransport();
}
