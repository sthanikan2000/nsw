FROM node:22-bookworm-slim AS builder

WORKDIR /workspace

RUN corepack enable

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml tsconfig.json ./
COPY apps/trader-app/package.json apps/trader-app/package.json
COPY apps/oga-app/package.json apps/oga-app/package.json
COPY packages/jsonforms-renderers/package.json packages/jsonforms-renderers/package.json
COPY packages/ui/package.json packages/ui/package.json

RUN pnpm install --frozen-lockfile

COPY . .

ARG VITE_PORT=5174
ARG VITE_INSTANCE_CONFIG=npqs
ARG VITE_OGA_API_BASE_URL=http://localhost:8081

ENV VITE_PORT=$VITE_PORT
ENV VITE_INSTANCE_CONFIG=$VITE_INSTANCE_CONFIG
ENV VITE_OGA_API_BASE_URL=$VITE_OGA_API_BASE_URL

RUN pnpm --filter oga-app build

FROM nginx:1.27-alpine

COPY apps/oga-app/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /workspace/apps/oga-app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
