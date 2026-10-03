# /etc/nginx/sites-available/uat-proto.example.com  (symlink into sites-enabled)
# Replace uat-proto.example.com with the real hostname; keep the port in step
# with SERVER_PORT in uat-proto-api.env and HEALTH_URL in uat-proto-deploy.env.
# Run bin/uat-proto-renew-web-certs once before linking this file, or
# nginx -t fails on the missing pem files.

upstream proto_api_server {
  server 127.0.0.1:8088 fail_timeout=0;
  keepalive 64;
}


server {
    if ($host = uat-proto.example.com) {
        return 301 https://uat-proto.example.com$request_uri;
    } # managed by Certbot

    listen 80;
    listen [::]:80;
    server_name uat-proto.example.com;
    return 404; # managed by Certbot
}


server { # simple reverse-proxy
    listen       443 ssl;
    ssl_certificate /srv/uat-proto/crt/uat-proto.example.com/fullchain.pem;
    ssl_certificate_key /srv/uat-proto/crt/uat-proto.example.com/privkey.pem;
    server_name  uat-proto.example.com;
    access_log   /srv/uat-proto/log/nginx_access.log  ;
    error_log   /srv/uat-proto/log/nginx_error.log  ;
    root    /srv/uat-proto/web-pub ;
    index index.html;
    client_max_body_size 20M;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "DENY" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    # The API. The trailing slash strips the /api prefix, because the Go server
    # registers /health, not /api/health.
    location /api/ {
        proxy_pass http://proto_api_server/;
        proxy_http_version 1.1;
        proxy_set_header Host $http_host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Origin $http_origin;
        proxy_redirect off;
        proxy_connect_timeout 30s;
        proxy_send_timeout    300s;
        proxy_read_timeout    300s;
    }

    # Hashed filenames, so the content at a given URL never changes. immutable
    # stops the browser revalidating on every navigation.
    #
    # The four security headers are repeated in every location block on purpose:
    # a single add_header anywhere in a location silently discards ALL
    # server-level add_headers for that location.
    location /assets/ {
        expires 1y;
        add_header Cache-Control "public, immutable" always;
        add_header X-Content-Type-Options "nosniff" always;
        add_header X-Frame-Options "DENY" always;
        add_header Referrer-Policy "strict-origin-when-cross-origin" always;
        add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
        try_files $uri =404;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }

    # index.html itself must never be cached, or a deploy leaves browsers
    # asking for hashed assets that no longer exist.
    location = /index.html {
        expires -1;
        add_header Cache-Control "no-store, no-cache, must-revalidate" always;
        add_header X-Content-Type-Options "nosniff" always;
        add_header X-Frame-Options "DENY" always;
        add_header Referrer-Policy "strict-origin-when-cross-origin" always;
        add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    }
}
