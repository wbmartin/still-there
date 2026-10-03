package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/auth"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/repository"
	"github.com/wbmartin/starter-api/internal/security"
)

// UserRepositoryInterface defines the methods needed by UserHandler
type UserRepositoryInterface interface {
	CreateUser(
		ctx context.Context,
		user *model.AppUserModel,
		securityContext int,
	) (*model.AppUserModel, error)
	RetrieveUserByUserName(
		ctx context.Context,
		username string,
	) (*model.AppUserModel, error)
	RetrieveUserList(
		ctx context.Context,
		securityContext int,
		idList []int,
		updateTs float64,
	) ([]*model.AppUserModel, error)
	UpdateUser(
		ctx context.Context,
		user *model.AppUserModel,
		clientUpdateTS float64,
		securityContext int,
	) error
	DeleteUser(
		ctx context.Context,
		userID int,
		securityContext int,
		updateTS float64,
		now float64,
		updateUserID int,
	) (int, error)
	UpdateUserPassword(
		ctx context.Context,
		userID int,
		hashedPassword string,
		now float64,
		updateUserID int,
		securityContext int,
	) error
	UpdateUserTOTPSeed(
		ctx context.Context,
		userID int,
		totpSeed string,
		updateTS float64,
		updateUserID int,
		securityContext int,
	) (float64, error)
	UpdateUserMFASetup(
		ctx context.Context,
		userID int,
		mfaSetup bool,
		expectedUpdateTS float64,
		updateUserID int,
		securityContext int,
	) (float64, error)
	UpdateUserMFAExempt(
		ctx context.Context,
		userID int,
		mfaExempt bool,
		updateTS float64,
		updateUserID int,
		securityContext int,
	) (float64, error)
	AddUserListToGroup(
		ctx context.Context,
		userIDs []int,
		groupID int,
		updateTsList []float64,
		updateUserID int,
		securityContext int,
	) ([]*model.AppUserModel, error)
	RemoveUserListFromGroup(
		ctx context.Context,
		userIDs []int,
		groupID int,
		updateTsList []float64,
		updateUserID int,
		securityContext int,
	) ([]*model.AppUserModel, error)
	GetGrantsForGroups(
		ctx context.Context,
		groupIDs []int,
	) (map[int][]int, error)
}

// AppUserGroupRepositoryInterface defines the methods needed by UserHandler
type AppUserGroupRepositoryInterface interface {
	RetrieveAppUserGroupListByIDList(
		ctx context.Context,
		groupIDs []int,
		securityContext int,
	) ([]*model.AppUserGroupModel, error)
	UpdateAppUserGroup(
		ctx context.Context,
		group *model.AppUserGroupModel,
		clientUpdateTS float64,
		securityContext int,
	) error
	CreateAppUserGroup(
		ctx context.Context,
		group *model.AppUserGroupModel,
		securityContext int,
	) (*model.AppUserGroupModel, error)
}

// UserHandler handles user management operations
type UserHandler struct {
	appsrv           *app.Services
	userRepo         UserRepositoryInterface
	appUserGroupRepo AppUserGroupRepositoryInterface
}

// NewUserHandler creates a new user handler
func NewUserHandler(
	appServices *app.Services,
	userRepo UserRepositoryInterface,
	appUserGroupRepo AppUserGroupRepositoryInterface,
) *UserHandler {
	return &UserHandler{
		appsrv:           appServices,
		userRepo:         userRepo,
		appUserGroupRepo: appUserGroupRepo,
	}
}

