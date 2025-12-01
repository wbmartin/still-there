# API Server Technical Specification

## Overview

This document specifies the ProtoStack-ReactGo API server, a production-ready Golang application server providing secure, multi-tenant CRUD operations with JWT-based authentication, RBAC authorization, and comprehensive security features.

### Architecture

- **Language**: Go 1.25.4
- **Framework**: Standard library with pgx for PostgreSQL
- **Architecture**: Clean Architecture with layered design
  - **Presentation Layer**: HTTP handlers with JSON APIs
  - **Application Layer**: Business logic and security checks
  - **Domain Layer**: Models and business rules
  - **Infrastructure Layer**: Database repositories and external services
- **External Dependencies**:
  - PostgreSQL database
  - shrmpl-kv (Redis-like key-value store)
  - shrmpl-log (TCP logging service)

### Project Structure

```
src/api-srv/
├── cmd/server/main.go              # Application entry point
├── internal/
│   ├── app_lib/                    # Shared application libraries
│   │   ├── auth.go                 # JWT handling, password hashing
│   │   ├── config.go               # Configuration management
│   │   ├── database.go             # PostgreSQL connection
│   │   ├── kv.go                   # shrmpl-kv client
│   │   ├── logging.go              # Structured logging
│   │   ├── rbac.go                 # RBAC logic
│   │   ├── security.go             # Security helpers (CSRF, rate limiting)
│   │   └── services.go             # Application services
│   ├── handler/                    # HTTP handlers
│   │   ├── auth_handler.go         # Authentication endpoints
│   │   ├── example_handler.go      # CRUD operations
│   │   ├── health_handler.go       # Health checks
│   │   └── common.go               # Shared handler utilities
│   ├── model/                      # Domain models
│   │   ├── app_user_model.go       # User entities
│   │   ├── example_model.go        # Example entities
│   │   ├── security_model.go       # Security-related models
│   │   └── user_grant_model.go     # RBAC models
│   └── repository/                 # Data access layer
│       ├── errors.go               # Repository errors
│       ├── example_repository.go   # Example data operations
│       ├── retry.go                # Database retry logic
│       └── user_repository.go      # User data operations
├── go.mod
├── go.sum
└── README.md
```

## Database Design

### Schema Guidelines

- **Primary Keys**: `entity_name_id` pattern (e.g., `example_id`, `app_user_id`)
- **Table Names**: Singular entity names
- **Timestamps**: `numeric(17,7)` for high-precision timestamps
- **Audit Fields**: All tables include:
  - `create_ts`, `create_user_id`
  - `update_ts`, `update_user_id`
  - `delete_ts`, `delete_user_id` (soft deletes)
- **Multi-tenancy**: `security_context` integer field on all tables
- **Naming**: Snake_case column names
- **Relationships**: No foreign keys; managed in application layer
- **Defaults/Triggers**: Prohibited; all logic in application code

### Core Tables

#### app_user
```sql
CREATE TABLE app_user (
    app_user_id serial PRIMARY KEY,
    user_name text NOT NULL,
    pw_hash text NOT NULL,
    first_name text,
    last_name text,
    email text,
    security_context integer NOT NULL,
    user_group_membership_list integer[],
    mfa_exempt boolean NOT NULL DEFAULT false,
    create_ts numeric(17,7) NOT NULL,
    create_user_id integer NOT NULL,
    update_ts numeric(17,7) NOT NULL,
    update_user_id integer NOT NULL,
    delete_ts numeric(17,7),
    delete_user_id integer
);
```

#### user_group
```sql
CREATE TABLE user_group (
    user_group_id integer PRIMARY KEY,
    user_grant_list integer[],
    security_context integer NOT NULL,
    create_ts numeric(17,7) NOT NULL,
    create_user_id integer NOT NULL,
    update_ts numeric(17,7) NOT NULL,
    update_user_id integer NOT NULL,
    delete_ts numeric(17,7),
    delete_user_id integer
);
```

#### user_grant
```sql
CREATE TABLE user_grant (
    user_grant_id integer PRIMARY KEY,
    grant_short_code text NOT NULL,
    grant_description text NOT NULL,
    security_context integer NOT NULL DEFAULT 0,
    create_ts numeric(17,7) NOT NULL,
    create_user_id integer NOT NULL,
    update_ts numeric(17,7) NOT NULL,
    update_user_id integer NOT NULL,
    delete_ts numeric(17,7),
    delete_user_id integer
);
```

#### example
```sql
CREATE TABLE example (
    example_id serial PRIMARY KEY,
    example_text text,
    example_int integer,
    example_float float,
    example_int_array integer[],
    example_float_array float[],
    security_context integer NOT NULL,
    create_ts numeric(17,7) NOT NULL,
    create_user_id integer NOT NULL,
    update_ts numeric(17,7) NOT NULL,
    update_user_id integer NOT NULL,
    delete_ts numeric(17,7),
    delete_user_id integer
);
```

### Data Seeding

