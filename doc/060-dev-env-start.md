
# Database
Start Docker
```bash
cd path/to/infrahub/loc-postgres  # your local infrahub checkout, outside this repo
source ./brandon_env_setup-mac1.sh
./515-run-postgres.sh
docker exec -it loc-postgres /bin/bash
#inside the container
su - postgres
# inside postgres shell
psql
# inside postgres
\c loc_starter_db
```

First time, or to rebuild the schema:
```bash
bin/510-reset-test-db
```

# new shell for the app server
```bash
./bin/100-build-api-server
# confirm it builds
./bin/215-run-api-srv
#visual confirm it's running
```

#In a new shell 
~/Code/ProtoStack-ReactGo/bin/220-run-web-app

# To stop
./bin/216-stop-local-stack
