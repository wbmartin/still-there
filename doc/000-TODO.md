# General List 
there is a new shrmpl-vault binary in the dist folder that needs to be committed and plugged into protostack.

# React App

- MFA
  - investigate passcode as second factor - doc in 120

- user profile page
  - preferences 
  - upload avatar button.  
  

# API Server


 - The project uses a hand-rolled TCP client to send logs to a custom service (`shrmpl-log`) in a custom, fixed-width text format. This reinvents the wheel of structured logging, is likely not as performant or robust  as standard libraries, and creates a dependency on a non-standard service.
 
 - The hand-rolled `.env` parser is fragile and doesn't handle common cases like quoted values. Using a standard library like `joho/godotenv` would be more robust.
 
 
 - page number on the user list page should be part of state so a refresh or link takes to same place.

 - **Introduce Unit Testing:** Use mocking frameworks (e.g., `pgxmock` for the database interface, or define a repository interface and mock it) to test handlers and other logic in isolation.

# Architectural Todos
- **shrmpl-chk and shrmpl-mon** need to add rust web server that listens to a port (may not need to be https) and receives health information, only in the query string. reads a cronitor type schedule and makes slack calls when things go wrong. needs to get slack webhook url from the shrmpl key server
also need to add a binary that checks website health (probably just status) and records the checkins to shrmpl-mon.

- **S3 replacement**Non Sensitive file storing -There will be an nginx server running on the server and we should use it to server static non sensitive files. we need to explore what is the right way to do this.

- need to compare https with http in the load test.


# Starting prompt
make sure you look at my styles and quirks in the instructions. we are building a template/initial scaffold CRUD app in Go and React. in the doc file is an 810 file that describes how to construct a new entity, we're currently building the first entity, "example", and documenting what's important to facilitate future new entity creation for apps built on this scaffolding.


# Code Review Prompt
**This works well for one AI to review what  you've been doing with antother AI**
Please perform a detailed code review of the API Server in the src/api-srv folder.  
- Background: This is a golang server project that will serv as a starter/base scaffold project for future projects.  It is intended to be paired iwth the web-app react app in src/web-app.  The goal is to have a well structured, secure, and maintainable codebase that follows best practices for golang web servers.
- Focus on inconsistencies, best practices, security, dead code, and note any tradeoffs made.  
- Don't look at the documentation in the doc folder because it may  bias the review.
- Output all of your results to /Users/brandon/Code/ProtoStack-ReactGo/doc/110-api-server-code-review.md.

Please perform a detailed code review of the WEB APP in the src/web-app folder.  
- Background: This is a react project that will serv as a starter/base scaffold project for future projects. It is intended to be paired with the api-srv in the sibling folder 
- Focus on inconsistencies, best practices, security, dead code, and note any tradeoffs made.  
- Don't look at the documentation in the doc folder because it may  bias the review.
- Output all of your results to /Users/brandon/Code/ProtoStack-ReactGo/doc/215-react-web-code-review.md.




=====
Thought to be complete

-   **Hardcoded Configuration:** The API proxy URL in `vite.config.ts` is hardcoded (`http://localhost:8080`) and includes a comment that it requires manual editing. This should be loaded from an environment variable (`.env` file) for a better developer experience.
-   **Inconsistent Routing Definitions:** The router configuration in `App.tsx` mixes hardcoded path strings (e.g., `/mfa-setup`) with constants from `constants/routes.ts`. It should exclusively use the constants for better maintainability.

-   **Dead Code:** The `src/services` directory is empty, and the `use-examples-sync.ts` hook is disabled. These should be removed to reduce clutter and confusion.

-   **Inconsistent Styling:** The project uses the powerful Mantine library but occasionally falls back to basic inline styles (`style={{...}}`). To fully leverage Mantine's theming and styling system, all styling should be done via Mantine props (`sx`, `style` functions) or layout components (`<Flex>`, `<Stack>`, etc.).

- react forms complaint - maybe should be using a library.


- Add account lockout after failed login attempts, need to track failed login attempts and lock the account for a period of time after too many failed attempts.  When a user fails a login attempt we need to increment two entries to the KV store that tracks both the IP address of the failed attempt and the username.  the keys could be something like "loginfail-ip-<ip address>" and "loginfail-user-<username>".  each failed attempt increments the count and sets an expiration time of say 5 minutes.  if either the IP address or the username exceeds a threshold (say 7 failed attempts) then we lock the account for a period of time (say 5 minutes).  during the lockout period any login attempts for that username or from that IP address are rejected immediately.  That means the login handler needs to check the KV store for both keys before attempting to validate the password. we'll als need to create test for this feature.

- The `CORSMiddleware` is defined but not used in `main.go`.

- The `ExecuteWithRetry` logic is an excellent feature for handling transient database errors but is only applied to a single repository method. It should be applied to all external database calls.

 - The `CheckPasswordHash` function should use `subtle.ConstantTimeCompare` to prevent potential timing attacks.

 - shrmple log client reconnection logic is not thread-safe)

 - shrmpl-kv client is not handling list correctly right now.
 
 - It is currently possible for someone with user update rights to add a group ID outside their context and gain access to another context.  The groups need to be checked to make sure they are within the users security context.

 - Check on all repo functions, they should be receving the security context from users jwt claims.
 
  - The `.golangci.yml` file exists but is empty, defeating its purpose. A good starter project should include a recommended set of linters.
  
- move the database connection string, JWT, and the TOTP code to a shrmple vault.


- **Complete Repository Coverage:** Address the "test environment issues" to un-skip the tests for `AddUserListToGroup` and `RemoveUserListFromGroup  `.
