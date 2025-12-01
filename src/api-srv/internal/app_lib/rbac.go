package app_lib

import (
	"fmt"

	"github.com/wbmartin/starter-api/internal/model"
)

// RBACManager handles role-based access control operations
type RBACManager struct {
	logger ThisAppLoggerInterface
}

// NewRBACManager creates a new RBAC manager
func NewRBACManager(logger ThisAppLoggerInterface) *RBACManager {
	return &RBACManager{
		logger: logger,
	}
}

// HasGrant checks if a user has a specific grant in a security context
func (r *RBACManager) HasGrant(claims *Claims, grantID int, securityContext int) bool {
	if claims == nil || claims.Grants == nil {
		r.logger.Debug("NCL0", "No claims or grants available")
		return false
	}

	contextGrants, exists := claims.Grants[securityContext]
	if !exists {
		r.logger.Debug("NCT0", fmt.Sprintf("No grants found for security context: %d", securityContext))
		return false
	}

	for _, userGrantID := range contextGrants {
		if userGrantID == grantID {
			r.logger.Debug("GFD0", fmt.Sprintf("Grant found - ID: %d, context: %d", userGrantID, securityContext))
			return true
		}
	}

	r.logger.Debug("GNF0", fmt.Sprintf("Grant not found - ID: %d, context: %d", grantID, securityContext))
	return false
}

// HasAnyGrantInContext checks if a user has any grants in a specific security context
func (r *RBACManager) HasAnyGrantInContext(claims *Claims, securityContext int) bool {
	if claims == nil || claims.Grants == nil {
		return false
	}

	contextGrants, exists := claims.Grants[securityContext]
	return exists && len(contextGrants) > 0
}

// GetUserGrantsByContext retrieves all grant short codes for a user by security context
func (r *RBACManager) GetUserGrantsByContext(claims *Claims) map[int][]string {
	result := make(map[int][]string)

	if claims == nil || claims.Grants == nil {
		return result
	}

	for securityContext, grantIDs := range claims.Grants {
		var shortCodesList []string
		for _, grantID := range grantIDs {
			if shortCode, exists := model.GrantIDToShortCodeMap[grantID]; exists {
				shortCodesList = append(shortCodesList, shortCode)
			} else {
				r.logger.Warn("UGT0", fmt.Sprintf("Unknown grant ID encountered: %d", grantID))
			}
		}
		result[securityContext] = shortCodesList
	}

	return result
}

// ValidateGrantID checks if a grant ID is valid
func (r *RBACManager) ValidateGrantID(grantID int) bool {
	_, exists := model.GrantIDToShortCodeMap[grantID]
	return exists
}

// GetGrantShortCode returns the short code for a grant ID
func (r *RBACManager) GetGrantShortCode(grantID int) (string, error) {
	shortCode, exists := model.GrantIDToShortCodeMap[grantID]
	if !exists {
		return "", fmt.Errorf("unknown grant ID: %d", grantID)
	}
	return shortCode, nil
}