Initial data uses CTE for common values:
```sql
WITH common_vars AS (
    SELECT NOW() as ts, 0 as system_id
)
INSERT INTO user_grant (user_grant_id, grant_short_code, grant_description, security_context, create_ts, create_user_id, update_ts, update_user_id)
SELECT data.user_grant_id, data.grant_short_code, data.grant_description, data.security_context, common_vars.ts, common_vars.system_id, common_vars.ts, common_vars.system_id
FROM common_vars,
(VALUES
    (1001, 'GET_EXAMPLE', 'Retrieve examples', 0),
    (1002, 'ADD_EXAMPLE', 'Add new examples', 0),
    (1003, 'CHG_EXAMPLE', 'Change existing examples', 0),
    (1004, 'DEL_EXAMPLE', 'Delete examples', 0),
    (8001, 'CHG_OWN_PW', 'Change own password', 0),
    (9001, 'ADD_USER', 'Add new users', 0),
    (9002, 'BLOCK_USER', 'Block user access', 0),
    (9003, 'CHG_OTHER_USER_PW', 'Reset another user password', 0)
) AS data(user_grant_id, grant_short_code, grant_description, security_context);
```

## API Design

### Endpoints

| Method | Path | Auth | CSRF | RBAC Grant | Description |
|--------|------|------|------|------------|-------------|
| GET | /health | No | No | N/A | Basic health check |
| POST | /auth | No | No | N/A | Username/password authentication |
| POST | /auth2 | No | No | N/A | MFA token verification |
| POST | /logoff | Yes | No | N/A | Clear authentication cookie |
| GET | /example | Yes | No | GET_EXAMPLE | Retrieve single example |
| GET | /exampleList | Yes | No | GET_EXAMPLE | Retrieve multiple examples |
| POST | /example | Yes | Yes | ADD_EXAMPLE | Create new example |
| POST | /exampleList | Yes | Yes | ADD_EXAMPLE | Bulk create examples |
| PUT | /example | Yes | Yes | CHG_EXAMPLE | Update example |
| PUT | /exampleList | Yes | Yes | CHG_EXAMPLE | Bulk update examples |
| DELETE | /example | Yes | Yes | DEL_EXAMPLE | Soft delete example |
| DELETE | /exampleList | Yes | Yes | DEL_EXAMPLE | Bulk soft delete examples |

### Request/Response Format

- **Content-Type**: `application/json`
- **Field Naming**: camelCase in JSON, snake_case in database
- **Timestamps**: Unix nanoseconds as float64 (17,7 precision)
- **Error Responses**: Standard HTTP status codes with JSON error messages

### Authentication Flow

1. **Initial Auth** (`/auth`):
   - Input: `{userName: string, password: string}`
   - Verify credentials against `app_user` table
   - If `mfa_exempt = true`: Return full JWT
   - If `mfa_exempt = false`: Store temp token in shrmpl-kv, return temp token

2. **MFA Auth** (`/auth2`):
   - Input: `{tempToken: string, totpCode: string}`
   - Verify temp token from shrmpl-kv
   - Verify TOTP (placeholder implementation)
   - Return full JWT with cookie

3. **JWT Structure**:
   ```json
   {
     "iss": "proto-go-app",
     "sub": "username",
     "exp": 1678886400,
     "nbf": 1678882800,
     "iat": 1678882800,
     "jti": "unique-session-id",
     "uid": 123,
     "ses": "session-id",
     "sctx": 1,
     "grants": {
       "1": [1001, 1002, 1003],
       "2": [1001]
     }
   }
   ```

## Security

### Authentication
- **JWT**: RS256 signed tokens with session management
- **Password Hashing**: bcrypt with cost factor 12
- **Session Management**: Session IDs stored in JWT, validated against shrmpl-kv for blocking

### Authorization (RBAC)
- **Grants**: Per-security-context permissions
- **User Groups**: Collections of grants assigned to users
- **Context Switching**: Users can operate in multiple security contexts

### Security Checks (Per Request)
1. **JWT Validation**: Verify token signature and claims
2. **Rate Limiting**:
   - Authenticated: 200 requests/minute per user
   - Unauthenticated: 20 requests/minute per IP
3. **User Blocking**: Check shrmpl-kv for blocked users
4. **CSRF Protection**: Compare `X-CSRF-Token` header with last 3 chars of session ID
5. **RBAC**: Verify user has required grant for target security context

### Multi-tenancy
- **Security Context**: Integer identifier for tenant isolation
- **Data Segregation**: All queries filter by `security_context`
- **Grant Isolation**: Permissions scoped to security contexts

## Configuration

