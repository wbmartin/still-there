# /etc/nginx/sites-available/uat-stillthere.example.com  (symlink into sites-enabled)
# Replace uat-stillthere.example.com with the real hostname; keep the port in step
# with SERVER_PORT in uat-stillthere-api.env and HEALTH_URL in uat-stillthere-deploy.env.
# Run bin/uat-stillthere-renew-web-certs once before linking this file, or
# nginx -t fails on the missing pem files.

upstream stillthere_api_server {
  server 127.0.0.1:8088 fail_timeout=0;
  keepalive 64;
}


# Plain redirect. certbot --nginx certonly answers its HTTP-01 challenge here
# by inserting a server-level rewrite ... break ahead of this return.
server {
    listen 80;
    listen [::]:80;
    server_name uat-stillthere.example.com;
    return 301 https://uat-stillthere.example.com$request_uri;
}


server { # simple reverse-proxy
    listen       443 ssl;
    listen       [::]:443 ssl;
    ssl_certificate /srv/uat-stillthere/crt/uat-stillthere.example.com/fullchain.pem;
    ssl_certificate_key /srv/uat-stillthere/crt/uat-stillthere.example.com/privkey.pem;
    server_name  uat-stillthere.example.com;
    access_log   /srv/uat-stillthere/log/nginx_access.log  ;
    error_log   /srv/uat-stillthere/log/nginx_error.log  ;
    root    /srv/uat-stillthere/web-pub ;
    index index.html;
    client_max_body_size 20M;

    # Security headers are set per web location, never at server level: a
    # location that sets any add_header drops every server-level one, and one
    # that sets none (like /api/) inherits them all on top of its own.
    #
    # The API. The trailing slash strips the /api prefix, because the Go server
    # registers /health, not /api/health. No add_header here: the API's own
    # headers (security.HeadersMiddleware) stand alone.
    location /api/ {
        proxy_pass http://stillthere_api_server/;
        # HTTP/1.1 with Connection cleared, or nginx sends Connection: close
        # and the upstream keepalive pool is never used.
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Host $http_host;
        # Appends $remote_addr; ClientIP counts TRUSTED_PROXY_COUNT hops from
        # the right, so never overwrite it.
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_redirect off;
        proxy_connect_timeout 30s;
        proxy_send_timeout    300s;
        proxy_read_timeout    300s;
    }

    # Hashed filenames, so the content at a given URL never changes. immutable
    # stops the browser revalidating on every navigation.
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
        add_header X-Content-Type-Options "nosniff" always;
        add_header X-Frame-Options "DENY" always;
        add_header Referrer-Policy "strict-origin-when-cross-origin" always;
        add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
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
