package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/repository"
	"github.com/wbmartin/starter-api/internal/security"
)

// maxAuthBodyBytes caps a body read with no session or before the auth
// gate: /auth, /mfaSetup, /mfaVerify, /changeSecurityContext and the public
// /userSelfCreate. Each carries a credential, a temp token or one id.
const maxAuthBodyBytes int64 = 4 << 10

// maxLoginLayerBodyBytes caps every signed-in login-layer write: user,
// group, membership, password and MFA-reset routes. 64 KiB holds a
// membership list of thousands of ids.
const maxLoginLayerBodyBytes int64 = 64 << 10

// decodeJSONBody caps r.Body at maxBytes and decodes it into dst. On failure
// it has already answered (413 over the cap, 400 otherwise) and returns false.
func decodeJSONBody(
	app *security.AppHelper,
	w http.ResponseWriter,
	r *http.Request,
	maxBytes int64,
	dst any,
) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeBodyError(app, w, err)
		return false
	}
	return true
}

// readBody is decodeJSONBody for a handler that needs the raw bytes.
func readBody(
	app *security.AppHelper,
	w http.ResponseWriter,
	r *http.Request,
	maxBytes int64,
) ([]byte, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeBodyError(app, w, err)
		return nil, false
	}
	return body, true
}

// writeBodyError answers a failed body read: 413 when the cap was hit, else
// 400. The decoder's text goes to the log only, never to the client.
func writeBodyError(
	app *security.AppHelper, w http.ResponseWriter, err error,
) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		app.WriteError(w, http.StatusRequestEntityTooLarge,
			"Request body too large", "BODY_TOO_LARGE", err)
		return
	}
	app.WriteError(w, http.StatusBadRequest, "Invalid request body",
		"INV_REQ_BODY", err)
}

// authCookieName is the cookie that carries the JWT on every authenticated
// request.
const authCookieName = "auth"

// The auth cookie's Max-Age values. The JWT's exp is the only session
// lifetime: every path that issues a token (login, MFA verify, refresh,
// context switch) sets a browser-session cookie with no Max-Age, so the
// browser never drops a token the server would still accept. Logoff
// expires the cookie.
const (
	authCookieSessionMaxAge = 0
	authCookieExpireNow     = -1
)

// setAuthCookie writes jwt as the HttpOnly, Secure, SameSite=Strict auth
// cookie. maxAge is authCookieSessionMaxAge or authCookieExpireNow.
func setAuthCookie(w http.ResponseWriter, jwt string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    jwt,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
	})
}

// mapRepoError answers a failed repository call. A sentinel decides the
// status and a generic message, or the repository's ClientMessageError text
// when it carries one; anything else is the server's fault and answers 500
// with failMsg and failCode, the error's text going to the log only.
// Handlers never pick a status by matching an error's text.
func mapRepoError(
	app *security.AppHelper,
	w http.ResponseWriter,
	err error,
	failMsg string,
	failCode string,
) {
	status, code, msg := repoErrorAnswer(err)
	if status == http.StatusInternalServerError {
		code, msg = failCode, failMsg
	}
	app.WriteError(w, status, msg, code, err)
}

// repoErrorAnswer is mapRepoError's table: status, log code and client
// message for err. A soft-deleted row is answered like a missing one, so the
// caller cannot tell that the id once existed.
func repoErrorAnswer(err error) (status int, code string, msg string) {
	var clientErr *repository.ClientMessageError
	detailed := errors.As(err, &clientErr) && clientErr.Message != ""
	withDetail := func(generic string) string {
		if detailed {
			return clientErr.Message
		}
		return generic
	}

	switch {
	case errors.Is(err, repository.ErrValidation):
		return http.StatusBadRequest, "VALIDATION_ERROR",
			withDetail("The request is not valid")
	case errors.Is(err, repository.ErrDuplicate):
		return http.StatusConflict, "DUPLICATE_ERROR",
			withDetail("A record with that name already exists")
	case errors.Is(err, repository.ErrInUse):
		return http.StatusConflict, "IN_USE_ERROR",
			withDetail("The record is still in use")
	case errors.Is(err, repository.ErrConflict):
		return http.StatusConflict, "CONFLICT_ERROR",
			withDetail("The record was modified by another user")
	case errors.Is(err, repository.ErrNotFound),
		errors.Is(err, repository.ErrDeleted):
		return http.StatusNotFound, "NOT_FOUND_ERROR", withDetail("Not found")
	case errors.Is(err, repository.ErrExpectationFailed):
		return http.StatusExpectationFailed, "EXPECTATION_FAILED",
			withDetail("One or more records could not be updated " +
				"due to concurrent modifications")
	}
	return http.StatusInternalServerError, "INTERNAL_ERROR",
		"The request could not be completed"
}

// ParseCommaSeparatedInts parses a comma-separated string of integers
// into a slice of ints
func ParseCommaSeparatedInts(s string) ([]int, error) {
	if s == "" {
		return []int{}, nil
	}
	idStrList := strings.Split(s, ",")
	idList := make([]int, 0, len(idStrList))
	for _, idStr := range idStrList {
		id, err := strconv.Atoi(strings.TrimSpace(idStr))
		if err != nil {
			return nil, err
		}
		idList = append(idList, id)
	}
	return idList, nil
}

