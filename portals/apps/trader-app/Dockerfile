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

ARG VITE_API_BASE_URL=http://localhost:8080/api
ARG VITE_IDP_BASE_URL=https://localhost:8090
ARG VITE_IDP_CLIENT_ID=TRADER_PORTAL_APP
ARG VITE_APP_URL=http://localhost:5173
ARG VITE_IDP_SCOPES=openid,profile,email
ARG VITE_IDP_PLATFORM=AsgardeoV2
ARG VITE_SHOW_AUTOFILL_BUTTON=true

ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_IDP_BASE_URL=$VITE_IDP_BASE_URL
ENV VITE_IDP_CLIENT_ID=$VITE_IDP_CLIENT_ID
ENV VITE_APP_URL=$VITE_APP_URL
ENV VITE_IDP_SCOPES=$VITE_IDP_SCOPES
ENV VITE_IDP_PLATFORM=$VITE_IDP_PLATFORM
ENV VITE_SHOW_AUTOFILL_BUTTON=$VITE_SHOW_AUTOFILL_BUTTON

RUN pnpm --filter trader-app build

FROM nginx:1.27-alpine

COPY apps/trader-app/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /workspace/apps/trader-app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
