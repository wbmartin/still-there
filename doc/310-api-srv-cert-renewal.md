# API server certificate renewal (local TLS testing)

`<api-host>` is the name the certificate was issued for; `<uat-vm>` is the
ssh alias of the machine that holds the Let's Encrypt files.

```
# On remote
ssh <uat-vm>
```
```
cp /etc/letsencrypt/live/<api-host>/fullchain.pem ~/<api-host>-fullchain.pem

cp /etc/letsencrypt/live/<api-host>/privkey.pem ~/<api-host>-privkey.pem
```

```
# On local, from the repo root
scp <uat-vm>:~/<api-host>-fullchain.pem etc/crt/api-srv-fullchain.pem
scp <uat-vm>:~/<api-host>-privkey.pem etc/crt/api-srv-privkey.pem
```
