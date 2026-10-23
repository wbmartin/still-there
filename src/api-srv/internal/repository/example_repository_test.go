package repository

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	"stillthere-api/internal/model"
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
	db := connectTestDB(s.T())
	s.db = db
	s.repo = NewExampleRepository(db)

	s.cleanup = func() {
		// Clean only test data
		_, _ = db.Exec(
			context.Background(),
			"DELETE FROM example WHERE example_text LIKE 'Test%' OR example_text LIKE 'RepoTest%'",
		)
	}
}

func (s *ExampleRepositoryTestSuite) TearDownSuite() {
	s.cleanup()
	s.db.Close()
}

func (s *ExampleRepositoryTestSuite) TestCreateExample() {
	example := &model.ExampleModel{
		ExampleText:     "RepoTest Example",
		SecurityContext: testTenant,
		ExampleUUID:     model.NewUUID(),
		CreateTS:        1234567890.123,
		CreateUserID:    1,
		UpdateTS:        1234567890.123,
		UpdateUserID:    1,
	}

	created, err := s.repo.CreateExample(context.Background(), example, testTenant)
	s.NoError(err)
	s.NotNil(created)
	s.NotZero(created.ExampleID)
	s.Equal("RepoTest Example", created.ExampleText)
}

func (s *ExampleRepositoryTestSuite) TestRetrieveExample() {
	// Create test data first
	example := &model.ExampleModel{
		ExampleText:     "RepoTest Read Example",
		SecurityContext: testTenant,
		ExampleUUID:     model.NewUUID(),
		CreateTS:        1234567890.123,
		CreateUserID:    1,
		UpdateTS:        1234567890.123,
		UpdateUserID:    1,
	}

	created, _ := s.repo.CreateExample(context.Background(), example, testTenant)

	// Test retrieval
	retrieved, err := s.repo.RetrieveExample(
		context.Background(),
		created.ExampleID,
		testTenant,
	)
	s.NoError(err)
	s.Equal(created.ExampleID, retrieved.ExampleID)
	s.Equal("RepoTest Read Example", retrieved.ExampleText)
}

func (s *ExampleRepositoryTestSuite) TestUpdateExample() {
	// Create test data
	example := &model.ExampleModel{
		ExampleText:     "RepoTest Update Example",
		SecurityContext: testTenant,
		ExampleUUID:     model.NewUUID(),
		CreateTS:        1234567890.123,
		CreateUserID:    1,
		UpdateTS:        1234567890.123,
		UpdateUserID:    1,
	}

	created, _ := s.repo.CreateExample(context.Background(), example, testTenant)

	// Update it
	created.ExampleText = "RepoTest Updated Example"
	created.UpdateTS = 1234567891.456
	created.UpdateUserID = 2

	err := s.repo.UpdateExample(context.Background(), created, 1234567890.123)
	s.NoError(err)

	// Verify update
	updated, _ := s.repo.RetrieveExample(
		context.Background(),
		created.ExampleID,
		testTenant,
	)
	s.Equal("RepoTest Updated Example", updated.ExampleText)
}

func (s *ExampleRepositoryTestSuite) TestDeleteExample() {
	// Create test data
	example := &model.ExampleModel{
		ExampleText:     "RepoTest Delete Example",
		SecurityContext: testTenant,
		ExampleUUID:     model.NewUUID(),
		CreateTS:        1234567890.123,
		CreateUserID:    1,
		UpdateTS:        1234567890.123,
		UpdateUserID:    1,
	}

	created, _ := s.repo.CreateExample(context.Background(), example, testTenant)

	// Delete it
	deletedID, err := s.repo.DeleteExample(
		context.Background(),
		created.ExampleID,
		testTenant,
		1234567890.123,
		1234567892.789,
		2,
	)
	s.NoError(err)
	s.Equal(created.ExampleID, deletedID)

	// Verify deletion (soft delete) - should return ErrDeleted
	_, err = s.repo.RetrieveExample(context.Background(), created.ExampleID, testTenant)
	s.Error(err)
	s.True(errors.Is(err, ErrDeleted))
}

func (s *ExampleRepositoryTestSuite) TestCreateExampleList() {
	examples := []*model.ExampleModel{
		{
			ExampleText:     "RepoTest List Example 1",
			SecurityContext: testTenant,
			ExampleUUID:     model.NewUUID(),
			CreateTS:        1234567890.123,
			CreateUserID:    1,
			UpdateTS:        1234567890.123,
			UpdateUserID:    1,
		},
		{
			ExampleText:     "RepoTest List Example 2",
			SecurityContext: testTenant,
			ExampleUUID:     model.NewUUID(),
			CreateTS:        1234567890.123,
			CreateUserID:    1,
			UpdateTS:        1234567890.123,
			UpdateUserID:    1,
		},
	}

	created, err := s.repo.CreateExampleList(context.Background(), examples, testTenant)
	s.NoError(err)
	s.Len(created, 2)
	s.NotZero(created[0].ExampleID)
	s.NotZero(created[1].ExampleID)
}

