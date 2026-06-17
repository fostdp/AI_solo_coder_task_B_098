FROM node:20-alpine AS builder

WORKDIR /build
COPY . .

FROM nginx:1.25-alpine

COPY --from=builder /build/css /usr/share/nginx/html/css
COPY --from=builder /build/js /usr/share/nginx/html/js
COPY --from=builder /build/index.html /usr/share/nginx/html/index.html

COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
    CMD wget -qO- http://localhost/ || exit 1