// ParseCommaSeparatedFloats parses a comma-separated string of floats
// into a slice of float64
func ParseCommaSeparatedFloats(s string) ([]float64, error) {
	if s == "" {
		return []float64{}, nil
	}
	strList := strings.Split(s, ",")
	floatList := make([]float64, 0, len(strList))
	for _, str := range strList {
		f, err := strconv.ParseFloat(strings.TrimSpace(str), 64)
		if err != nil {
			return nil, err
		}
		floatList = append(floatList, f)
	}
	return floatList, nil
}

// GetIntFromQuery safely parses an integer from query parameters (max 10 chars)
func GetIntFromQuery(r *http.Request, key string) (int, error) {
	val := r.URL.Query().Get(key)
	if val == "" {
		return 0, fmt.Errorf("missing required parameter: %s", key)
	}
	if len(val) > 10 {
		return 0, fmt.Errorf("parameter %s too long: max 10 characters", key)
	}
	num, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: must be integer", key)
	}
	return num, nil
}

// GetFloatFromQuery safely parses a float from query parameters (max 20 chars)
func GetFloatFromQuery(r *http.Request, key string) (float64, error) {
	val := r.URL.Query().Get(key)
	if val == "" {
		return 0, fmt.Errorf("missing required parameter: %s", key)
	}
	if len(val) > 20 {
		return 0, fmt.Errorf("parameter %s too long: max 20 characters", key)
	}
	num, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: must be number", key)
	}
	return num, nil
}

// GetStringFromQuery safely retrieves a string from query parameters
// (max 255 chars)
func GetStringFromQuery(r *http.Request, key string) (string, error) {
	val := r.URL.Query().Get(key)
	if val == "" {
		return "", fmt.Errorf("missing required parameter: %s", key)
	}
	if len(val) > 255 {
		return "", fmt.Errorf("parameter %s too long: max 255 characters", key)
	}
	return val, nil
}

// GetOptionalStringFromQuery safely retrieves an optional string
// from query parameters (max 255 chars if present)
func GetOptionalStringFromQuery(r *http.Request, key string) (string, error) {
	val := r.URL.Query().Get(key)
	if val == "" {
		return "", nil
	}
	if len(val) > 255 {
		return "", fmt.Errorf("parameter %s too long: max 255 characters", key)
	}
	return val, nil
}

// GetCommaSeparatedIntListFromQuery parses a comma-separated list of integers
// from query parameters (optional, returns empty slice if missing)
func GetCommaSeparatedIntListFromQuery(r *http.Request,
	key string) ([]int, error) {
	str, err := GetOptionalStringFromQuery(r, key)
	if err != nil {
		return nil, err
	}
	return ParseCommaSeparatedInts(str)
}

// GetCommaSeparatedFloats parses a comma-separated list of floats
// from query parameters (optional, returns empty slice if missing)
func GetCommaSeparatedFloats(r *http.Request, key string) ([]float64, error) {
	str, err := GetOptionalStringFromQuery(r, key)
	if err != nil {
		return nil, err
	}
	return ParseCommaSeparatedFloats(str)
}

// GetOptionalFloatFromQuery safely parses an optional float
// from query parameters (max 20 chars if present)
func GetOptionalFloatFromQuery(r *http.Request, key string) (float64, error) {
	str, err := GetOptionalStringFromQuery(r, key)
	if err != nil {
		return 0, err
	}
	if str == "" {
		return 0, nil
	}
	num, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: must be number", key)
	}
	return num, nil
}

// CurrentTimestamp returns the current timestamp as a float64
// (seconds since Unix epoch with nanosecond precision)
func CurrentTimestamp() float64 {
	return float64(time.Now().UnixNano()) / 1e9
}

// GetOptionalIntFromQuery reads an integer query parameter, returning 0 when it
// is absent. Used for optional narrowing filters where 0 means "no filter"
// rather than "invalid".
func GetOptionalIntFromQuery(r *http.Request, key string) (int, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return parsed, nil
}

// revokeUserSessionList ends every session of the named logins: their tokens
// carry the identity and grants a write just changed. The write has
// committed; a failed revocation is answered 500, never hidden.
func revokeUserSessionList(
	appHelper *security.AppHelper,
	w http.ResponseWriter,
	appServices *app.Services,
	userNameList ...string,
) bool {
	for _, userName := range userNameList {
		if userName == "" {
			continue
		}
		err := security.RevokeUserSessions(appServices, userName, "")
		if err != nil {
			appHelper.WriteError(
				w,
				http.StatusInternalServerError,
				"Saved; ending the login's open sessions failed",
				"SES_REVOKE",
				err,
			)
			return false
		}
	}
	return true
}

// userNameListOf returns the user_name of each login in userList.
func userNameListOf(userList []*model.AppUserModel) []string {
	userNameList := make([]string, 0, len(userList))
	for _, user := range userList {
		userNameList = append(userNameList, user.UserName)
	}
	return userNameList
}

// userNameOf returns the current user_name of one login of securityContext,
// "" when there is none: the name its tokens carry, read before a write that
// may rename or delete it.
func userNameOf(
	r *http.Request,
	userRepo UserRepositoryInterface,
	securityContext int,
	userID int,
) (string, error) {
	userList, err := userRepo.RetrieveUserList(
		r.Context(), securityContext, []int{userID}, 0,
	)
	if err != nil || len(userList) == 0 {
		return "", err
	}
	return userList[0].UserName, nil
}