// validateGroupsSecurityContext ensures all groups belong to the expected security context
func (h *UserHandler) validateGroupsSecurityContext(
	groupIDs []int,
	expectedSecurityContext int,
	ctx context.Context,
) (bool, error) {
	if len(groupIDs) == 0 {
		return true, nil
	}

	groups, err := h.appUserGroupRepo.RetrieveAppUserGroupListByIDList(
		ctx,
		groupIDs,
		expectedSecurityContext,
	)
	if err != nil {
		return false, fmt.Errorf(
			"failed to retrieve groups for validation: %w",
			err,
		)
	}

	if len(groups) != len(groupIDs) {
		return false, fmt.Errorf(
			"some groups not found: requested %d, found %d",
			len(groupIDs),
			len(groups),
		)
	}

	for _, group := range groups {
		if group.SecurityContext != expectedSecurityContext {
			return false, fmt.Errorf(
				"group %d has invalid security context: expected %d, got %d",
				group.AppUserGroupID,
				expectedSecurityContext,
				group.SecurityContext,
			)
		}
	}
	return true, nil
}

// validateUsersSecurityContext ensures all users belong to the expected security context
func (h *UserHandler) validateUsersSecurityContext(
	userIDs []int,
	expectedSecurityContext int,
	ctx context.Context,
) (bool, error) {
	if len(userIDs) == 0 {
		return true, nil
	}

	users, err := h.userRepo.RetrieveUserList(
		ctx,
		expectedSecurityContext,
		userIDs,
		0,
	)
	if err != nil {
		return false, fmt.Errorf(
			"failed to retrieve users for validation: %w",
			err,
		)
	}

	if len(users) != len(userIDs) {
		return false, fmt.Errorf(
			"some users not found or inaccessible: requested %d, found %d",
			len(userIDs),
			len(users),
		)
	}

	for _, user := range users {
		if user.SecurityContext != expectedSecurityContext {
			return false, fmt.Errorf(
				"user %d has invalid security context: expected %d, got %d",
				user.AppUserID,
				expectedSecurityContext,
				user.SecurityContext,
			)
		}
	}
	return true, nil
}

// RetrieveUserList handles GET /userList
func (h *UserHandler) RetrieveUserList(w http.ResponseWriter, r *http.Request) {
	// Security checks
	app := security.NewAppHelper(w, r, h.appsrv, "RetrieveUserList")
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserRead)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("USR_LIST_START", "ReadUserList invoked")

	jwtClaims := app.GetClaims()

	// Parse optional appUserIdList query parameter
	idList, err := GetCommaSeparatedIntListFromQuery(r, "appUserIdList")
	if err != nil {
		app.WriteError(w, http.StatusBadRequest, err.Error(), "PRQ0", nil)
		return
	}
	app.Info(
		"USR_LIST_PARAMS",
		fmt.Sprintf(
			"ReadUserList with appUserIdList='%s', idCount=%d",
			r.URL.Query().Get("appUserIdList"),
			len(idList),
		),
	)

	// Parse optional afterUpdateTs query parameter
	afterUpdateTs, err := GetFloatFromQuery(r, "afterUpdateTs")
	if err != nil {
		afterUpdateTs = 0 // Default to 0 if not provided
	}

	// Read users (filtered by security context, optional idList, and afterUpdateTs)
	users, err := h.userRepo.RetrieveUserList(
		r.Context(),
		jwtClaims.SecurityContext,
		idList,
		afterUpdateTs,
	)
	if err != nil {
		app.WriteError(
			w,
			http.StatusInternalServerError,
			fmt.Sprintf(
				"Failed to retrieve users: idList=%v, afterUpdateTs=%v",
				idList,
				afterUpdateTs,
			),
			"ULF0",
			err,
		)
		return
	}

	app.Debug(
		"USR_LIST_READ",
		fmt.Sprintf("Users read: idList=%v, count=%d", idList, len(users)),
	)

	// Ensure users is never nil
	if users == nil {
		users = []*model.AppUserModel{}
	}

	// Return as JSON
	wrapper := model.DataWrapper{Data: users}
	app.WriteJSON(w, http.StatusOK, wrapper)

	app.Info("USR_LIST_DONE", "User list read completed")
}

