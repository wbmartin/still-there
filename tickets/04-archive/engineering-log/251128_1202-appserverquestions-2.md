> Archived from `doc/archive/102-appserverquestions.md` on 2026-09-21 by PS0006; point-in-time, not maintained.

# 102 - Remaining App Server Questions

After reviewing your answers to the 101 questions, I have a few remaining clarifications:

## Infrastructure Implementation

1. **shrmpl Clients**: You mentioned checking the Go example in shrmpl-srv-preview. Since I can't access external repos directly, can you provide the key code snippets or interfaces for the shrmpl-kv and shrmpl-log Go clients? What methods do they expose (e.g., for KV: Get, Set, Incr, Exists; for Log: Log method)?

I copied the rep into examples/shrmpl-srv-preview

2. **Bin Script**: For downloading the shrmpl executables from dist/mac, should this be a shell script called `bin/download-shrmpl.sh` that uses curl/wget to download the binaries? What should it do if the files already exist?

call it bin/010-download-shrmpl.sh and use curl

## Configuration

3. **Config File Location**: You mentioned creating a new config in the "etc folder". Should this be `etc/config.env` or `etc/app-config.env`? Should it follow the .env format from the specs exactly?
don't care.  i would call it etc/loc-dev-config.json to follow the example i gave you.

## Development Tools

4. **Testing Commands**: Since you said "you tell me if we need them", and the specs mention lint and typecheck, should we include basic Go commands like `go vet`, `go fmt`, and `golangci-lint` in the build process? What specific commands should be run?

let's defer this till we get it up and running.  just keep the requirement to run tests in

## Schema Confirmation

5. **Note Table Fields**: Confirming the note table from specs:
   ```
   example_id serial
   example_text text
   example_int integer
   example_float float
   example_int_array integer[]
   example_float_array float[]
   security_context integer
   create_ts numeric(17,7)
   create_user_id integer
   update_ts numeric(17,7)
   update_user_id integer
   delete_ts numeric(17,7)
   delete_user_id integer
   ```
   Is `example_float` supposed to be `float` or `numeric`? And `example_float_array` should be `float[]` or `numeric[]`?
   the first word is the field name so it should be example_float  the second word is the data type so float.

6. **Grant IDs**: Should I use the exact grant IDs from the specs (1001-1004 for notes, 8001, 9001-9003), or adjust them? You said "you can adjust them, but you should have a good reason."

start with what I said.

## Scope Confirmation

7. **Initial Implementation**: For the first iteration (auth + CRUD), which specific endpoints should be included?
   - /health
   - /auth (login)
   - /logoff
   - /note (GET, POST, PUT, DELETE)
   - Any others?

   seems like you should have /auth that will receive a username and password and return a temporary token and a second /auth2 for the second factor which will receive the token from the previous /auth exchange and the time based one time password from the user.  We should also add a field to the app_user table called mfa_exempt as a boolean.  If that boolean is true, /auth will return the full jwt token.  if app_user.mfa_exempt = false, it  will return the token to prepare for the auth2 exchange.

8. **Security Context in app_user_grant**: You said "If it improves consistency, you can add a security_context and we'll set it to 0". Should I add it to maintain consistency, or keep it without for now?

let's add it.
</content>
<parameter name="filePath">doc/102-appserverquestions.md
