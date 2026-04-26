# syntax=docker/dockerfile:1.7

# 1) Build the static bundle with pnpm.
FROM node:20-alpine AS builder
RUN corepack enable
WORKDIR /app

# Use pnpm via corepack, pinned by the `packageManager` field in package.json.
COPY package.json pnpm-lock.yaml* .npmrc* ./
RUN pnpm install --frozen-lockfile

COPY . .
ARG VITE_API_BASE_URL=http://localhost:8080/api/v1
ARG VITE_WS_BASE_URL=ws://localhost:8080/api/v1
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_WS_BASE_URL=$VITE_WS_BASE_URL
RUN pnpm build

# 2) Serve the bundle with nginx.
FROM nginx:1.27-alpine AS runtime
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
