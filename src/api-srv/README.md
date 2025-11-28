# Starter API Server

A production-ready Go API server starter template with comprehensive security features, multi-tenancy support, and JWT-based authentication.

## Features

- **Security First**: JWT authentication, CSRF protection, rate limiting, user blocking
- **Multi-Tenant**: Security context-based isolation
- **RBAC**: Role-based access control with granular permissions
- **CRUD Operations**: Full REST API for note entities with optimistic locking
- **Logging**: Structured logging with shrmpl-log integration
- **Caching**: shrmpl-kv integration for session and security data
- **Database**: PostgreSQL with pgx driver
- **Clean Architecture**: Separated concerns with handlers, repositories, and models

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL
- shrmpl-kv and shrmpl-log services

### Setup

1. **Download shrmpl services**:
   ```bash
   cd src/api-srv
   chmod +x bin/010-download-shrmpl.sh
   ./bin/010-download-shrmpl.sh
   ```

2. **Setup database**:
   ```bash
   createdb loc_starter_db
   psql -d loc_starter_db -f sql/010-initialcreate.sql
   psql -d loc_starter_db -f sql/015-data.sql
   ```

3. **Run the server**:
   ```bash
   go run cmd/server/main.go
   ```

2. **Start services**:
   ```bash
   # Start shrmpl-kv (port 7171)
    ./bin/tools/skv-srv

   # Start shrmpl-log (port 7379)
    ./bin/tools/slog-srv
   ```

3. **Setup database**:
   ```bash
   createdb loc_starter_db
   psql -d loc_starter_db -f sql/010-initialcreate.sql
   psql -d loc_starter_db -f sql/data.sql
   ```

4. **Run the server**:
   ```bash
   go run cmd/server/main.go
   ```

## API Endpoints

### Authentication
- `POST /auth` - Initial authentication (username/password)
- `POST /auth2` - MFA authentication (temp token + TOTP)
- `POST /logoff` - User logout

### Examples (CRUD)
- `GET /example?exampleId={id}` - Get example
- `POST /example` - Create example
- `PUT /example` - Update example
- `DELETE /example?exampleId={id}&updateTs={ts}` - Delete example

### Example Batch Operations
- `GET /exampleList?ids={comma-separated}` - Get multiple examples
- `POST /exampleList` - Create multiple examples (all-or-nothing transaction)
- `PUT /exampleList` - Update multiple examples (all-or-nothing transaction)
- `DELETE /exampleList?ids={comma-separated}` - Delete multiple examples (all-or-nothing transaction)

### Health Check
- `GET /health` - Service health check

## Security Features

### JWT Authentication
- RS256 signed tokens with configurable expiration
- Custom claims including user ID, session ID, security context, and grants map

### CSRF Protection
- X-CSRF-Token header validation against JWT session ID suffix
- Automatic skipping for safe HTTP methods

### Rate Limiting
- Per-user and unauthenticated request limits
- shrmpl-kv backed counters with automatic expiration

### User Blocking
- Administrative user blocking via shrmpl-kv flags
- Immediate effect across all sessions

### RBAC
- Grant-based permissions per security context
- Centralized grant definitions with short codes
- Context-aware authorization checks

## Configuration

Edit `etc/loc-dev-app-srv-config.env`:

```env
DATABASE_URL=postgres://user:pass@host:port/db
JWT_SECRET=your-secret-key
JWT_SESSION_EXPIRE_HOURS=72
MAX_USER_REQ_PER_MIN=200
MAX_UNAUTH_REQ_PER_MIN=20
LOG_RECEIVER_HOST_PORT=127.0.0.1:7379
KV_HOST_PORT=127.0.0.1:7171
SERVER_NAME=proto-go-app
SERVER_PORT=8080
TLS_CERT_PATH=/etc/ssl/certs/server.crt
TLS_KEY_PATH=/etc/ssl/private/server.key
```

## Database Schema