// RetrieveGrantList handles GET /grantList
func (h *UserHandler) RetrieveGrantList(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Security checks
	app := security.NewAppHelper(w, r, h.appsrv, "RetrieveGrantList")
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserCreate)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("GRT_LIST_START", "ReadGrantList invoked")

	// Convert GrantIDToShortCodeMap to array of objects
	grantList := make(
		[]map[string]interface{},
		0,
		len(model.GrantIDToShortCodeMap),
	)
	for id, text := range model.GrantIDToShortCodeMap {
		grantList = append(grantList, map[string]interface{}{
			"id":   id,
			"text": text,
		})
	}

	app.Debug("GRT_LST_READ", "Grant list read")
	wrapper := model.DataWrapper{Data: grantList}
	app.WriteJSON(w, http.StatusOK, wrapper)
	app.Info("GRT_LIST_DONE", "Grant list read completed")
}

// CreateUser handles POST /user
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Security checks
	app := security.NewAppHelper(w, r, h.appsrv, "CreateUser")
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserCreate)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("USR_CREATE_START", "CreateUser invoked")

	jwtClaims := app.GetClaims()

	// Parse request body into CreateUserRequest
	var req model.CreateUserRequest
	if !decodeJSONBody(app, w, r, maxLoginLayerBodyBytes, &req) {
		return
	}

	app.Debug("UCP0", "User creation request parsed")

	// Validate required fields
	if req.UserName == "" || req.FirstName == "" || req.LastName == "" ||
		req.Email == "" {
		app.WriteError(
			w,
			http.StatusBadRequest,
			fmt.Sprintf(
				"Required fields missing for create user: userName='%s', firstName='%s', lastName='%s', email='%s'",
				req.UserName,
				req.FirstName,
				req.LastName,
				req.Email,
			),
			"BRQ0",
			nil,
		)
		return
	}

	// Validate group memberships are in the same security context
	if valid, err := h.validateGroupsSecurityContext(req.AppUserGroupMembershipList, jwtClaims.SecurityContext, r.Context()); !valid ||
		err != nil {
		app.WriteError(
			w,
			http.StatusForbidden,
			"Access denied: all groups must be in the same security context as the user",
			"SEC_CTX_VIOLATION",
			nil,
		)
		return
	}

	// Validate password length
	if len(req.PwRaw) < 8 {
		app.WriteError(
			w,
			http.StatusBadRequest,
			fmt.Sprintf(
				"Password must be at least 8 characters long: len=%d",
				len(req.PwRaw),
			),
			"BRQ0",
			nil,
		)
		return
	}

	// Create AppUserModel from request
	now := CurrentTimestamp()
	user := model.AppUserModel{
		UserName:                   req.UserName,
		FirstName:                  req.FirstName,
		LastName:                   req.LastName,
		Email:                      req.Email,
		AppUserGroupMembershipList: req.AppUserGroupMembershipList,
		MFAExempt:                  req.MFAExempt,
		SecurityContext:            jwtClaims.SecurityContext,
		CreateTS:                   now,
		CreateUserID:               jwtClaims.UserID,
		UpdateTS:                   now,
		UpdateUserID:               jwtClaims.UserID,
	}

	// Hash password
	hashed, err := auth.HashPassword(req.PwRaw)
	if err != nil {
		app.WriteError(
			w,
			http.StatusInternalServerError,
			fmt.Sprintf(
				"Failed to process password for user: username=%s",
				req.UserName,
			),
			"UCH0",
			err,
		)
		return
	}
	user.PwHash = hashed

	// Create user
	createdUser, err := h.userRepo.CreateUser(
		r.Context(),
		&user,
		jwtClaims.SecurityContext,
	)
	if err != nil {
		// Check if it's a validation error
		if strings.Contains(err.Error(), "username must contain only") {
			app.WriteError(
				w,
				http.StatusBadRequest,
				fmt.Sprintf("Invalid username: %s", err.Error()),
				"BRQ0",
				nil,
			)
			return
		}
		if errors.Is(err, repository.ErrDuplicate) {
			app.WriteError(
				w,
				http.StatusConflict,
				fmt.Sprintf("Username already exists: username=%s", req.UserName),
				"DUPLICATE_ERROR",
				err,
			)
			return
		}
		app.WriteError(
			w,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to create user: username=%s", req.UserName),
			"UCF0",
			err,
		)
		return
	}

	app.Debug("UCD0", "User created")
	wrapper := model.DataWrapper{Data: createdUser}
	writeJSON(w, http.StatusCreated, wrapper, h.appsrv.Logger)
	app.Info(
		"USR_CREATE_DONE",
		fmt.Sprintf(
			"User creation completed for userId=%d, username='%s'",
			createdUser.AppUserID,
			createdUser.UserName,
		),
	)
}