### Environment Variables (.env format)
```env
# Database
ENV_CFG_SQL_DB_NAME=proto_db
ENV_CFG_SQL_DB_HOST=localhost
ENV_CFG_SQL_DB_USER=app_user
ENV_CFG_SQL_DB_PASSWORD=secure_password
ENV_CFG_SQL_DB_PORT=5432

# Security
ENV_SEC_APP_JWT_SECRET_KEY=high_entropy_secret
ENV_CFG_JWT_SESSION_EXPIRE_HOURS=72

# Rate Limiting
ENV_CFG_MAX_USER_REQ_PER_MIN=200
ENV_CFG_MAX_UNAUTH_REQ_PER_MIN=20

# External Services
ENV_CFG_SHRMPL_KV_HOST=localhost:7379
ENV_CFG_SHRMPL_LOG_DEST=localhost:7380

# Application
ENV_CFG_SERVER_NAME=proto-go-app
```

### Loading
- Command-line argument specifies config file path
- Simple environment variable parsing (no external libraries)
- Validation of required fields

## Logging

### Structured Logging
- **Library**: slog (Go standard library)
- **Format**: JSON with consistent field names
- **Levels**: DEBUG, INFO, WARN, ERROR
- **Context**: Username, user ID, request details
- **Transport**: TCP connection to shrmpl-log service

### Log Codes
- 4-character codes for consistent categorization
- Examples: STRT (start), AUTH (authentication), DBFL (database failure)

### Username Tracking
- Extracted from JWT claims for authenticated requests
- Defaults to "PreAuth" for unauthenticated endpoints
- Included in all error responses and logs

## Error Handling

### HTTP Status Codes
- **200**: Success
- **201**: Created
- **400**: Bad Request (validation errors)
- **401**: Unauthorized (invalid/missing auth)
- **403**: Forbidden (insufficient permissions)
- **404**: Not Found
- **409**: Conflict (optimistic locking)
- **410**: Gone (soft deleted)
- **429**: Too Many Requests (rate limited)
- **500**: Internal Server Error

### Optimistic Locking
- Client sends `updateTs` in requests
- Server compares with database value in WHERE clause
- On conflict: Query to distinguish between modified (409) and deleted (410)

### Database Resilience
- **Retry Logic**: Exponential backoff for transient connection errors
- **Error Classification**: Retryable vs permanent errors
- **Timeout Handling**: Context-based cancellation

## Code Conventions

### Naming
- **Variables**: camelCase, descriptive names (e.g., `userName`, `securityContext`)
- **Lists**: Suffix with "List" (e.g., `userList`, `grantIdList`)
- **Functions**: PascalCase, action-oriented (e.g., `GetUserByUserName`)
- **Files**: snake_case, descriptive (e.g., `auth_handler.go`)

### Structure
- **Imports**: Grouped (standard library, third-party, internal)
- **Error Handling**: Early returns, wrapped errors
- **DRY Principle**: Shared utilities in `common.go`, `app_lib/`
- **Single Responsibility**: Focused functions and files

### Database Queries
- **Explicit Fields**: No wildcards in SELECT statements
- **Prepared Statements**: Parameterized queries
- **Multi-line**: Each field on separate line for readability

## Testing

### Unit Tests
- **Coverage**: Repository layer with pgx mocking
- **Framework**: testify/suite
- **Focus**: Business logic, error conditions

### Integration Tests
- **Python-based**: pytest with requests
- **Coverage**: Full API workflows
- **Environment**: Requires running server and dependencies

### Test Data
- **Isolation**: Separate test database/schema
- **Cleanup**: Proper teardown between tests
- **Fixtures**: Reusable test data setup

## Deployment

### Build Process
- **Commands**: `go build ./...`, `go test ./...`
- **Dependencies**: Download shrmpl binaries via `bin/010-download-shrmpl.sh`
- **Configuration**: Environment-specific config files

### Runtime Requirements
- **PostgreSQL**: Database server
- **shrmpl-kv**: Key-value store service
- **shrmpl-log**: Logging service
- **Network**: TCP connectivity between services

### Operational Considerations
- **Health Checks**: `/health` endpoint for load balancers
- **Metrics**: Basic request counting (future enhancement)
- **Scaling**: Stateless design supports horizontal scaling
- **Security**: No sensitive data in logs, secure cookie settings

## Development Workflow

### Local Development
1. Start shrmpl services: `bin/010-download-shrmpl.sh`
2. Initialize database: `psql -f sql/010-initialcreate.sql`
3. Seed data: `psql -f sql/015-initialdata.sql`
4. Run server: `go run cmd/server/main.go /path/to/config.env`
5. Run tests: `go test ./...`

### Code Quality
- **Formatting**: `go fmt`
- **Linting**: Manual review (future: golangci-lint)
- **Testing**: Required before commits
- **Documentation**: Inline comments, README updates

## Future Enhancements

### High Priority
- TOTP implementation for MFA
- Input validation middleware
- OpenAPI/Swagger documentation
- Comprehensive integration tests

### Medium Priority
- Performance monitoring
- Caching layer
- Configuration hot-reloading
- Advanced rate limiting

### Low Priority
- Docker containerization
- CI/CD pipeline
- Log aggregation
- Feature flags

This specification captures the current implementation and serves as a foundation for future development and maintenance of the ProtoStack-ReactGo API server.</content>
<parameter name="filePath">doc/105-api-srv-tech-spec.md