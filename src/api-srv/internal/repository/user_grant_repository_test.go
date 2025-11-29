package repository

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	"github.com/wbmartin/starter-api/internal/app_lib"
	"github.com/wbmartin/starter-api/internal/model"
	"testing"
)

func TestUserGrantRepository(t *testing.T) {
	suite.Run(t, &UserGrantRepositoryTestSuite{})
}

type UserGrantRepositoryTestSuite struct {
	suite.Suite
	db      *pgxpool.Pool
	repo    *UserGrantRepository
	cleanup func()
}

func (s *UserGrantRepositoryTestSuite) SetupSuite() {
	// Load configuration from the same file used by the application
	config, err := app_lib.LoadConfig("../../../../etc/loc-dev-app-srv-config.env")
	s.Require().NoError(err)

	// Connect to the database using the configured URL
	db, err := pgxpool.New(context.Background(), config.DatabaseURL)
	s.Require().NoError(err)
	s.db = db
	s.repo = NewUserGrantRepository(db)

	s.cleanup = func() {
		// Clean only test data if any was created
		// User grants are typically static data from initial migration
	}
}

func (s *UserGrantRepositoryTestSuite) TearDownSuite() {
	s.cleanup()
	s.db.Close()
}

func (s *UserGrantRepositoryTestSuite) TestNewUserGrantRepository() {
	// Test constructor
	repo := NewUserGrantRepository(s.db)
	s.NotNil(repo)
	s.Equal(s.db, repo.db)
}

func (s *UserGrantRepositoryTestSuite) TestGetUserGrants() {
	// First, check what grants actually exist in the database
	// This makes the test work regardless of whether initial data was loaded
	allGrants, err := s.db.Query(context.Background(), "SELECT user_grant_id FROM user_grant WHERE delete_user_id IS NULL ORDER BY user_grant_id")
	s.NoError(err)
	defer allGrants.Close()

	var existingGrantIDs []int
	for allGrants.Next() {
		var id int
		err := allGrants.Scan(&id)
		s.NoError(err)
		existingGrantIDs = append(existingGrantIDs, id)
	}

	if len(existingGrantIDs) == 0 {
		// No grants in database, skip this test
		s.T().Skip("No user grants found in database - skipping test")
		return
	}

	// Test retrieving existing grants
	grants, err := s.repo.GetUserGrants(context.Background(), existingGrantIDs)
	s.NoError(err)
	s.NotNil(grants)
	s.Len(grants, len(existingGrantIDs))

	// Validate each returned grant
	for _, grant := range grants {
		s.NotZero(grant.UserGrantID)
		s.NotEmpty(grant.GrantShortCode)
		s.NotEmpty(grant.GrantDescription)
		s.NotZero(grant.CreateTS)
		s.NotZero(grant.CreateUserID)
		s.NotZero(grant.UpdateTS)
		s.NotZero(grant.UpdateUserID)
		s.Nil(grant.DeleteTS) // Should not be deleted
		s.Nil(grant.DeleteUserID)
	}
}

func (s *UserGrantRepositoryTestSuite) TestGetUserGrantsEmpty() {
	// Test with empty slice
	grants, err := s.repo.GetUserGrants(context.Background(), []int{})
	s.NoError(err)
	s.Equal(0, len(grants))
	s.Equal([]*model.UserGrantModel{}, grants)
}

func (s *UserGrantRepositoryTestSuite) TestGetUserGrantsNonExistent() {
	// Test with IDs that don't exist
	grantIDs := []int{99999, 88888} // Non-existent grant IDs

	grants, err := s.repo.GetUserGrants(context.Background(), grantIDs)
	s.NoError(err)
	s.Equal(0, len(grants)) // Should return empty slice for non-existent grants
}

func (s *UserGrantRepositoryTestSuite) TestGetUserGrantsPartial() {
	// First, check what grants actually exist
	allGrants, err := s.db.Query(context.Background(), "SELECT user_grant_id, grant_short_code FROM user_grant WHERE delete_user_id IS NULL ORDER BY user_grant_id LIMIT 1")
	s.NoError(err)
	defer allGrants.Close()

	var existingID int
	var existingCode string
	if allGrants.Next() {
		err := allGrants.Scan(&existingID, &existingCode)
		s.NoError(err)
	} else {
		// No grants in database, skip this test
		s.T().Skip("No user grants found in database - skipping partial test")
		return
	}

	// Test with mix of existing and non-existing grants
	grantIDs := []int{existingID, 99999} // One existing, one non-existing

	grants, err := s.repo.GetUserGrants(context.Background(), grantIDs)
	s.NoError(err)

	// Should find only the existing grant
	s.Len(grants, 1)
	s.Equal(existingID, grants[0].UserGrantID)
	s.Equal(existingCode, grants[0].GrantShortCode)
}
