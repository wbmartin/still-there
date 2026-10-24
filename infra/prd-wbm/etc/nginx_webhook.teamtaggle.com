# /srv/prd-stillthere/etc/nginx_webhook.teamtaggle.com, symlinked from
# /etc/nginx/sites-enabled/ (infra/README.md). The public door for Telnyx's
# inbound-text webhook, and nothing else: POST /telnyx-inbound goes to the
# API's POST /smsInbound on loopback; every other path and method is
# refused here. A name of its own so it can move to another host or app
# later without touching the web app's site.
#
# Keep the port in step with SERVER_PORT in prd-stillthere-api.env (and
# nginx_stillthere.teamtaggle.com's upstream). TLS is certbot's lineage in
# /etc/letsencrypt/live/webhook.teamtaggle.com/, issued and renewed by
# webroot (doc/320, Certificates).
#
# The API authenticates every call by its Telnyx Ed25519 signature
# (TELNYX_PUBLIC_KEY), so nginx adds no auth; it only narrows the surface.


# Plain redirect, and the HTTP-01 webroot certbot issues and renews this
# name's certificate through.
server {
    listen 80;
    listen [::]:80;
    server_name webhook.teamtaggle.com;
    access_log /var/log/nginx/prd-stillthere-webhook_access.log;
    error_log  /var/log/nginx/prd-stillthere-webhook_error.log;

    location ^~ /.well-known/acme-challenge/ {
        root /var/www/letsencrypt;
        default_type text/plain;
        try_files $uri =404;
    }

    location / {
        return 301 https://webhook.teamtaggle.com$request_uri;
    }
}


server {
    listen       443 ssl;
    listen       [::]:443 ssl;
    http2        on;
    ssl_certificate /etc/letsencrypt/live/webhook.teamtaggle.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/webhook.teamtaggle.com/privkey.pem;
    server_name  webhook.teamtaggle.com;
    # /var/log/nginx, so Debian's logrotate for nginx rotates them. The
    # fail2ban jail prd-stillthere reads the access log.
    access_log   /var/log/nginx/prd-stillthere-webhook_access.log;
    error_log    /var/log/nginx/prd-stillthere-webhook_error.log;

    # A message.received event is about 2 KiB; the API caps the body at
    # 64 KiB itself (smsWebhookMaxBodyBytes). Same cap here, so a larger
    # body never reaches it.
    client_max_body_size 64k;

    # Telnyx's webhook: POST only. The exact match strips nothing; the
    # proxy_pass URI names the API's route.
    location = /telnyx-inbound {
        if ($request_method != POST) {
            return 405;
        }
        proxy_pass http://127.0.0.1:8090/smsInbound;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Host $http_host;
        # Appends $remote_addr; ClientIP counts TRUSTED_PROXY_COUNT hops from
        # the right, so never overwrite it. The telnyx-signature-ed25519 and
        # telnyx-timestamp headers pass through as sent.
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_redirect off;
        proxy_connect_timeout 30s;
        proxy_send_timeout    300s;
        proxy_read_timeout    300s;
    }

    # Everything else: no web root, no API.
    location / {
        add_header X-Content-Type-Options "nosniff" always;
        add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
        return 404;
    }
}