// UpdateUser handles PUT /user
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	// Security checks
	app := security.NewAppHelper(w, r, h.appsrv, "UpdateUser")
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserUpdate)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("USR_UPDATE_START", "UpdateUser invoked")

	jwtClaims := app.GetClaims()

	// Parse request body
	var user model.AppUserModel
	if !decodeJSONBody(app, w, r, maxLoginLayerBodyBytes, &user) {
		return
	}

	app.Debug("UUP0", "User update request parsed")

	// Validate required fields
	if user.AppUserID == 0 || user.UserName == "" {
		app.WriteError(
			w,
			http.StatusBadRequest,
			fmt.Sprintf(
				"appUserId and userName are required for update: appUserId=%d, userName='%s'",
				user.AppUserID,
				user.UserName,
			),
			"BRQ0",
			nil,
		)
		return
	}

	// Validate group memberships are in the same security context
	if valid, err := h.validateGroupsSecurityContext(user.AppUserGroupMembershipList, jwtClaims.SecurityContext, r.Context()); !valid ||
		err != nil {
		app.WriteError(
			w,
			http.StatusForbidden,
			"Access denied: all groups must be in the same security context as the user",
			"SEC_CTX_VIOLATION",
			nil,
		)
		return
	}

	app.Info(
		"USR_UPDATE_PARAMS",
		fmt.Sprintf(
			"UpdateUser for userId=%d, username='%s'",
			user.AppUserID,
			user.UserName,
		),
	)

	// Set update metadata
	clientUpdateTS := user.UpdateTS
	now := CurrentTimestamp()
	user.SecurityContext = jwtClaims.SecurityContext
	user.UpdateTS = now
	user.UpdateUserID = jwtClaims.UserID

	// The name the login's tokens carry, before an update that may rename it.
	priorUserName, err := userNameOf(
		r, h.userRepo, jwtClaims.SecurityContext, user.AppUserID,
	)
	if err != nil {
		app.WriteError(
			w,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to read user: app_user_id=%d", user.AppUserID),
			"USR_UPD_FAIL",
			err,
		)
		return
	}

	// Update user with optimistic locking
	err = h.userRepo.UpdateUser(
		r.Context(),
		&user,
		clientUpdateTS,
		jwtClaims.SecurityContext,
	)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConflict):
			app.WriteError(
				w,
				http.StatusConflict,
				fmt.Sprintf(
					"User was modified by another user: app_user_id=%d",
					user.AppUserID,
				),
				"CONFLICT_ERROR",
				err,
			)
		case errors.Is(err, repository.ErrNotFound):
			app.WriteError(
				w,
				http.StatusNotFound,
				fmt.Sprintf("User not found: app_user_id=%d", user.AppUserID),
				"NOT_FOUND_ERROR",
				err,
			)
		case errors.Is(err, repository.ErrDuplicate):
			app.WriteError(
				w,
				http.StatusConflict,
				fmt.Sprintf("Username already exists: username=%s", user.UserName),
				"DUPLICATE_ERROR",
				err,
			)
		default:
			app.WriteError(
				w,
				http.StatusInternalServerError,
				fmt.Sprintf(
					"Failed to update user: app_user_id=%d",
					user.AppUserID,
				),
				"USR_UPD_FAIL",
				err,
			)
		}
		return
	}

	app.Debug("UUD0", "User updated")
	// Name, groups and MFA exemption are in the login's tokens.
	if !revokeUserSessionList(
		app, w, h.appsrv, priorUserName, user.UserName,
	) {
		return
	}
	wrapper := model.DataWrapper{Data: user}
	app.WriteJSON(w, http.StatusOK, wrapper)
	app.Info(
		"USR_UPDATE_DONE",
		fmt.Sprintf(
			"User update completed for userId=%d, username='%s'",
			user.AppUserID,
			user.UserName,
		),
	)
}