func (s *ExampleRepositoryTestSuite) TestRetrieveExampleList() {
	// Create test data
	example1 := &model.ExampleModel{
		ExampleText:     "RepoTest List Get 1",
		SecurityContext: testTenant,
		ExampleUUID:     model.NewUUID(),
		CreateTS:        1234567890.123,
		CreateUserID:    1,
		UpdateTS:        1234567890.123,
		UpdateUserID:    1,
	}
	example2 := &model.ExampleModel{
		ExampleText:     "RepoTest List Get 2",
		SecurityContext: testTenant,
		ExampleUUID:     model.NewUUID(),
		CreateTS:        1234567890.123,
		CreateUserID:    1,
		UpdateTS:        1234567890.123,
		UpdateUserID:    1,
	}

	created1, _ := s.repo.CreateExample(context.Background(), example1, testTenant)
	created2, _ := s.repo.CreateExample(context.Background(), example2, testTenant)

	// Test list retrieval
	ids := []int{created1.ExampleID, created2.ExampleID}
	retrieved, err := s.repo.RetrieveExampleList(
		context.Background(),
		ids,
		testTenant,
		0,
	)
	s.NoError(err)
	s.Len(retrieved, 2)
}

func (s *ExampleRepositoryTestSuite) TestUpdateExampleList() {
	// Create test data
	examples := []*model.ExampleModel{
		{
			ExampleText:     "RepoTest Update List 1",
			SecurityContext: testTenant,
			ExampleUUID:     model.NewUUID(),
			CreateTS:        1234567890.123,
			CreateUserID:    1,
			UpdateTS:        1234567890.123,
			UpdateUserID:    1,
		},
		{
			ExampleText:     "RepoTest Update List 2",
			SecurityContext: testTenant,
			ExampleUUID:     model.NewUUID(),
			CreateTS:        1234567890.123,
			CreateUserID:    1,
			UpdateTS:        1234567890.123,
			UpdateUserID:    1,
		},
	}

	created, _ := s.repo.CreateExampleList(context.Background(), examples, testTenant)

	// Update them
	created[0].ExampleText = "RepoTest Updated List 1"
	created[1].ExampleText = "RepoTest Updated List 2"
	created[0].UpdateTS = 1234567891.456
	created[1].UpdateTS = 1234567891.456
	created[0].UpdateUserID = 2
	created[1].UpdateUserID = 2

	originalUpdateTSList := []float64{1234567890.123, 1234567890.123}
	err := s.repo.UpdateExampleList(
		context.Background(),
		created,
		originalUpdateTSList,
		testTenant,
	)
	s.NoError(err)
}

func (s *ExampleRepositoryTestSuite) TestDeleteExampleList() {
	// Create test data
	examples := []*model.ExampleModel{
		{
			ExampleText:     "RepoTest Delete List 1",
			SecurityContext: testTenant,
			ExampleUUID:     model.NewUUID(),
			CreateTS:        1234567890.123,
			CreateUserID:    1,
			UpdateTS:        1234567890.123,
			UpdateUserID:    1,
		},
		{
			ExampleText:     "RepoTest Delete List 2",
			SecurityContext: testTenant,
			ExampleUUID:     model.NewUUID(),
			CreateTS:        1234567890.123,
			CreateUserID:    1,
			UpdateTS:        1234567890.123,
			UpdateUserID:    1,
		},
	}

	created, _ := s.repo.CreateExampleList(context.Background(), examples, testTenant)
	ids := []int{created[0].ExampleID, created[1].ExampleID}
	updateTs := []float64{1234567890.123, 1234567890.123}

	// Delete them
	deletedIDs, err := s.repo.DeleteExampleList(
		context.Background(),
		ids,
		updateTs,
		testTenant,
		1234567892.789,
		2,
	)
	s.NoError(err)
	s.Len(deletedIDs, 2)
	s.Equal(ids, deletedIDs)
}

