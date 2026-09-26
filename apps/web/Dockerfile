# syntax=docker/dockerfile:1

# Multi-stage build for the Next.js dashboard (apps/web) inside an npm
# workspaces monorepo. Context MUST be the repo root so root package.json /
# package-lock.json and the shared packages/ are visible.

FROM node:22-alpine AS deps
WORKDIR /app

# Minimal manifests so `npm ci` can resolve the workspace graph from the
# committed lockfile without copying all sources yet.
COPY package.json package-lock.json ./
COPY apps/web/package.json apps/web/package.json
COPY packages/api-client/package.json packages/api-client/package.json
COPY packages/contracts/package.json packages/contracts/package.json
COPY packages/cli/package.json packages/cli/package.json
RUN npm ci --no-audit --no-fund

FROM node:22-alpine AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /app/package-lock.json ./package-lock.json
COPY package.json ./
COPY apps/web ./apps/web
COPY packages ./packages
RUN npm run build --workspace=@ai-governance/web

FROM node:22-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV HOSTNAME=0.0.0.0
ENV PORT=3000

COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /app/package-lock.json ./package-lock.json
COPY --from=build /app/package.json ./package.json
COPY --from=build /app/apps/web ./apps/web
COPY --from=build /app/packages ./packages

# Non-root; matches the backend containers.
USER node
EXPOSE 3000

CMD ["npm", "run", "start", "--workspace=@ai-governance/web"]