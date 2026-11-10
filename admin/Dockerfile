# Multi-stage build for the platform console (ai-gateway-admin).
# Context MUST be the repo root so the workspace lockfile and packages resolve.
# compose sets: context: ../.. dockerfile: admin/Dockerfile

# ── deps ─────────────────────────────────────────────────────────────────────
FROM node:22-alpine AS deps
WORKDIR /app

# Every workspace's package.json is needed: npm ci checks the whole lockfile.
COPY package.json package-lock.json ./
COPY packages/contracts/package.json packages/contracts/
COPY packages/api-client/package.json packages/api-client/
COPY packages/console-ui/package.json packages/console-ui/
COPY packages/cli/package.json packages/cli/
COPY app/package.json app/
COPY admin/package.json admin/
RUN --mount=type=cache,target=/root/.npm npm ci --ignore-scripts --fetch-retries=5 --fetch-retry-mintimeout=20000 --fetch-retry-maxtimeout=120000

# ── builder ───────────────────────────────────────────────────────────────────
FROM node:22-alpine AS builder
WORKDIR /app

# Same heap ceiling as the dashboard build, for the same small-VM reason.
ENV NODE_OPTIONS=--max-old-space-size=1024

COPY package.json package-lock.json ./
COPY packages ./packages
COPY admin ./admin
# After the sources, which would otherwise overwrite the nested tree. npm keeps
# react and react-dom under admin/node_modules.
COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /app/admin/node_modules ./admin/node_modules

# Rewrites are resolved at build time, so the proxy target is a build argument.
ARG CONTROL_API_URL
ENV CONTROL_API_URL=${CONTROL_API_URL}

RUN --mount=type=cache,target=/app/admin/.next/cache npm run build -w admin

# ── runner ────────────────────────────────────────────────────────────────────
FROM node:22-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN addgroup --system --gid 1001 nodejs \
 && adduser  --system --uid 1001 nextjs

COPY --from=builder /app/admin/public ./public

# Tracing root is the repo root, so standalone mirrors it: server.js lives at
# standalone/admin/server.js. Flatten it up and merge the nested node_modules.
COPY --from=builder /app/admin/.next/standalone/node_modules ./node_modules
COPY --from=builder /app/admin/.next/standalone/admin ./
COPY --from=builder /app/admin/.next/static ./.next/static

RUN mkdir -p .next/cache && chown -R nextjs:nodejs .next/cache

USER nextjs
EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME=0.0.0.0

CMD ["node", "server.js"]
