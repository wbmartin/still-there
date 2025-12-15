# API Server Technical Specification

## Overview

This document specifies the ProtoStack-ReactGo API server, a production-ready Golang application server providing secure, multi-tenant CRUD operations with JWT-based authentication, RBAC authorization, and comprehensive security features.

### Architecture

- **Language**: Go 1.25.4
- **Framework**: Standard library with pgx for PostgreSQL
- **Architecture**: n-tier
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
│   │   ├── app_helper.go           # Error handling and response utilities
│   │   ├── auth.go                 # JWT handling, password hashing
│   │   ├── config.go               # Configuration management
│   │   ├── kv.go                   # shrmpl-kv client
│   │   ├── logging.go              # Structured logging
│   │   ├── rbac.go                 # RBAC logic
│   │   └── services.go             # Application services
│   ├── handler/                    # HTTP handlers
│   │   ├── auth_handler.go         # Authentication endpoints
│   │   ├── common.go               # Shared handler utilities
│   │   ├── example_handler.go      # Example CRUD operations
│   │   ├── health_handler.go       # Health checks
│   │   ├── user_group_handler.go   # User group CRUD operations
│   │   └── user_handler.go         # User CRUD operations
│   ├── model/                      # Domain models
│   │   ├── app_user_model.go       # User entities
│   │   ├── example_model.go        # Example entities
│   │   ├── example_model_test.go   # Example model tests
│   │   ├── grants.go               # Grant definitions
│   │   ├── security_model.go       # Security-related models
│   │   ├── user_grant_model.go     # RBAC models
│   │   ├── user_grant_model_test.go # User grant model tests
│   │   └── user_group_model.go     # User group entities
│   └── repository/                 # Data access layer
│       ├── errors.go               # Repository errors
│       ├── example_repository.go   # Example data operations
│       ├── example_repository_test.go # Example repository tests
│       ├── retry.go                # Database retry logic
│       ├── user_group_repository.go # User group data operations
│       ├── user_group_repository_test.go # User group repository tests
│       ├── user_repository.go      # User data operations
│       └── user_repository_test.go # User repository tests
├── .golangci.yml                   # Go linting configuration
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
  - `update_ts`, `update_user_id` (optimistic locking)
  - `delete_ts`, `delete_user_id` (soft deletes)
- **Multi-tenancy**: `security_context` integer field on all tables
- **Naming**: Snake_case column names
- **Relationships**: No foreign keys; managed in application layer
- **Defaults/Triggers**: Prohibited; all logic in application code

## Database
initial/examples in sql/010-inititial-create.sql (DDL) and sql/015-initialdata.sql (DML)

## API Design

### Endpoints

