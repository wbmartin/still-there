# On remote
ssh pathway-atticus-uat-vm
```
cp /etc/letsencrypt/live/local-atticus-snipper.teamtaggle.com/fullchain.pem ~brandon/local-atticus-snipper.teamtaggle.com-fullchain.pem

cp /etc/letsencrypt/live/local-atticus-snipper.teamtaggle.com/privkey.pem ~brandon/local-atticus-snipper.teamtaggle.com-privkey.pem
```

# on Local
```
scp pathway-atticus-uat-vm:~/local-atticus-snipper.teamtaggle.com-fullchain.pem ~/Code/ProtoStack-ReactGo/etc/crt/api-srv-fullchain.pem
scp pathway-atticus-uat-vm:~/local-atticus-snipper.teamtaggle.com-privkey.pem ~/Code/ProtoStack-ReactGo/etc/crt/api-srv-privkey.pem
```
