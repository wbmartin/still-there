# TODO: Add Security Checks and Refactor writeJSON

## Overview
Refactor `writeJSON` from a global function in `common.go` to an `AppHelper` method with security checks. Ensure `writeJSON` and `WriteError` can only be called after all security checks are complete using the existing `CheckAllChecksComplete()` method.

## Key Changes
1. **Modify AppHelper writeJSON:**
   - Change `func writeJSON(...)` to `(ah *AppHelper) writeJSON(...)` in `app_helper.go`.
   - Add security check: `if !ah.CheckAllChecksComplete() { ah.appServices.Logger.Error("SEC0", "Attempted to write JSON before security checks complete"); http.Error(w, "Security check failed", http.StatusInternalServerError); return }`
   - Update the call in `WriteError` from `writeJSON(...)` to `ah.writeJSON(...)`.

2. **Update WriteError:**
   - Add the same security check at the start: `if !ah.CheckAllChecksComplete() { ... }` (before existing logging).
   - This ensures both success and error responses are gated.

3. **Remove Common.go writeJSON:**
   - Delete the `writeJSON` function from `common.go` (lines 14-24).

4. **Handler Refactors:**
   - Update all `writeJSON` calls in handlers (e.g., `example_handler.go`, `user_handler.go`, etc.) to `app.writeJSON(w, status, data)`.
   - Remove the `logger` parameter since it's now accessed via `ah.appServices.Logger`.

5. **Testing and Validation:**
   - Run Go tests to ensure no compilation errors.
   - Add unit tests for the security checks in `writeJSON` and `WriteError`.
   - Verify that calling `writeJSON` before checks fail appropriately.

## Tradeoffs and Considerations
- **Security Enforcement:** This adds defensive checks, but handlers already return early if checks fail. It's a safeguard against bugs.
- **Breaking Changes:** Handlers must be updated, but the refactor centralizes response writing in `AppHelper`.
- **No Feature Loss:** The method provides the same functionality, now with security validation.
- **Potential Issue:** If `CheckAllChecksComplete()` is called multiple times, ensure it's idempotent (it is, as it just checks internal state).