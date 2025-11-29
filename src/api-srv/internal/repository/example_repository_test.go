package repository

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	"github.com/wbmartin/starter-api/internal/app_lib"
	"github.com/wbmartin/starter-api/internal/model"
	"testing"
)

func TestExampleRepository(t *testing.T) {
	suite.Run(t, &ExampleRepositoryTestSuite{})
}

type ExampleRepositoryTestSuite struct {
	suite.Suite
	db      *pgxpool.Pool
	repo    *ExampleRepository
	cleanup func()
}

func (s *ExampleRepositoryTestSuite) SetupSuite() {
	// Load configuration from the same file used by the application
	config, err := app_lib.LoadConfig("../../../../etc/api-srv-loc.env")
	s.Require().NoError(err)

	// Connect to the database using the configured URL
	db, err := pgxpool.New(context.Background(), config.DatabaseURL)
	s.Require().NoError(err)
	s.db = db
	s.repo = NewExampleRepository(db)

	s.cleanup = func() {
		// Clean only test data
		db.Exec(context.Background(),
			"DELETE FROM example WHERE example_text LIKE 'Test%' OR example_text LIKE 'RepoTest%'")
	}
}

func (s *ExampleRepositoryTestSuite) TearDownSuite() {
	s.cleanup()
	s.db.Close()
}

func (s *ExampleRepositoryTestSuite) TestCreateExample() {
	example := &model.ExampleModel{
		ExampleText:     "RepoTest Example",
		SecurityContext: 1,
		CreateTS:        1234567890.123,
		CreateUserID:    1,
		UpdateTS:        1234567890.123,
		UpdateUserID:    1,
	}

	created, err := s.repo.CreateExample(context.Background(), example)
	s.NoError(err)
	s.NotNil(created)
	s.NotZero(created.ExampleID)
	s.Equal("RepoTest Example", created.ExampleText)
}

func (s *ExampleRepositoryTestSuite) TestGetExample() {
	// Create test data first
	example := &model.ExampleModel{
		ExampleText:     "RepoTest Get Example",
		SecurityContext: 1,
		CreateTS:        1234567890.123,
		CreateUserID:    1,
		UpdateTS:        1234567890.123,
		UpdateUserID:    1,
	}

	created, _ := s.repo.CreateExample(context.Background(), example)

	// Test retrieval
	retrieved, err := s.repo.GetExample(context.Background(), created.ExampleID, 1)
	s.NoError(err)
	s.Equal(created.ExampleID, retrieved.ExampleID)
	s.Equal("RepoTest Get Example", retrieved.ExampleText)
}

func (s *ExampleRepositoryTestSuite) TestUpdateExample() {
	// Create test data
	example := &model.ExampleModel{
		ExampleText:     "RepoTest Update Example",
		SecurityContext: 1,
		CreateTS:        1234567890.123,
		CreateUserID:    1,
		UpdateTS:        1234567890.123,
		UpdateUserID:    1,
	}

	created, _ := s.repo.CreateExample(context.Background(), example)

	// Update it
	created.ExampleText = "RepoTest Updated Example"
	created.UpdateTS = 1234567891.456
	created.UpdateUserID = 2

	err := s.repo.UpdateExample(context.Background(), created, 1234567890.123)
	s.NoError(err)

	// Verify update
	updated, _ := s.repo.GetExample(context.Background(), created.ExampleID, 1)
	s.Equal("RepoTest Updated Example", updated.ExampleText)
}

func (s *ExampleRepositoryTestSuite) TestDeleteExample() {
	// Create test data
	example := &model.ExampleModel{
		ExampleText:     "RepoTest Delete Example",
		SecurityContext: 1,
		CreateTS:        1234567890.123,
		CreateUserID:    1,
		UpdateTS:        1234567890.123,
		UpdateUserID:    1,
	}

	created, _ := s.repo.CreateExample(context.Background(), example)

	// Delete it
	deletedID, err := s.repo.DeleteExample(context.Background(), created.ExampleID, 1, 1234567890.123, 1234567892.789, 2)
	s.NoError(err)
	s.Equal(created.ExampleID, deletedID)

	// Verify deletion (soft delete)
	_, err = s.repo.GetExample(context.Background(), created.ExampleID, 1)
	s.Error(err) // Should not find the deleted record
}