### Core Tables
- `app_user` - User accounts with MFA settings
- `user_group` - User groups with grant assignments
- `user_grant` - Available permissions
- `note` - Example entity with full audit trail

### Key Design Decisions
- Soft deletes with `delete_ts` and `delete_user_id`
- Optimistic locking using `update_ts`
- Security context on all tables for multi-tenancy
- Snake_case column naming
- Numeric(17,7) timestamps for microsecond precision

## Testing

```bash
# Health check
curl https://localhost:8080/health

# Authenticate (MFA exempt user)
curl -X POST https://localhost:8080/auth \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123!"}'

# Get example (requires auth cookie)
curl "https://localhost:8080/example?exampleId=1" \
  -H "Cookie: auth=YOUR_JWT_TOKEN"

# Get multiple examples
curl "https://localhost:8080/exampleList?ids=1,2,3" \
  -H "Cookie: auth=YOUR_JWT_TOKEN"

# Create multiple examples
curl -X POST https://localhost:8080/exampleList \
  -H "Content-Type: application/json" \
  -H "Cookie: auth=YOUR_JWT_TOKEN" \
  -d '[{"example_text":"Batch example 1"}, {"example_text":"Batch example 2"}]'

# Update multiple examples
curl -X PUT https://localhost:8080/exampleList \
  -H "Content-Type: application/json" \
  -H "Cookie: auth=YOUR_JWT_TOKEN" \
  -d '[{"example_id":1, "example_text":"Updated 1"}, {"example_id":2, "example_text":"Updated 2"}]'

# Delete multiple examples
curl -X DELETE "https://localhost:8080/exampleList?ids=1,2" \
  -H "Cookie: auth=YOUR_JWT_TOKEN"
```

## Architecture

```
cmd/server/
├── main.go                 # Server entry point and routing

internal/
├── app_lib/                # Shared application libraries
│   ├── config.go          # Configuration loading
│   ├── kv.go              # Key-value store interface (ThisAppKVInterface)
│   ├── logging.go         # Logging interface (ThisAppLoggerInterface)
│   ├── auth.go            # JWT and password utilities
│   ├── rbac.go            # Authorization logic
│   └── security.go        # Security helper with checks
│
├── handler/                        # HTTP handlers
│   ├── health_handler.go           # Health check
│   ├── auth_handler.go             # Authentication endpoints
│   ├── example_handler.go          # Example CRUD operations
│   └── common.go                   # Shared handler utilities
│
├── model/                  # Data models and constants
│   ├── app_user.go        # User model
│   ├── user_grant.go      # Grant definitions
│   ├── user_group.go      # Group model
│   ├── example.go         # Example model
│   └── request_response.go # API DTOs
│
└── repository/             # Data access layer
    ├── user_repository.go
    ├── user_group_repository.go
    ├── user_grant_repository.go
    └── example_repository.go
```

## Development

### Adding New Entities

1. Create model in `internal/model/`
2. Add grant constants to `user_grant.go`
3. Create repository in `internal/repository/`
4. Implement handlers in `internal/handler/`
5. Add routes in `cmd/server/main.go`

### Security Considerations

- All handlers use the SecurityHelper for consistent security checks
- JWT tokens are HttpOnly and Secure in production
- Passwords are bcrypt hashed
- All database operations include security context validation
- Optimistic locking prevents concurrent modification conflicts

## Clean Abstractions

The application uses clean interface abstractions for external dependencies:

- **`ThisAppLoggerInterface`**: Logging abstraction that hides shrmpl-log implementation
- **`ThisAppKVInterface`**: Key-value store abstraction that hides shrmpl-kv implementation

**Benefits:**
- **Implementation Hiding**: Main application code never sees shrmpl-specific details
- **Testability**: Interfaces can be easily mocked for unit testing
- **Flexibility**: Can swap implementations by editing only `logging.go` and `kv.go`
- **Clean Architecture**: Dependencies are injected as interfaces, not concrete types

## License

This is a starter template - customize as needed for your project.