# API server certificate renewal (local TLS testing)

`<api-host>` is the name the certificate was issued for; `<cert-host>` is the
ssh alias of the machine that holds the Let's Encrypt files.

```
# On remote
ssh <cert-host>
```
```
cp /etc/letsencrypt/live/<api-host>/fullchain.pem ~/<api-host>-fullchain.pem

cp /etc/letsencrypt/live/<api-host>/privkey.pem ~/<api-host>-privkey.pem
```

```
# On local, from the repo root
scp <cert-host>:~/<api-host>-fullchain.pem etc/crt/api-srv-fullchain.pem
scp <cert-host>:~/<api-host>-privkey.pem etc/crt/api-srv-privkey.pem
```
