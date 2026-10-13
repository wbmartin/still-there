package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"

	"github.com/wbmartin/starter-api/internal/model"
)

func TestHelpEventRepository(t *testing.T) {
	suite.Run(t, &HelpEventRepositoryTestSuite{})
}

type HelpEventRepositoryTestSuite struct {
	suite.Suite
	db   *pgxpool.Pool
	repo *HelpEventRepository
}

// Every row this suite writes is on this route, so teardown removes them
// and nothing else.
const helpTestRoute = "/repoTestHelp"

// A login id no fixture uses, so the daily count is this suite's alone.
const helpTestUserID = 900701

func (s *HelpEventRepositoryTestSuite) SetupSuite() {
	s.db = connectTestDB(s.T())
	s.repo = NewHelpEventRepository(s.db)
	s.removeRows()
}

func (s *HelpEventRepositoryTestSuite) TearDownSuite() {
	s.removeRows()
	s.db.Close()
}

func (s *HelpEventRepositoryTestSuite) SetupTest() {
	s.removeRows()
}

// removeRows deletes this suite's rows (test data, scratch database).
func (s *HelpEventRepositoryTestSuite) removeRows() {
	_, err := s.db.Exec(context.Background(),
		`DELETE FROM help_event WHERE route = $1`, helpTestRoute)
	s.Require().NoError(err)
}

// createIn stores one event in securityContext; key is the entry id, or
// the query for search_no_result.
func (s *HelpEventRepositoryTestSuite) createIn(
	event string, key string, createTS float64, securityContext int,
) int {
	row := &model.HelpEventModel{
		AppUserID:    helpTestUserID,
		Event:        event,
		Route:        helpTestRoute,
		CreateTS:     createTS,
		CreateUserID: helpTestUserID,
		UpdateTS:     createTS,
		UpdateUserID: helpTestUserID,
	}
	if event == model.HelpEventSearchNoResult {
		row.Query = &key
	} else {
		row.EntryID = &key
	}
	helpEventID, err := s.repo.CreateHelpEvent(context.Background(), row,
		securityContext)
	s.Require().NoError(err)
	s.Require().Positive(helpEventID)
	return helpEventID
}

// The row's tenant is the argument, not anything on the model.
func (s *HelpEventRepositoryTestSuite) TestCreateStoresTheArgumentTenant() {
	helpEventID := s.createIn(model.HelpEventEntryView, "glossary-tenant",
		model.Now(), testTenant)
	var securityContext int
	var entryID, query *string
	s.Require().NoError(s.db.QueryRow(context.Background(), `
		SELECT security_context_id, entry_id, query FROM help_event
		WHERE help_event_id = $1`, helpEventID,
	).Scan(&securityContext, &entryID, &query))
	s.Equal(testTenant, securityContext)
	s.Require().NotNil(entryID)
	s.Equal("glossary-tenant", *entryID)
	s.Nil(query)
}

// Counts group by key, most frequent first then by key, honour since and
// the limit, and count each event kind on its own column.
func (s *HelpEventRepositoryTestSuite) TestInsightCountList() {
	ctx := context.Background()
	now := model.Now()
	for _, key := range []string{"b-topic", "a-topic", "b-topic", "c-topic"} {
		s.createIn(model.HelpEventEntryView, key, now, testTenant)
	}
	s.createIn(model.HelpEventEntryView, "old-topic", now-90*86400,
		testTenant)
	s.createIn(model.HelpEventSearchNoResult, "invoice", now, testTenant)
	s.createIn(model.HelpEventSearchNoResult, "invoice", now, testTenant)

	viewList, err := s.repo.RetrieveHelpInsightCountList(ctx,
		model.HelpEventEntryView, now-86400, 25, testTenant)
	s.Require().NoError(err)
	s.Equal([]model.HelpInsightCount{
		{Key: "b-topic", Count: 2},
		{Key: "a-topic", Count: 1},
		{Key: "c-topic", Count: 1},
	}, viewList, "the 90-day-old view is outside the window")

	limitedList, err := s.repo.RetrieveHelpInsightCountList(ctx,
		model.HelpEventEntryView, now-86400, 1, testTenant)
	s.Require().NoError(err)
	s.Len(limitedList, 1)

	searchList, err := s.repo.RetrieveHelpInsightCountList(ctx,
		model.HelpEventSearchNoResult, now-86400, 25, testTenant)
	s.Require().NoError(err)
	s.Equal([]model.HelpInsightCount{{Key: "invoice", Count: 2}},
		searchList)

	emptyList, err := s.repo.RetrieveHelpInsightCountList(ctx,
		model.HelpEventTourSkip, now-86400, 25, testTenant)
	s.Require().NoError(err)
	s.NotNil(emptyList, "an empty list, not null, for the JSON answer")
	s.Empty(emptyList)

	_, err = s.repo.RetrieveHelpInsightCountList(ctx, "panel_open",
		now-86400, 25, testTenant)
	s.Error(err, "an event the table does not take is refused")
}

// The daily count is one user's, since a time.
func (s *HelpEventRepositoryTestSuite) TestEventCountSince() {
	now := model.Now()
	s.createIn(model.HelpEventTourComplete, "example-list", now, testTenant)
	s.createIn(model.HelpEventTourSkip, "example-list", now, testTenant)
	s.createIn(model.HelpEventTourSkip, "example-list", now-2*86400,
		testTenant)

	count, err := s.repo.RetrieveHelpEventCountSince(context.Background(),
		helpTestUserID, now-86400, testTenant)
	s.Require().NoError(err)
	s.Equal(2, count)
}

// Cross-tenant negative test (doc/110): tenant 3's events are invisible
// from tenant 2, to the insights counts and to the daily cap, and the
// other way round.
func (s *HelpEventRepositoryTestSuite) TestCrossTenantIsolation() {
	ctx := context.Background()
	now := model.Now()
	s.createIn(model.HelpEventEntryView, "other-tenant-topic", now,
		otherTenant)
	s.createIn(model.HelpEventSearchNoResult, "other tenant search", now,
		otherTenant)

	// Read from tenant 2 first: proves the read sees what is there.
	s.createIn(model.HelpEventEntryView, "own-topic", now, testTenant)
	viewList, err := s.repo.RetrieveHelpInsightCountList(ctx,
		model.HelpEventEntryView, now-86400, 25, testTenant)
	s.Require().NoError(err)
	s.Equal([]model.HelpInsightCount{{Key: "own-topic", Count: 1}},
		viewList)

	searchList, err := s.repo.RetrieveHelpInsightCountList(ctx,
		model.HelpEventSearchNoResult, now-86400, 25, testTenant)
	s.Require().NoError(err)
	s.Empty(searchList, "tenant 3's search text never reaches tenant 2")

	count, err := s.repo.RetrieveHelpEventCountSince(ctx, helpTestUserID,
		now-86400, testTenant)
	s.Require().NoError(err)
	s.Equal(1, count, "tenant 3's events do not count toward tenant 2's cap")

	otherList, err := s.repo.RetrieveHelpInsightCountList(ctx,
		model.HelpEventEntryView, now-86400, 25, otherTenant)
	s.Require().NoError(err)
	s.Equal([]model.HelpInsightCount{
		{Key: "other-tenant-topic", Count: 1}}, otherList)
}
