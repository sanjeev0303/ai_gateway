# syntax=docker/dockerfile:1

# Multi-stage build for Next.js app (ai-gateway-web).
# Context MUST be the repo root so root .dockerignore applies.
# compose sets: context: ../.. dockerfile: app/Dockerfile

# ── deps ─────────────────────────────────────────────────────────────────────
FROM node:22-alpine AS deps
WORKDIR /app

COPY app/package.json app/package-lock.json ./
RUN npm ci --ignore-scripts

# ── builder ───────────────────────────────────────────────────────────────────
FROM node:22-alpine AS builder
WORKDIR /app

# Accept build-time public env vars
ARG NEXT_PUBLIC_API_URL=http://localhost:8081
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL

COPY --from=deps /app/node_modules ./node_modules
COPY app/ .

RUN npm run build

# ── runner ────────────────────────────────────────────────────────────────────
FROM node:22-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN addgroup --system --gid 1001 nodejs \
 && adduser  --system --uid 1001 nextjs

# Public files
COPY --from=builder /app/public ./public

# Standalone output (next.config must set output: 'standalone')
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs
EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME=0.0.0.0

CMD ["node", "server.js"]