// DeleteUser handles DELETE /user
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Security checks
	app := security.NewAppHelper(w, r, h.appsrv, "DeleteUser")
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserDelete)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("USR_DELETE_START", "DeleteUser invoked")

	jwtClaims := app.GetClaims()

	// Parse query parameters
	userID, err := GetIntFromQuery(r, "appUserId")
	if err != nil {
		app.WriteError(w, http.StatusBadRequest, err.Error(), "PRQ0", nil)
		return
	}

	app.Info(
		"USR_DELETE_PARAMS",
		fmt.Sprintf("DeleteUser for userId=%d", userID),
	)

	updateTS, err := GetFloatFromQuery(r, "updateTs")
	if err != nil {
		app.WriteError(w, http.StatusBadRequest, err.Error(), "PRQ0", nil)
		return
	}

	deletedUserName, err := userNameOf(
		r, h.userRepo, jwtClaims.SecurityContext, userID,
	)
	if err != nil {
		app.WriteError(
			w,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to read user: app_user_id=%d", userID),
			"UDF0",
			err,
		)
		return
	}

	// Delete user with optimistic locking
	now := CurrentTimestamp()
	deletedID, err := h.userRepo.DeleteUser(
		r.Context(),
		userID,
		jwtClaims.SecurityContext,
		updateTS,
		now,
		jwtClaims.UserID,
	)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConflict):
			app.WriteError(
				w,
				http.StatusConflict,
				fmt.Sprintf(
					"User was modified by another user: app_user_id=%d",
					userID,
				),
				"CNF0",
				nil,
			)
		case errors.Is(err, repository.ErrNotFound):
			app.WriteError(
				w,
				http.StatusNotFound,
				fmt.Sprintf("User not found: app_user_id=%d", userID),
				"NFN0",
				nil,
			)
		default:
			app.WriteError(
				w,
				http.StatusInternalServerError,
				fmt.Sprintf("Failed to delete user: app_user_id=%d", userID),
				"UDF0",
				err,
			)
		}
		return
	}

	if !revokeUserSessionList(app, w, h.appsrv, deletedUserName) {
		return
	}
	data := map[string]interface{}{
		"message":   "User deleted successfully",
		"appUserId": deletedID,
	}
	wrapper := model.DataWrapper{Data: data}
	app.WriteJSON(w, http.StatusOK, wrapper)
	app.Info(
		"USR_DELETE_DONE",
		fmt.Sprintf("User deletion completed for userId=%d", deletedID),
	)
}

