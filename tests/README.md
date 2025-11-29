# Integration Tests

This directory contains integration tests that validate the full HTTPS stack of the API server, including authentication, CSRF protection, and CRUD operations. Tests run against the configured TLS endpoint.

## Prerequisites

- Python 3.8+
- The API server must be running with TLS enabled
- All required services (KV store, logging, database) must be running
- `TEST_BASE_URL` must be configured in `etc/api-srv-loc.env`

## Configuration

Add the test base URL to your configuration file:

```bash
# etc/api-srv-loc.env
TEST_BASE_URL=https://local-atticus-snipper.teamtaggle.com:8443
```

This should match your TLS certificate's hostname and the port your application is running on.

## Setup

1. **Configure the test URL:**
   Add `TEST_BASE_URL=https://your-tls-hostname:port` to `etc/api-srv-loc.env`

2. **Start the application:**
   ```bash
   ./bin/215-run-api-srv
   ```

3. **Run the integration tests:**
   ```bash
   ./bin/320-test-web-service
   ```

   The script will automatically:
   - Create a Python virtual environment (if it doesn't exist)
   - Install test dependencies in the virtual environment
   - Run the complete test suite

## Virtual Environment

The tests use a dedicated virtual environment at `integration-tests/.venv/` to:
- Isolate test dependencies from your system Python
- Ensure reproducible test execution
- Avoid conflicts with other Python projects

The virtual environment is automatically created and managed by the test script.

**To recreate the virtual environment:**
```bash
cd integration-tests
rm -rf .venv
# Run the test script again to recreate it
```

## Running Tests

### Option 1: Using the test script (recommended)
```bash
./bin/320-test-web-service
```
The script automatically reads `TEST_BASE_URL` from the config file and runs tests against your TLS endpoint.

### Option 2: Manual execution
```bash
cd integration-tests
export TEST_BASE_URL=https://your-tls-hostname:port
python -m pytest -v --html=integration_test_report.html
```

## Test Coverage

### Authentication & Security
- JWT token authentication
- CSRF token validation
- Invalid credential handling
- Malformed request handling

### Example CRUD Operations
- Create single example
- Retrieve single example
- Update single example
- Delete single example
- Bulk create examples
- Bulk retrieve examples
- Bulk update examples
- Bulk delete examples

### Error Scenarios
- Non-existent resources (404)
- Invalid request data (400)
- Missing authentication (403)
- Missing CSRF tokens (403)

## Test Architecture

### TestClient Class
- Handles JWT and CSRF token management
- Tracks created examples for automatic cleanup
- Provides authenticated request methods

### Fixtures
- `test_client`: Session-scoped authenticated client
- `example_tracker`: Tracks examples per test

### Automatic Cleanup
- All test examples are automatically cleaned up after test execution
- Uses proper optimistic locking for safe deletion

## Test Data Management

Tests create temporary examples with identifiable names and automatically clean them up. The cleanup process:

1. Tracks all created example IDs and update timestamps
2. Performs bulk deletion at the end of the test session
3. Uses optimistic locking to ensure safe deletion

## Reports

Test execution generates:
- **Console output**: Real-time test results
- **HTML report**: `integration_test_report.html` with detailed results
- **Coverage data**: Integration test validation of HTTP endpoints

## Troubleshooting

### Application Not Running
```
❌ ERROR: Application is not running on https://local-atticus-snipper.teamtaggle.com:8443
```
**Solution:**
1. Start the application first:
   ```bash
   ./bin/215-run-api-srv
   ```
2. Ensure `TEST_BASE_URL` in `etc/api-srv-loc.env` matches your TLS hostname and port
3. Verify the TLS certificate is valid for the hostname

### Missing Dependencies
```
ModuleNotFoundError: No module named 'pytest'
```
**Solution:** The test script automatically installs dependencies in the virtual environment. If you encounter this error, try:
```bash
cd integration-tests
rm -rf .venv  # Remove old virtual environment
# Run the test script again to recreate it
```

### Virtual Environment Issues
If you encounter virtual environment problems:
```bash
cd integration-tests
rm -rf .venv
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

### Test Failures
- Check that all services are running
- Verify database connectivity
- Check application logs for errors
- Ensure test data cleanup from previous runs

## Best Practices

- Tests are isolated and don't depend on each other
- Automatic cleanup prevents test data pollution
- Comprehensive error scenario testing
- Real HTTP requests validate the complete stack
- Session-scoped authentication for efficiency
