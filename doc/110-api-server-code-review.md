# API Server Code Review

- **Date:** 2025-12-24
- **Reviewer:** Gemini CLI

## 1. Executive Summary

This code review of the Go API server found a well-structured project with a clear separation of concerns (handler, repository, model). The code demonstrates a strong awareness of several security best practices, including the use of the `argon2id` password hashing algorithm, constant-time comparisons to prevent timing attacks, and the implementation of security headers.

However, the review identified one **critical architectural flaw** and several **high-risk security vulnerabilities** that undermine these positive aspects. The core issue is the complete absence of a standard middleware pattern for handling cross-cutting concerns like authentication, authorization, and CSRF protection. Instead, this logic is manually orchestrated within each HTTP handler using a stateful `AppHelper` object, a pattern that is brittle, error-prone, and a significant departure from idiomatic Go practices.

The most urgent recommendations are to **refactor the security model to use a standard middleware chain** and to **remediate the insecure handling of MFA TOTP seeds**.

## 2. Critical & High-Risk Findings

### 2.1. Critical: Absence of a Middleware Pattern for Security

**Observation:**
The application does not use a middleware chain for handling security. Instead, each authenticated HTTP handler relies on an `app_lib.AppHelper` object and is responsible for manually invoking a series of security checks in the correct order:

*File: `src/api-srv/internal/handler/user_handler.go`*
```go
// Boilerplate code at the start of every authenticated handler
app := app_lib.NewAppHelper(w, r, h.appsrv)
app.CheckJWT()
app.CheckRateLimitAndUserBlock()
app.CheckCSRF()
app.CheckAuthorization(model.GrantUserRead)

if !app.CheckAllChecksComplete() {
    return
}
```

