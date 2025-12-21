# React App
- MFA
  - investigate passcode as second factor - doc in 120

- react forms complaint - maybe should be using a library.
- user profile page
  - preferences 
  - upload avatar button.  
  









# API Server
- account lockout after failed login attempts, need to track failed login attempts and lock the account for a period of time after too many failed attempts.

- It is currently possible for someone with user update rights to add a group ID outside their context and gain access to another context.  The groups need to be checked to make sure they are within the users security context.

- need to compare https with http in the load test.


- Check on all repo functions, they should be receving the security context from users jwt claims.
- page number on the user list page should be part of state so a refresh or link takes to same place.

- move the database connection string, JWT, and the TOTP code to a shrmple vault.

- The `ExecuteWithRetry` logic is an excellent feature for handling transient database errors but is only applied to a single repository method. It should be applied to all external database calls.

- The `CORSMiddleware` is defined but not used in `main.go`.

 - The project uses a hand-rolled TCP client to send logs to a custom service (`shrmpl-log`) in a custom, fixed-width text format. This reinvents the wheel of structured logging, is likely not as performant or robust (e.g., reconnection logic is not thread-safe) as standard libraries, and creates a dependency on a non-standard service.
 
 - The hand-rolled `.env` parser is fragile and doesn't handle common cases like quoted values. Using a standard library like `joho/godotenv` would be more robust.
 
 - The `.golangci.yml` file exists but is empty, defeating its purpose. A good starter project should include a recommended set of linters.
 
 - The `CheckPasswordHash` function should use `subtle.ConstantTimeCompare` to prevent potential timing attacks.

 - **Introduce Unit Testing:** Use mocking frameworks (e.g., `pgxmock` for the database interface, or define a repository interface and mock it) to test handlers and other logic in isolation.
- **Complete Repository Coverage:** Address the "test environment issues" to un-skip the tests for `AddUserListToGroup` and `RemoveUserListFromGroup`.

# Architectural Todos
- **shrmpl-chk and shrmpl-mon** need to add rust web server that listens to a port (may not need to be https) and receives health information, only in the query string. reads a cronitor type schedule and makes slack calls when things go wrong. needs to get slack webhook url from the shrmpl key server
also need to add a binary that checks website health (probably just status) and records the checkins to shrmpl-mon.

- **S3 replacement**Non Sensitive file storing -There will be an nginx server running on the server and we should use it to server static non sensitive files. we need to explore what is the right way to do this.


# Starting prompt
make sure you look at my styles and quirks in the instructions. we are building a template/initial scaffold CRUD app in Go and React. in the doc file is an 810 file that describes how to construct a new entity, we're currently building the first entity, "example", and documenting what's important to facilitate future new entity creation for apps built on this scaffolding.
