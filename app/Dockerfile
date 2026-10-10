
# Multi-stage build for Next.js app (ai-gateway-web).
# Context MUST be the repo root so root .dockerignore applies.
# compose sets: context: ../.. dockerfile: app/Dockerfile

# ── deps ─────────────────────────────────────────────────────────────────────
FROM node:22-alpine AS deps
WORKDIR /app

# Install from the root workspace so @ai-governance/* resolve to the local
# packages rather than the (unpublished) registry.
COPY package.json package-lock.json ./
COPY packages/contracts/package.json packages/contracts/
COPY packages/api-client/package.json packages/api-client/
COPY packages/cli/package.json packages/cli/
COPY app/package.json app/
RUN npm ci --ignore-scripts

# ── builder ───────────────────────────────────────────────────────────────────
FROM node:22-alpine AS builder
WORKDIR /app

# Bounds the heap so `next build` cannot grow into a kernel OOM-kill (exit 137)
# on a small builder VM. 1024MB is verified sufficient; do not raise or remove.
ENV NODE_OPTIONS=--max-old-space-size=1024

COPY package.json package-lock.json ./
COPY packages ./packages
COPY app ./app
# Copied after the sources, which would otherwise overwrite the nested tree.
# npm left next under app/node_modules rather than hoisting it.
COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /app/app/node_modules ./app/node_modules

ARG CONTROL_API_URL
ENV CONTROL_API_URL=${CONTROL_API_URL}

RUN --mount=type=cache,target=/app/app/.next/cache npm run build -w app

# ── runner ────────────────────────────────────────────────────────────────────
FROM node:22-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN addgroup --system --gid 1001 nodejs \
 && adduser  --system --uid 1001 nextjs

# Public files
COPY --from=builder /app/app/public ./public

# Tracing root is the repo root, so standalone mirrors it: server.js lives at
# standalone/app/server.js. Flatten that subtree up so the CMD stays unchanged,
# and merge the nested node_modules (npm left next under app/).
COPY --from=builder /app/app/.next/standalone/node_modules ./node_modules
COPY --from=builder /app/app/.next/standalone/app ./
COPY --from=builder /app/app/.next/static ./.next/static

USER nextjs
EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME=0.0.0.0

CMD ["node", "server.js"]
