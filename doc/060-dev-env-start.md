cd Code/ProtoStack-ReactGo

./bin/010-download-shrmpl.sh

# Database
Start Docker
```bash
cd /Users/brandon/proj/infrahub/loc-postgres
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

# new shell for shrimpl server
```bash

./bin/210-run-shrmpl-app-infra
```

# new shell for the app server
```bash
./bin/100-build-api-server
# confirm it builds
./bin/215-run-api-srv
#visual confirm it's running
```