**Impact:**
This pattern is a critical design flaw for several reasons:
- **Error-Prone:** A developer can easily forget a check, misorder them, or fail to call `CheckAllChecksComplete()`, creating a security vulnerability.
- **Violation of DRY (Don't Repeat Yourself):** The same block of security boilerplate is repeated across all handlers, leading to code bloat and making universal changes difficult.
- **Maintenance Nightmare:** Adding a new global security check (e.g., a new logging step) requires modifying every single handler in the codebase.
- **Un-idiomatic:** This approach is contrary to the standard, established practice of using `http.Handler` middleware chains in Go for managing request lifecycles.

**Recommendation:**
Refactor the application to use a standard middleware architecture.
1.  Create individual middleware functions for each security concern (e.g., `JWTMiddleware`, `CSRFMiddleware`, `RBACMiddleware`).
2.  In `main.go`, wrap the protected routes with this middleware chain. This declaratively and reliably enforces security on all relevant endpoints.

**Example of a middleware-based approach:**
```go
// In main.go
authedMux := http.NewServeMux()
authedMux.HandleFunc("/user", userHandler.RetrieveUserList)
// ... other protected routes

// Chain the middleware
var protectedHandler http.Handler = authedMux
protectedHandler = rbacMiddleware(protectedHandler, model.GrantUserRead) // Innermost
protectedHandler = csrfMiddleware(protectedHandler)
protectedHandler = jwtAuthMiddleware(protectedHandler) // Outermost

mux.Handle("/user", protectedHandler)
```

### 2.2. High: Insecure Encryption of MFA TOTP Seeds

**Observation:**
The MFA TOTP seeds, which are critical security secrets, are encrypted using AES-GCM with a single, global encryption key stored in the application's configuration file.

*File: `src/api-srv/internal/handler/user_handler.go`*
```go
// Encryption call
encrypted, err := encryptTOTPSeed(secret, appServices.Config.TOTPEncryptionKey)

// Decryption call
decryptedSecret, err := decryptTOTPSeed(
    *user.TOTPSeed,
    appServices.Config.TOTPEncryptionKey,
)
```

**Impact:**
If an attacker gains read-access to the server's environment or configuration file, they can retrieve this global `TOTPEncryptionKey`. With this key, they can decrypt the `totp_seed` for **every user in the database**, completely bypassing the MFA protection for the entire application.

**Recommendation:**
Do not use a single, static key for encrypting all user secrets. Consider one of the following approaches:
- **Use a Key Management Service (KMS):** Integrate with a service like AWS KMS, Google Cloud KMS, or HashiCorp Vault to manage encryption keys. This is the most secure option.
- **Derive a User-Specific Key:** Derive the encryption key from a combination of a global application secret (a "pepper") and a value unique to the user (like the user's password hash or a dedicated secret). This ensures that compromising the global secret alone is not enough to decrypt all user data.

### 2.3. High: Weak CSRF Token Generation

**Observation:**
The Cross-Site Request Forgery (CSRF) token is generated from a predictable, low-entropy substring of the session ID stored in the JWT.

*File: `src/api-srv/internal/app_lib/app_helper.go`*
```go
expectedToken := sh.claims.SessionID[len(sh.claims.SessionID)-3:]
```

**Impact:**
A CSRF token's security relies on its randomness and unpredictability. By using only the last 3 characters of the session ID, the token's entropy is drastically reduced, making it potentially guessable and weakening the protection it offers.

**Recommendation:**
Implement the standard "double-submit cookie" pattern for CSRF protection.
1.  Upon login, generate a high-entropy, random string to be the CSRF token.
2.  Store this token in a `HttpOnly=false`, `SameSite=Strict` cookie that is accessible to the frontend JavaScript.
3.  The frontend must read this token from the cookie and include it in a request header (e.g., `X-CSRF-Token`) for all state-changing requests (`POST`, `PUT`, `DELETE`).
4.  The server-side middleware then simply checks if the value in the cookie matches the value in the header.

## 3. Other Findings & Recommendations

### 3.1. Code Structure & Best Practices

- **Handler Cohesion (Medium):** Files like `user_handler.go` are overly large and violate the Single Responsibility Principle. They manage user CRUD, MFA setup, password changes, and more.
    - **Recommendation:** Split these large handlers into smaller, more focused files (e.g., `mfa_handler.go`, `password_handler.go`).

- **Password Hash Verification (Low):** The `CheckPasswordHash` function in `auth.go` does not parse the Argon2 parameters from the hash string. It uses hardcoded constants.
    - **Impact:** This prevents future upgrades to the hashing parameters (e.g., increasing memory/time cost) without invalidating all existing user passwords.
    - **Recommendation:** Modify `CheckPasswordHash` to parse the `m=`, `t=`, and `p=` values from the hash string and use them during verification.

- **Symmetric JWT Algorithm (Low - Tradeoff):** The server uses `HS256` for JWT signing.
    - **Tradeoff:** `HS256` is a symmetric algorithm, meaning the same key is used to sign and verify tokens. While simple and acceptable for a monolithic service, an asymmetric algorithm (like `RS256` or `ES256`) is more secure and scalable, as it allows verification with a public key without exposing the private signing key. This was likely a deliberate tradeoff for simplicity. No immediate action is required, but it's worth noting for future architecture decisions.

- **Hardcoded Rate Limit (Low):** The `CheckRateLimitAndUserBlock` function in `app_helper.go` contains a hardcoded rate limit of `100`, which is inconsistent with the configurable limit used elsewhere.
    - **Recommendation:** Remove the hardcoded value and read the limit from the `appServices.Config` object to ensure consistency.

### 3.2. Security Context Enforcement

- **Redundant Validation (Low):** Handlers frequently contain code to validate that all entities in a request belong to the operator's current `SecurityContext`.
    - **Observation:** This suggests a lack of trust in the repository layer. The repository should be the ultimate authority for data isolation. A query to the repository should *never* be able to return data from outside the provided security context.
    - **Recommendation:** Move all `SecurityContext` enforcement logic into the repository layer. The handlers should not need to perform these redundant checks. This will simplify handler logic and strengthen the data isolation guarantee.

## 4. Conclusion

The API server has a solid foundation but is undermined by the decision to forgo a standard middleware architecture. The current `AppHelper` pattern is a critical flaw that is insecure, difficult to maintain, and a source of further bugs and vulnerabilities.

**Priority of actions should be:**
1.  **Remediate the critical flaws:** Replace the `AppHelper` with a proper middleware chain and fix the insecure TOTP seed encryption.
2.  **Fix the high-risk CSRF vulnerability.**
3.  Address the other findings, such as improving handler cohesion and strengthening data layer enforcement of security contexts.

By adopting a standard middleware pattern, the project will become significantly more secure, robust, and easier to maintain.