func (s *ExampleRepositoryTestSuite) TestCreateExampleList() {
	examples := []*model.ExampleModel{
		{
			ExampleText:     "RepoTest List Example 1",
			SecurityContext: 1,
			CreateTS:        1234567890.123,
			CreateUserID:    1,
			UpdateTS:        1234567890.123,
			UpdateUserID:    1,
		},
		{
			ExampleText:     "RepoTest List Example 2",
			SecurityContext: 1,
			CreateTS:        1234567890.123,
			CreateUserID:    1,
			UpdateTS:        1234567890.123,
			UpdateUserID:    1,
		},
	}

	created, err := s.repo.CreateExampleList(context.Background(), examples)
	s.NoError(err)
	s.Len(created, 2)
	s.NotZero(created[0].ExampleID)
	s.NotZero(created[1].ExampleID)
}

func (s *ExampleRepositoryTestSuite) TestGetExampleList() {
	// Create test data
	example1 := &model.ExampleModel{
		ExampleText:     "RepoTest List Get 1",
		SecurityContext: 1,
		CreateTS:        1234567890.123,
		CreateUserID:    1,
		UpdateTS:        1234567890.123,
		UpdateUserID:    1,
	}
	example2 := &model.ExampleModel{
		ExampleText:     "RepoTest List Get 2",
		SecurityContext: 1,
		CreateTS:        1234567890.123,
		CreateUserID:    1,
		UpdateTS:        1234567890.123,
		UpdateUserID:    1,
	}

	created1, _ := s.repo.CreateExample(context.Background(), example1)
	created2, _ := s.repo.CreateExample(context.Background(), example2)

	// Test list retrieval
	ids := []int{created1.ExampleID, created2.ExampleID}
	retrieved, err := s.repo.GetExampleList(context.Background(), ids, 1)
	s.NoError(err)
	s.Len(retrieved, 2)
}

func (s *ExampleRepositoryTestSuite) TestUpdateExampleList() {
	// Create test data
	examples := []*model.ExampleModel{
		{
			ExampleText:     "RepoTest Update List 1",
			SecurityContext: 1,
			CreateTS:        1234567890.123,
			CreateUserID:    1,
			UpdateTS:        1234567890.123,
			UpdateUserID:    1,
		},
		{
			ExampleText:     "RepoTest Update List 2",
			SecurityContext: 1,
			CreateTS:        1234567890.123,
			CreateUserID:    1,
			UpdateTS:        1234567890.123,
			UpdateUserID:    1,
		},
	}

	created, _ := s.repo.CreateExampleList(context.Background(), examples)

	// Update them
	created[0].ExampleText = "RepoTest Updated List 1"
	created[1].ExampleText = "RepoTest Updated List 2"
	created[0].UpdateTS = 1234567891.456
	created[1].UpdateTS = 1234567891.456
	created[0].UpdateUserID = 2
	created[1].UpdateUserID = 2

	err := s.repo.UpdateExampleList(context.Background(), created)
	s.NoError(err)
}

func (s *ExampleRepositoryTestSuite) TestDeleteExampleList() {
	// Create test data
	examples := []*model.ExampleModel{
		{
			ExampleText:     "RepoTest Delete List 1",
			SecurityContext: 1,
			CreateTS:        1234567890.123,
			CreateUserID:    1,
			UpdateTS:        1234567890.123,
			UpdateUserID:    1,
		},
		{
			ExampleText:     "RepoTest Delete List 2",
			SecurityContext: 1,
			CreateTS:        1234567890.123,
			CreateUserID:    1,
			UpdateTS:        1234567890.123,
			UpdateUserID:    1,
		},
	}

	created, _ := s.repo.CreateExampleList(context.Background(), examples)
	ids := []int{created[0].ExampleID, created[1].ExampleID}
	updateTs := []float64{1234567890.123, 1234567890.123}

	// Delete them
	deletedIDs, err := s.repo.DeleteExampleList(context.Background(), ids, updateTs, 1, 1234567892.789, 2)
	s.NoError(err)
	s.Len(deletedIDs, 2)
	s.Equal(ids, deletedIDs)
}
