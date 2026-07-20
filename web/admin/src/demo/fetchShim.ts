// A few screens read plain-HTTP surfaces of the moth server directly
// (SDK versions from the embedded pub/npm registries, the setup status)
// instead of going through the connect transport. On the static demo host
// those endpoints don't exist, so demo mode answers them from a fetch shim.

const DEMO_SDK_VERSION = "1.0.0";

function demoResponse(pathname: string): Response | null {
  if (pathname.endsWith("/pub/api/packages/moth_auth")) {
    return Response.json({ latest: { version: DEMO_SDK_VERSION } });
  }
  if (pathname.endsWith("/npm/@moth/react")) {
    return Response.json({ "dist-tags": { latest: DEMO_SDK_VERSION } });
  }
  if (pathname.endsWith("/admin/status")) {
    return Response.json({ needsSetup: false });
  }
  return null;
}

export function installFetchShim(): void {
  const original = window.fetch.bind(window);
  window.fetch = (input, init) => {
    const url = typeof input === "string" || input instanceof URL ? String(input) : input.url;
    try {
      const faked = demoResponse(new URL(url, window.location.href).pathname);
      if (faked) {
        return Promise.resolve(faked);
      }
    } catch {
      // Unparseable URL: let the real fetch deal with it.
    }
    return original(input, init);
  };
}