// example_uuid is fixed at create and example_jsonb is stored verbatim; both
// come back on a read, and an update keeps the uuid and rewrites the jsonb.
func (s *ExampleRepositoryTestSuite) TestJSONBAndUUIDRoundTrip() {
	uuid := model.NewUUID()
	example := &model.ExampleModel{
		ExampleText:     "RepoTest JSONB UUID",
		ExampleUUID:     uuid,
		ExampleJSONB:    json.RawMessage(`{"colour": "teal", "count": 3, "tagList": ["a", "b"]}`),
		SecurityContext: testTenant,
		CreateTS:        1234567890.123,
		CreateUserID:    1,
		UpdateTS:        1234567890.123,
		UpdateUserID:    1,
	}
	created, err := s.repo.CreateExample(context.Background(), example, testTenant)
	s.NoError(err)

	retrieved, err := s.repo.RetrieveExample(context.Background(), created.ExampleID, testTenant)
	s.NoError(err)
	s.Equal(uuid, retrieved.ExampleUUID)
	s.JSONEq(`{"colour": "teal", "count": 3, "tagList": ["a", "b"]}`, string(retrieved.ExampleJSONB))

	// Update: a client-supplied uuid is ignored, the jsonb is replaced, and
	// the row's uuid comes back on the updated copy.
	retrieved.ExampleUUID = "00000000-0000-4000-8000-000000000000"
	retrieved.ExampleJSONB = json.RawMessage(`[1, 2, 3]`)
	retrieved.UpdateTS = 1234567891.456
	s.NoError(s.repo.UpdateExample(context.Background(), retrieved, 1234567890.123))
	s.Equal(uuid, retrieved.ExampleUUID)

	updated, err := s.repo.RetrieveExample(context.Background(), created.ExampleID, testTenant)
	s.NoError(err)
	s.Equal(uuid, updated.ExampleUUID)
	s.JSONEq(`[1, 2, 3]`, string(updated.ExampleJSONB))

	// Absent jsonb is SQL NULL, which reads back as a nil RawMessage.
	updated.ExampleJSONB = nil
	updated.UpdateTS = 1234567892.789
	s.NoError(s.repo.UpdateExample(context.Background(), updated, 1234567891.456))
	cleared, err := s.repo.RetrieveExample(context.Background(), created.ExampleID, testTenant)
	s.NoError(err)
	s.Nil(cleared.ExampleJSONB)

	// The uuid is unique across rows.
	dup := &model.ExampleModel{
		ExampleText: "RepoTest dup uuid", ExampleUUID: uuid, SecurityContext: testTenant,
		CreateTS: 1, CreateUserID: 1, UpdateTS: 1, UpdateUserID: 1,
	}
	_, err = s.repo.CreateExample(context.Background(), dup, testTenant)
	s.Error(err)
}

// B098: tenant isolation. A row of otherTenant (3) is invisible to testTenant
// (2) through every repository entry point: reads, updates and deletes behave
// as if the row did not exist, and the row is left exactly as it was.

func (s *ExampleRepositoryTestSuite) createIn(
	text string,
	tenant int,
) *model.ExampleModel {
	created, err := s.repo.CreateExample(context.Background(),
		&model.ExampleModel{
			ExampleText: text, SecurityContext: tenant,
			ExampleUUID: model.NewUUID(),
			CreateTS:    1234567890.123, CreateUserID: 1,
			UpdateTS: 1234567890.123, UpdateUserID: 1,
		}, tenant)
	s.Require().NoError(err)
	return created
}

func (s *ExampleRepositoryTestSuite) TestCrossTenantIsolation() {
	ctx := context.Background()
	mine := s.createIn("RepoTest Tenant Mine", testTenant)
	theirs := s.createIn("RepoTest Tenant Theirs", otherTenant)

	// List: tenant 2 sees its own row and never the tenant-3 row, whether
	// the list is asked for by id or in full.
	byID, err := s.repo.RetrieveExampleList(ctx,
		[]int{mine.ExampleID, theirs.ExampleID}, testTenant, 0)
	s.Require().NoError(err)
	s.Require().Len(byID, 1)
	s.Equal(mine.ExampleID, byID[0].ExampleID)

	full, err := s.repo.RetrieveExampleList(ctx, nil, testTenant, 0)
	s.Require().NoError(err)
	for _, example := range full {
		s.Equal(testTenant, example.SecurityContext)
		s.NotEqual(theirs.ExampleID, example.ExampleID)
	}

	// Get.
	_, err = s.repo.RetrieveExample(ctx, theirs.ExampleID, testTenant)
	s.True(errors.Is(err, ErrNotFound), "get: %v", err)

	// Update: the tenant-3 id with tenant 2 as the caller's context.
	attempt := *theirs
	attempt.SecurityContext = testTenant
	attempt.ExampleText = "RepoTest Tenant Hijacked"
	attempt.UpdateTS = 1234567899.0
	err = s.repo.UpdateExample(ctx, &attempt, theirs.UpdateTS)
	s.True(errors.Is(err, ErrNotFound), "update: %v", err)

	// Delete.
	_, err = s.repo.DeleteExample(ctx, theirs.ExampleID, testTenant,
		theirs.UpdateTS, 1234567899.0, 2)
	s.True(errors.Is(err, ErrNotFound), "delete: %v", err)

	// Batch update and delete leave it alone too.
	err = s.repo.UpdateExampleList(ctx, []*model.ExampleModel{&attempt},
		[]float64{theirs.UpdateTS}, testTenant)
	s.True(errors.Is(err, ErrExpectationFailed), "update list: %v", err)
	_, err = s.repo.DeleteExampleList(ctx, []int{theirs.ExampleID},
		[]float64{theirs.UpdateTS}, testTenant, 1234567899.0, 2)
	s.NoError(err)

	// The tenant-3 row is unchanged and still live for its owner.
	after, err := s.repo.RetrieveExample(ctx, theirs.ExampleID, otherTenant)
	s.Require().NoError(err)
	s.Equal("RepoTest Tenant Theirs", after.ExampleText)
	s.Equal(theirs.UpdateTS, after.UpdateTS)
	s.Nil(after.DeleteUserID)
	s.Equal(otherTenant, after.SecurityContext)
}
