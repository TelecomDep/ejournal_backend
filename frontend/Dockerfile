FROM node:20-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY public ./public
COPY src ./src
RUN npm run build

FROM nginx:1.27

COPY nginx.conf.template /etc/nginx/nginx.conf.template
COPY 40-envsubst-domain.sh /docker-entrypoint.d/40-envsubst-domain.sh
RUN chmod +x /docker-entrypoint.d/40-envsubst-domain.sh
COPY --from=builder /app/build /usr/share/nginx/html

EXPOSE 80 443