// AddUserListToGroupHandler handles POST /addUserListToGroup
func (h *UserHandler) AddUserListToGroupHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Security checks
	app := security.NewAppHelper(w, r, h.appsrv, "AddUserListToGroupHandler")
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserGroupUpdate)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("ADD_USR_GRP_S", "AddUserListToGroup invoked")

	jwtClaims := app.GetClaims()

	// Parse request
	var req struct {
		UserIds      []int     `json:"userIds"`
		GroupId      int       `json:"groupId"`
		UpdateTsList []float64 `json:"updateTsList"`
	}
	if !decodeJSONBody(app, w, r, maxLoginLayerBodyBytes, &req) {
		return
	}

	if len(req.UserIds) == 0 || len(req.UpdateTsList) != len(req.UserIds) {
		app.WriteError(
			w,
			http.StatusBadRequest,
			fmt.Sprintf(
				"userIds and updateTsList must be provided and have matching lengths: userIds=%d, updateTsList=%d",
				len(req.UserIds),
				len(req.UpdateTsList),
			),
			"BRQ0",
			nil,
		)
		return
	}

	// Validate group is in the same security context
	if valid, err := h.validateGroupsSecurityContext([]int{req.GroupId}, jwtClaims.SecurityContext, r.Context()); !valid ||
		err != nil {
		app.WriteError(
			w,
			http.StatusForbidden,
			"Access denied: group must be in the same security context",
			"SEC_CTX_VIOLATION",
			nil,
		)
		return
	}

	// Validate all users are in the same security context
	if valid, err := h.validateUsersSecurityContext(req.UserIds, jwtClaims.SecurityContext, r.Context()); !valid ||
		err != nil {
		app.WriteError(
			w,
			http.StatusForbidden,
			"Access denied: all users must be in the same security context",
			"SEC_CTX_VIOLATION",
			nil,
		)
		return
	}

	// Add users to group
	updatedUsers, err := h.userRepo.AddUserListToGroup(
		r.Context(),
		req.UserIds,
		req.GroupId,
		req.UpdateTsList,
		jwtClaims.UserID,
		jwtClaims.SecurityContext,
	)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			app.WriteError(
				w,
				http.StatusConflict,
				fmt.Sprintf(
					"Concurrent update detected for add to group: group_id=%d, app_user_ids=%v",
					req.GroupId,
					req.UserIds,
				),
				"CONFLICT_ERROR",
				err,
			)
			return
		}
		app.WriteError(
			w,
			http.StatusInternalServerError,
			fmt.Sprintf(
				"Failed to add users to group: group_id=%d, user_count=%d",
				req.GroupId,
				len(req.UserIds),
			),
			"ADD_GRP_FAIL",
			err,
		)
		return
	}

	app.Info("ADD_USR_GRP_D", "Users added to group successfully")
	// Group membership is in every token's grants.
	if !revokeUserSessionList(
		app, w, h.appsrv, userNameListOf(updatedUsers)...,
	) {
		return
	}
	wrapper := model.DataWrapper{Data: updatedUsers}
	app.WriteJSON(w, http.StatusOK, wrapper)
}

// RemoveUserListFromGroupHandler handles POST /removeUserListFromGroup
func (h *UserHandler) RemoveUserListFromGroupHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Security checks
	app := security.NewAppHelper(w, r, h.appsrv, "RemoveUserListFromGroupHandler")
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserGroupUpdate)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("REM_USR_GRP_S", "RemoveUserListFromGroup invoked")

	jwtClaims := app.GetClaims()

	// Parse request
	var req struct {
		UserIds      []int     `json:"userIds"`
		GroupId      int       `json:"groupId"`
		UpdateTsList []float64 `json:"updateTsList"`
	}
	if !decodeJSONBody(app, w, r, maxLoginLayerBodyBytes, &req) {
		return
	}

	if len(req.UserIds) == 0 || len(req.UpdateTsList) != len(req.UserIds) {
		app.WriteError(
			w,
			http.StatusBadRequest,
			"userIds and updateTsList must be provided and have matching lengths",
			"BRQ0",
			nil,
		)
		return
	}

	// Validate group is in the same security context
	if valid, err := h.validateGroupsSecurityContext([]int{req.GroupId}, jwtClaims.SecurityContext, r.Context()); !valid ||
		err != nil {
		app.WriteError(
			w,
			http.StatusForbidden,
			"Access denied: group must be in the same security context",
			"SEC_CTX_VIOLATION",
			nil,
		)
		return
	}

	// Validate all users are in the same security context
	if valid, err := h.validateUsersSecurityContext(req.UserIds, jwtClaims.SecurityContext, r.Context()); !valid ||
		err != nil {
		app.WriteError(
			w,
			http.StatusForbidden,
			"Access denied: all users must be in the same security context",
			"SEC_CTX_VIOLATION",
			nil,
		)
		return
	}

	// Remove users from group
	updatedUsers, err := h.userRepo.RemoveUserListFromGroup(
		r.Context(),
		req.UserIds,
		req.GroupId,
		req.UpdateTsList,
		jwtClaims.UserID,
		jwtClaims.SecurityContext,
	)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			app.WriteError(
				w,
				http.StatusConflict,
				"Concurrent update detected",
				"CONFLICT_ERROR",
				err,
			)
			return
		}
		app.WriteError(
			w,
			http.StatusInternalServerError,
			"Failed to remove users from group",
			"REM_GRP_FAIL",
			err,
		)
		return
	}

	app.Info("REM_USR_GRP_D", "Users removed from group successfully")
	// Group membership is in every token's grants.
	if !revokeUserSessionList(
		app, w, h.appsrv, userNameListOf(updatedUsers)...,
	) {
		return
	}
	wrapper := model.DataWrapper{Data: updatedUsers}
	app.WriteJSON(w, http.StatusOK, wrapper)
}

