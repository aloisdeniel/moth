import type { DescMethodUnary } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { createConnectQueryKey } from "@connectrpc/connect-query";
import { QueryClient } from "@tanstack/react-query";

import { createDemoTransport } from "./demo";

// The SPA always talks to the moth instance that serves it; in `make dev`
// the Vite proxy forwards RPCs to the Go server. Demo builds (`--mode demo`,
// hosted on the website) swap the network for an in-browser fake backend;
// in production builds VITE_DEMO is unset and the demo code is dropped.
export const transport = import.meta.env.VITE_DEMO
  ? createDemoTransport()
  : createConnectTransport({ baseUrl: "/" });

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error) => {
        // Auth failures redirect to the login screen; retrying only delays
        // that.
        if (error instanceof ConnectError && error.code === Code.Unauthenticated) {
          return false;
        }
        return failureCount < 2;
      },
      staleTime: 5_000,
      refetchOnWindowFocus: false,
    },
  },
});

// invalidate marks every cached result of the given RPCs stale (their key
// is a partial match over any input).
export function invalidate(...schemas: DescMethodUnary[]) {
  for (const schema of schemas) {
    void queryClient.invalidateQueries({
      queryKey: createConnectQueryKey({ schema, cardinality: "finite" }),
    });
  }
}

export function isUnauthenticated(error: unknown): boolean {
  return error instanceof ConnectError && error.code === Code.Unauthenticated;
}

// errorMessage renders a ConnectError without the "[code]" prefix noise.
export function errorMessage(error: unknown): string {
  if (error instanceof ConnectError) {
    return error.rawMessage;
  }
  return error instanceof Error ? error.message : String(error);
}
