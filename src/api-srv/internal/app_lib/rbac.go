package app_lib

import (
	"fmt"

	"github.com/wbmartin/starter-api/internal/model"
)

// RBACManager handles role-based access control operations
type RBACManager struct {
	logger   ThisAppLoggerInterface
	username string
}

// NewRBACManager creates a new RBAC manager
func NewRBACManager(logger ThisAppLoggerInterface,
	username string) *RBACManager {
	return &RBACManager{
		logger:   logger,
		username: username,
	}
}

// HasGrant checks if a user has a specific grant in a security context
func (r *RBACManager) HasGrant(claims *Claims, grantID int,
	securityContext int) bool {
	if claims == nil || claims.Grants == nil {
		r.logger.Debug("NCL0", "No claims or grants available",
			"username", r.username)
		return false
	}

	contextGrants, exists := claims.Grants[securityContext]
	if !exists {
		r.logger.Debug("NCT0",
			fmt.Sprintf("No grants found for security context: %d", securityContext),
			"username", r.username)
		return false
	}

	for _, userGrantID := range contextGrants {
		if userGrantID == grantID {
			r.logger.Debug("RFD0",
				fmt.Sprintf("Grant found - ID: %d, context: %d",
					userGrantID, securityContext),
				"username", r.username)
			return true
		}
	}

	r.logger.Debug("RNF0",
		fmt.Sprintf("Grant not found - ID: %d, context: %d",
			grantID, securityContext),
		"username", r.username)
	return false
}

// HasAnyGrantInContext checks if a user has any grants
// in a specific security context
// WBM 2025.12.11 I don't think is used anywhere, but leaving it for now.
func (r *RBACManager) HasAnyGrantInContext(claims *Claims,
	securityContext int) bool {
	if claims == nil || claims.Grants == nil {
		return false
	}

	contextGrants, exists := claims.Grants[securityContext]
	return exists && len(contextGrants) > 0
}

// RetrieveUserGrantsByContext retrieves all grant short codes for a user
// by security context
func (r *RBACManager) RetrieveUserGrantsByContext(claims *Claims) map[int][]string {
	result := make(map[int][]string)

	if claims == nil || claims.Grants == nil {
		return result
	}

	for securityContext, grantIDs := range claims.Grants {
		var shortCodeList []string
		for _, grantID := range grantIDs {
			if shortCode, exists := model.GrantIDToShortCodeMap[grantID]; exists {
				shortCodeList = append(shortCodeList, shortCode)
			} else {
				r.logger.Warn("UGT0",
					fmt.Sprintf("Unknown grant ID encountered: %d", grantID),
					"username", r.username)
			}
		}
		result[securityContext] = shortCodeList
	}

	return result
}

// ValidateGrantID checks if a grant ID is valid
func (r *RBACManager) ValidateGrantID(grantID int) bool {
	_, exists := model.GrantIDToShortCodeMap[grantID]
	return exists
}

// RetrieveGrantShortCode returns the short code for a grant ID
func (r *RBACManager) RetrieveGrantShortCode(grantID int) (string, error) {
	shortCode, exists := model.GrantIDToShortCodeMap[grantID]
	if !exists {
		return "", fmt.Errorf("unknown grant ID: %d", grantID)
	}
	return shortCode, nil
}
