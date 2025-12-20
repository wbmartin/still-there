# Project Commands

## API Server Management

To run the api server:
```
./bin/215-run-api-srv
```
- **Interactive mode** (default): Shows live logs, blocks until server exits
- **Daemon mode** (background): Runs in background, exits after startup message
- **Detection**: Automatically detects based on terminal output redirection
- **Force interactive**: `./bin/215-run-api-srv --interactive` or `./bin/215-run-api-srv -i`
- **Force daemon**: `./bin/215-run-api-srv --daemon` or `./bin/215-run-api-srv -d`

To check if api server is running:
```
curl -k https://local-atticus-snipper.teamtaggle.com:8443/health
```

To view server logs (when running in background):
```
tail -f tmp/server.log
```

To stop the api server:
```
pkill -f "proto-go-app" || kill $(cat tmp/server.pid 2>/dev/null) 2>/dev/null || true
```

## Infrastructure Management

To run the shrmpl infrastructure services (kv, log, vault):
```
./bin/210-run-shrmpl-app-infra
```

To stop all shrmpl app infrastructure (services, api server, web app):
```
./bin/211-stop-shrmpl-app-infra
```

To run the web app:
```
./bin/220-run-web-app
```

## Testing Commands

To test the api server (Unit tests):
```
./bin/310-test-repo-model
```

To test the api server (integration tests):
```
./bin/320-test-web-service
```

To load test the api server:
```
./bin/330-test-load
```
