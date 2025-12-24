# Create the database

- TODO add database setup instructions.

- copy and run the sql/010-initialcreate.sql in a psql session
- copy and run the sql/015-initialdata.sql in a psql session


# Nginx setup
wil mayneed to have a container running nginx.  This will depend on how we solve the file upload/download problem


# Setting up certificates for local shrmpl vault (non MTLS).

1. register a domain with your name registrar (godaddy, namecheap, etc)
2. create a subdomain for your config server, e.g. config.yourdomain.com and A record it to your public ip address
  ```
   e.g. loc-proto // 155.138.163.197 
  ```
3. shell to a cloud server, sudo up, request a certificate with certbot.
  ```
  certbot -d loc-proto.cnfg.li
  ```
4. cat the cert and key files and copy them to your local machine
  ```
  cat /etc/letsencrypt/live/loc-proto.cnfg.li/fullchain.pem
  cat /etc/letsencrypt/live/loc-proto.cnfg.li/privkey.pem
  ```
and paste them in
  ```
  ~/Code/ProtoStack-ReactGo/etc/crt/shrmpl_vault_server_fullchain.pem
  ~/Code/ProtoStack-ReactGo/etc/crt/shrmpl_vault_server_privkey.pem
  ```

5. Update the ~/Code/ProtoStack-ReactGo/etc/loc-api-srv.env
  ```
VAULT_SERVER=https://loc-proto.cnfg.li:7474
  ```

6. Update your local /etc/hosts files
  ```
127.0.0.1       loc-proto.cnfg.li
  ```

7. Restart the shrmpl-vault server
8. restart the api-srv
