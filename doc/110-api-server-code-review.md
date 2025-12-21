# Code Review: `src/api-srv`

**Generated:** 2025-12-21

## Overall Assessment

The project is a solid, functional API server with a strong emphasis on security and data integrity. It makes a clear architectural tradeoff: **minimize dependencies in favor of using the Go standard library and hand-rolled solutions**. This makes the project transparent and self-contained but comes at the cost of not using common, powerful libraries that could improve robustness and developer experience.

The data model and repository layers are strong, while the web/handler layer and overall testing strategy have the most significant areas for improvement. The code is generally easy to read and understand.

## Key Strengths (Best Practices)

-   **Security Posture:** The project demonstrates a strong commitment to security.
    -   **Password Hashing:** Use of `argon2id` is the current best practice.
    -   **HTTP Headers:** A comprehensive set of security headers is applied via middleware.
    -   **Session Management:** Standard JWTs are used for authentication tokens.
    -   **Data Serialization:** Sensitive fields like `PwHash` are correctly marked with `json:"-"` to prevent them from ever being serialized into API responses.
-   **Data Integrity & Access Layer:** The data layer is well-designed and robust.
    -   **Optimistic Locking:** Update and delete operations use the `update_ts` field to prevent lost updates in concurrent situations. This is implemented correctly.
    -   **Transactions:** Batch operations (e.g., `AddUserListToGroup`) correctly use database transactions to ensure atomicity.
    -   **Soft Deletes:** The use of `delete_ts` and `delete_user_id` is a solid implementation of the soft delete pattern.
    -   **Sentinel Errors:** Defining package-level errors like `ErrNotFound` and `ErrConflict` is an excellent practice that allows calling code to handle specific error cases cleanly.
-   **Permissions Model:** The `grants.go` file provides a clear, centralized, and auditable model for Role-Based Access Control (RBAC).

## Primary Areas for Improvement

### 1. Web Layer Architecture (Handlers & Middleware)

-   **Finding:** The project avoids web frameworks and instead implements a custom "check" pattern inside every handler (e.g., `app.CheckJWT()`, `app.CheckCSRF()`, etc.) rather than using a standard middleware chain.
-   **Impact:** This pattern is non-idiomatic, verbose, and error-prone. A developer adding a new handler could easily forget a critical check, leading to a security vulnerability. The flow of control (`CheckAllChecksComplete`) is also not immediately obvious.
-   **Recommendation:** Refactor to use a standard middleware pattern. This is perfectly achievable with the standard `net/http` library by creating functions that wrap `http.Handler`s. This would centralize authentication, authorization, rate-limiting, and logging logic cleanly and apply it consistently to all relevant endpoints, removing duplication from the handlers.

### 2. Testing Strategy

-   **Finding:** The project contains repository-level **integration tests** that connect to a live database, which is valuable. However, it appears to lack **unit tests** entirely. Logic within the HTTP handlers and `app_lib` is untested. Furthermore, coverage is incomplete, with key repository tests for batch operations explicitly skipped.
-   **Impact:** Business logic is not tested in isolation, making refactoring or adding new features risky. Without handler tests, it's impossible to verify that security checks and routing are correctly configured.
-   **Recommendation:**
    1.  **Introduce Unit Testing:** Use mocking frameworks (e.g., `pgxmock` for the database interface, or define a repository interface and mock it) to test handlers and other logic in isolation.
    2.  **Complete Repository Coverage:** Address the "test environment issues" to un-skip the tests for `AddUserListToGroup` and `RemoveUserListFromGroup`.
    3.  **Improve Test Cleanup:** Switch from manual `DELETE` statements to wrapping each test in a database transaction that is rolled back at the end. This provides perfect test isolation.

### 3. Data Scanning in Repository (Brittle Code)

-   **Finding:** The repositories use manual `row.Scan(&field1, &field2, ...)` calls to populate model structs. The models themselves have `db` tags which are currently unused.
-   **Impact:** This code is brittle. If a developer changes the column order in a `SELECT` statement or adds a column without perfectly matching the `Scan` call, the application will fail at runtime.
-   **Recommendation:** Introduce a lightweight library like `sqlx`. A single call to `sqlx.Get(&user, ...)` or `sqlx.Select(&users, ...)` can safely and automatically scan data into structs using the existing `db` tags. This would make the data access code much more robust, readable, and maintainable.

### 4. Custom Logging Implementation

-   **Finding:** The project uses a hand-rolled TCP client to send logs to a custom service (`shrmpl-log`) in a custom, fixed-width text format.
-   **Impact:** This reinvents the wheel of structured logging, is likely not as performant or robust (e.g., reconnection logic is not thread-safe) as standard libraries, and creates a dependency on a non-standard service.
-   **Recommendation:** Replace the custom logger with the standard library's `slog` (available in Go 1.21+). It can be configured to output structured JSON to `stdout`, which is a more flexible and conventional approach that integrates with any standard log aggregation tool (Fluentd, Logstash, Vector, etc.).

## Secondary Findings & Inconsistencies

-   **Configuration Loading:** The hand-rolled `.env` parser is fragile and doesn't handle common cases like quoted values. Using a standard library like `joho/godotenv` would be more robust.
-   **Linter Configuration:** The `.golangci.yml` file exists but is empty, defeating its purpose. A good starter project should include a recommended set of linters.
-   **Inconsistent Resiliency:** The `ExecuteWithRetry` logic is an excellent feature for handling transient database errors but is only applied to a single repository method. It should be applied to all external database calls.
-   **Unused Code:** The `CORSMiddleware` is defined but not used in `main.go`.
-   **Non-RESTful Routing:** API routes like `GET /exampleList` and `POST /userSelfCreate` are not RESTful. A more standard approach would be `GET /examples` and `POST /users`.
-   **Timestamps:** The use of `float64` for timestamps is unusual. The idiomatic Go type is `time.Time`, which provides better type safety and functionality.
-   **Constant-Time Comparison:** The `CheckPasswordHash` function should use `subtle.ConstantTimeCompare` to prevent potential timing attacks.
