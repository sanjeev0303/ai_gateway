import path from "node:path";
import { fileURLToPath } from "node:url";
import type { NextConfig } from "next";

const adminDir = path.dirname(fileURLToPath(import.meta.url));

// Server-side only, read at runtime. Locally the control API runs on :8081;
// in compose the container reaches it by service name on :8080.
const controlApiUrl = process.env.CONTROL_API_URL ?? "http://localhost:8081";

const nextConfig: NextConfig = {
  reactCompiler: true,
  output: "standalone",
  // The api-client workspace package lives outside admin/.
  outputFileTracingRoot: path.join(adminDir, ".."),
  // Shared design system, shipped as TypeScript source.
  transpilePackages: ["@ai-governance/console-ui"],
  async rewrites() {
    // Same-origin proxy to the control API, so sign-in sets the session cookie
    // on this app's own origin.
    return [{ source: "/api/v1/:path*", destination: `${controlApiUrl}/v1/:path*` }];
  },
};

export default nextConfig;
