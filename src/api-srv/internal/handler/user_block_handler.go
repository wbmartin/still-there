package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/security"
)

// Blocking a login (B034) is three writes, because the KV is in-process:
//
//   - account_status = blocked, the durable half: /auth refuses any login
//     that is not active, so a blocked user cannot sign in again;
//   - every session of the login is revoked (RevokeUserSessions), so a
//     token minted before the block gets 401 from CheckJWT and the web app
//     signs out, and an unblock does not revive it;
//   - the BLOCKED*<user name> KV flag, which CheckRateLimitAndUserBlock
//     answers 403. Revocations are lost on a restart; the flag is not: the
//     API reloads every blocked login's flag at start (LoadBlockedUsers).
//     After a restart a pre-block token therefore gets 403 "User account is
//     blocked" on each call instead of 401, and the web app (B006) does not
//     sign out on 403, until the token expires. The flag lives
//     JWT_SESSION_EXPIRE_HOURS, the longest a token can.
//
// Unblocking sets the row active and clears the flag. The `admin` subject's
// exemption from the rate limit and block check stands; its row still stops
// it signing in. No caller blocks their own login.

// blockedKey is the KV flag CheckRateLimitAndUserBlock reads.
func blockedKey(userName string) string { return "BLOCKED*" + userName }

// LoadBlockedUsers sets the block flag of every login in userNameList; the
// server calls it at start with every blocked login.
func LoadBlockedUsers(
	appServices *app.Services, userNameList []string,
) error {
	ttl := strconv.Itoa(appServices.Config.JWTSessionExpireHours) + "h"
	for _, userName := range userNameList {
		err := appServices.KV.Set(blockedKey(userName), "1", ttl)
		if err != nil {
			return fmt.Errorf("block flag for %s: %w", userName, err)
		}
	}
	return nil
}

// UserBlockRepository is the slice of the user repository the two block
// routes need.
type UserBlockRepository interface {
	RetrieveUserByUserName(
		ctx context.Context,
		username string,
	) (*model.AppUserModel, error)
	UpdateUserAccountStatus(
		ctx context.Context,
		userID int,
		accountStatus string,
		expectedUpdateTS float64,
		updateUserID int,
		securityContext int,
	) (float64, error)
}

// AdminBlockUserHandler handles POST /admin/blockUser (block = true) and
// POST /admin/unblockUser (block = false), body {"userName": ...}, grant
// BLOCK_USER (9008). The target must be a login of the caller's active
// security context; any other name is 404, never 403.
func AdminBlockUserHandler(
	appServices *app.Services,
	userBlockRepository UserBlockRepository,
	block bool,
) http.HandlerFunc {
	name, logCode := "AdminUnblockUserHandler", "ADM_UNBLK"
	if block {
		name, logCode = "AdminBlockUserHandler", "ADM_BLK"
	}
	return func(w http.ResponseWriter, r *http.Request) {
		appHelper := security.NewAppHelper(w, r, appServices, name)
		appHelper.CheckJWT()
		appHelper.CheckRateLimitAndUserBlock()
		appHelper.CheckCSRF()
		appHelper.CheckAuthorization(model.GrantUserBlock)
		if !appHelper.CheckAllChecksComplete() {
			return
		}

		var req model.BlockUserRequest
		if !decodeJSONBody(appHelper, w, r, maxLoginLayerBodyBytes, &req) {
			return
		}
		if req.UserName == "" {
			appHelper.WriteError(w, http.StatusBadRequest,
				"userName is required", "BLK_NO_USER", nil)
			return
		}
		claims := appHelper.GetClaims()
		if block && req.UserName == claims.Subject {
			appHelper.WriteError(w, http.StatusBadRequest,
				"You cannot block your own login", "BLK_SELF", nil)
			return
		}

		target, err := userBlockRepository.RetrieveUserByUserName(
			r.Context(), req.UserName)
		if err != nil {
			mapRepoError(appHelper, w, err, "Failed to read user", "BLK_READ")
			return
		}
		if target.SecurityContext != claims.SecurityContext {
			// another tenant's login: not found, never forbidden
			appHelper.WriteError(w, http.StatusNotFound, "Not found",
				"NOT_FOUND_ERROR", nil)
			return
		}

		want, from := model.AccountStatusActive, model.AccountStatusBlocked
		if block {
			want, from = model.AccountStatusBlocked, model.AccountStatusActive
		}
		if target.AccountStatus != from {
			appHelper.WriteError(w, http.StatusConflict,
				fmt.Sprintf("%s is %s, not %s", target.UserName,
					target.AccountStatus, from),
				"BLK_STATE", nil)
			return
		}

		newUpdateTS, err := userBlockRepository.UpdateUserAccountStatus(
			r.Context(), target.AppUserID, want, target.UpdateTS,
			claims.UserID, target.SecurityContext,
		)
		if err != nil {
			mapRepoError(appHelper, w, err, "Failed to change the block",
				"BLK_WRITE")
			return
		}

		// The row is committed; the flag and revocation follow it. A failed
		// KV write is answered 500, never hidden; the row stays as written.
		if block {
			err = LoadBlockedUsers(appServices, []string{target.UserName})
		} else {
			err = appServices.KV.Del(blockedKey(target.UserName))
		}
		if err != nil {
			appHelper.WriteError(w, http.StatusInternalServerError,
				"Saved; the session flag failed", "BLK_KV", err)
			return
		}
		if block && !revokeUserSessionList(
			appHelper, w, appServices, target.UserName,
		) {
			return
		}

		appHelper.WriteJSON(w, http.StatusOK, model.DataWrapper{
			Data: map[string]any{
				"userName":      target.UserName,
				"accountStatus": want,
				"updateTs":      newUpdateTS,
			},
		})
		appHelper.Info(logCode, name+" completed for user: "+target.UserName)
	}
}
