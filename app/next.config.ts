import path from "node:path";
import { fileURLToPath } from "node:url";
import type { NextConfig } from "next";

const appDir = path.dirname(fileURLToPath(import.meta.url));

// Server-side only. A NEXT_PUBLIC_* var would be inlined at build time, which
// is wrong for a runtime proxy target — and would leak the API address to the
// browser. Local dev runs on the host (:8081); in compose the app container
// reaches the service by name on its internal port (:8080).
const controlApiUrl = process.env.CONTROL_API_URL ?? "http://localhost:8081";

const nextConfig: NextConfig = {
  reactCompiler: true,
  output: "standalone",
  // Workspace packages live outside app/, so trace from the repo root or they
  // are dropped from the standalone bundle.
  outputFileTracingRoot: path.join(appDir, ".."),
  async rewrites() {
    return [
      // Same-origin proxy to the control API. This is what lets the browser
      // send the session cookie and receive Set-Cookie from sign-in without
      // any CORS configuration.
      { source: "/api/v1/:path*", destination: `${controlApiUrl}/v1/:path*` },
      { source: "/api/control/v1/:path*", destination: `${controlApiUrl}/v1/:path*` },
    ];
  },
};

export default nextConfig;
