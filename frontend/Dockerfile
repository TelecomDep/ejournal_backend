FROM node:22.23.2-alpine3.24 AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY index.html vite.config.mjs ./
COPY public ./public
COPY src ./src
RUN npm run build

FROM nginx:1.30.4-alpine3.24

COPY nginx.conf.template /etc/nginx/nginx.conf.template
COPY 40-envsubst-domain.sh /docker-entrypoint.d/40-envsubst-domain.sh
RUN chmod +x /docker-entrypoint.d/40-envsubst-domain.sh
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80 443