// ChangeOtherUserPassword handles POST /changeOtherUserPassword
func (h *UserHandler) ChangeOtherUserPassword(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Security checks
	app := security.NewAppHelper(w, r, h.appsrv, "ChangeOtherUserPassword")
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantChangeOtherUserPassword)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("CPWS", "Change user password invoked")

	jwtClaims := app.GetClaims()

	// Parse request body
	var req struct {
		UserID      int    `json:"appUserId"`
		NewPassword string `json:"newPassword"`
	}
	if !decodeJSONBody(app, w, r, maxLoginLayerBodyBytes, &req) {
		return
	}

	// Validate required fields
	if req.UserID == 0 || req.NewPassword == "" {
		app.WriteError(
			w,
			http.StatusBadRequest,
			"appUserId and newPassword are required",
			"BRQ0",
			nil,
		)
		return
	}

	// Basic password validation (minimum length)
	if len(req.NewPassword) < 8 {
		app.WriteError(
			w,
			http.StatusBadRequest,
			"New password must be at least 8 characters long",
			"BRQ0",
			nil,
		)
		return
	}

	// Verify the user exists and get their details for logging
	users, err := h.userRepo.RetrieveUserList(
		r.Context(),
		jwtClaims.SecurityContext,
		[]int{req.UserID},
		0,
	)
	if err != nil {
		app.WriteError(
			w,
			http.StatusInternalServerError,
			"Failed to retrieve user",
			"USF0",
			err,
		)
		return
	}
	if len(users) == 0 {
		app.WriteError(w, http.StatusNotFound, "User not found", "NFN0", nil)
		return
	}
	user := users[0]

	// Hash new password
	hashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		app.WriteError(
			w,
			http.StatusInternalServerError,
			"Failed to process password",
			"UCH0",
			err,
		)
		return
	}

	// Update password
	now := CurrentTimestamp()
	err = h.userRepo.UpdateUserPassword(
		r.Context(),
		req.UserID,
		hashedPassword,
		now,
		jwtClaims.UserID,
		user.SecurityContext,
	)
	if err != nil {
		app.WriteError(
			w,
			http.StatusInternalServerError,
			"Failed to update password",
			"UUF0",
			err,
		)
		return
	}

	app.Debug("PW_CHANGED", "Password updated for user: "+user.UserName)
	if !revokeUserSessionList(app, w, h.appsrv, user.UserName) {
		return
	}

	wrapper := model.DataWrapper{
		Data: map[string]string{"message": "Password changed successfully"},
	}
	app.WriteJSON(w, http.StatusOK, wrapper)
	app.Info("PW_CHANGED", "User password change completed")
}