| Method | Path | Auth | CSRF | RBAC Grant | Description |
|--------|------|------|------|------------|-------------|
| GET | /health | No | No | N/A | Basic health check |
| POST | /auth | No | No | N/A | Username/password authentication |
| GET | /refresh | Yes | No | N/A | Refresh JWT token |
| POST | /changeSecurityContext | Yes | No | N/A | Switch security context |
| POST | /logoff | Yes | No | N/A | Clear authentication cookie |
| GET | /example | Yes | No | GET_EXAMPLE | Retrieve single example |
| GET | /exampleList | Yes | No | GET_EXAMPLE | Retrieve multiple examples |
| POST | /example | Yes | Yes | ADD_EXAMPLE | Create new example |
| POST | /exampleList | Yes | Yes | ADD_EXAMPLE | Bulk create examples |
| PUT | /example | Yes | Yes | CHG_EXAMPLE | Update example |
| PUT | /exampleList | Yes | Yes | CHG_EXAMPLE | Bulk update examples |
| DELETE | /example | Yes | Yes | DEL_EXAMPLE | Soft delete example |
| DELETE | /exampleList | Yes | Yes | DEL_EXAMPLE | Bulk soft delete examples |
| GET | /userList | Yes | No | GET_USER | Retrieve user list |
| POST | /user | Yes | Yes | ADD_USER | Create new user |
| PUT | /user | Yes | Yes | CHG_USER | Update user |
| DELETE | /user | Yes | Yes | DEL_USER | Delete user |
| POST | /addUserListToGroup | Yes | Yes | CHG_USER_GROUP | Add users to group |
| POST | /removeUserListFromGroup | Yes | Yes | CHG_USER_GROUP | Remove users from group |
| GET | /grantList | Yes | No | GET_USER | Retrieve available grants |
| GET | /userGroupList | Yes | No | GET_USER_GROUP | Retrieve user group list |
| POST | /userGroup | Yes | Yes | ADD_USER_GROUP | Create user group |
| PUT | /userGroup | Yes | Yes | CHG_USER_GROUP | Update user group |
| DELETE | /userGroup | Yes | Yes | DEL_USER_GROUP | Delete user group |

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
     "grants": [1001, 1002, 1003, 8001, 9001]
   }
   ```

## Security

### Authentication
- **JWT**: RS256 signed tokens with session management
- **Password Hashing**: Argon2id with time=3, memory=64MB, threads=4, key length=32
- **Session Management**: Session IDs stored in JWT, validated against shrmpl-kv for blocking

### Authorization (RBAC)
- **Grants**: Context-specific permissions in JWT (only current context's grants)
- **User Groups**: Collections of grants assigned to users per security context
- **Context Switching**: Users can switch between security contexts, JWT updated accordingly

### Security Checks (Per Request)
1. **JWT Validation**: Verify token signature and claims
2. **Rate Limiting**:
   - Authenticated: 200 requests/minute per user
   - Unauthenticated: 20 requests/minute per IP
3. **User Blocking**: Check shrmpl-kv for blocked users
4. **CSRF Protection**: Compare `X-CSRF-Token` header with last 3 chars of session ID
5. **RBAC**: Verify user has required grant for target security context

### Security Headers (All Responses)
- **Content-Security-Policy**: Restricts resource loading to prevent XSS
- **Strict-Transport-Security**: Forces HTTPS connections (1 year + subdomains + preload)
- **X-Frame-Options**: Prevents clickjacking attacks
- **X-Content-Type-Options**: Prevents MIME type sniffing
- **Referrer-Policy**: Controls referrer information leakage
- **Permissions-Policy**: Restricts browser features (camera, microphone, geolocation, etc.)
- **X-Permitted-Cross-Domain-Policies**: Restricts Flash cross-domain requests
- **Cache-Control**: Prevents caching of sensitive API responses

### Multi-tenancy
- **Security Context**: Integer identifier for tenant isolation
- **Data Segregation**: All queries filter by `security_context`
- **Grant Isolation**: Permissions scoped to security contexts, JWT contains only current context grants
- **Context Switching**: `/changeSecurityContext` endpoint allows users to switch contexts

## Configuration

### Environment Variables (.env format)
Environment files follow the naming convention `loc-{service}.env` for local development configurations.

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

# Testing (only in loc-api-srv.env)
TEST_BASE_URL=https://localhost:8443
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
- **Coverage**: Comprehensive API testing including:
  - Security context isolation and multi-tenancy
  - Error scenarios (401, 403, 404, 409, 410, 429, 400)
  - Bulk operations performance and correctness
  - Authentication and authorization flows
  - Optimistic locking and conflict resolution
  - Rate limiting and CSRF protection
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
- **Linting**: golangci-lint with `.golangci.yml` configuration
- **Testing**: Run `go test ./...` to validate changes
- **Documentation**: Inline comments, README updates

## Future Enhancements

### High Priority
- TOTP implementation for MFA
- Input validation middleware
- OpenAPI/Swagger documentation

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

## Recent Implementations

### Completed Features
- **Comprehensive Integration Tests**: 43 test cases covering authentication, authorization, CRUD operations, error scenarios, and security features
- **User Management**: Full CRUD operations for users with proper RBAC controls
- **User Group Management**: Group-based permission assignment and management
- **Security Context Switching**: Multi-tenant context switching with JWT updates
- **Enhanced Error Handling**: Specific error codes and improved logging
- **Database Optimization**: Strategic indexing for update_ts fields and optimistic locking
- **Code Quality**: golangci-lint configuration and comprehensive test coverage

This specification captures the current implementation and serves as a foundation for future development and maintenance of the ProtoStack-ReactGo API server.</content>
<parameter name="filePath">doc/105-api-srv-tech-spec.md
