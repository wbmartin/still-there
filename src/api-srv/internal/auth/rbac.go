package auth

import (
	"fmt"

	"stillthere-api/internal/logging"
	"stillthere-api/internal/model"
)

// RBACManager handles role-based access control operations
type RBACManager struct {
	logger   *logging.Logger
	username string
}

// NewRBACManager creates a new RBAC manager
func NewRBACManager(logger *logging.Logger,
	username string) *RBACManager {
	return &RBACManager{
		logger:   logger,
		username: username,
	}
}

// HasGrant checks if a user has a specific grant in their
// current security context
func (r *RBACManager) HasGrant(claims *Claims, grantID int,
	securityContext int) bool {
	if claims == nil || claims.Grants == nil {
		r.logger.Debug("NCL0", "No claims or grants available",
			"username", r.username)
		return false
	}

	// Since grants are now context-specific, check if the security context matches
	if claims.SecurityContext != securityContext {
		r.logger.Debug(
			"SCT0",
			fmt.Sprintf(
				"Security context mismatch - requested: %d, user: %d",
				securityContext,
				claims.SecurityContext,
			),
			"username",
			r.username,
		)
		return false
	}

	for _, userGrantID := range claims.Grants {
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

	// Check if the security context matches and has grants
	return claims.SecurityContext == securityContext && len(claims.Grants) > 0
}

// ValidateGrantID checks if a grant ID is valid
func (r *RBACManager) ValidateGrantID(grantID int) bool {
	_, exists := model.GrantIDToShortCodeMap[grantID]
	return exists
}
