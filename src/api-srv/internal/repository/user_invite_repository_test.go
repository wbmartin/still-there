package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	"github.com/wbmartin/starter-api/internal/model"
)

// B035: the invite repository against the scratch database, in the test
// tenant. Every login here is named RepoTestInvitee*; the suite removes
// them and their invites before and after.

func TestUserInviteRepository(t *testing.T) {
	suite.Run(t, &UserInviteRepositoryTestSuite{})
}

type UserInviteRepositoryTestSuite struct {
	suite.Suite
	db                   *pgxpool.Pool
	userInviteRepository *UserInviteRepository
}

const inviteAdminUserID = 1

func (s *UserInviteRepositoryTestSuite) SetupSuite() {
	s.db = connectTestDB(s.T())
	s.userInviteRepository = NewUserInviteRepository(s.db)
	s.wipe()
}

func (s *UserInviteRepositoryTestSuite) TearDownSuite() {
	s.wipe()
	s.db.Close()
}

func (s *UserInviteRepositoryTestSuite) wipe() {
	ctx := context.Background()
	_, err := s.db.Exec(ctx, `
		DELETE FROM user_invite WHERE app_user_id IN (
		  SELECT app_user_id FROM app_user
		  WHERE user_name LIKE 'RepoTestInvitee%')`)
	s.Require().NoError(err)
	_, err = s.db.Exec(ctx,
		"DELETE FROM app_user WHERE user_name LIKE 'RepoTestInvitee%'")
	s.Require().NoError(err)
}

func (s *UserInviteRepositoryTestSuite) invitee(
	userName string,
) *model.AppUserModel {
	now := model.Now()
	return &model.AppUserModel{
		UserName: userName, FirstName: "Ivy", LastName: "Vine",
		Email: userName + "@example.com", SecurityContext: testTenant,
		AppUserGroupMembershipList: []int{1},
		CreateTS:                   now, CreateUserID: inviteAdminUserID,
		UpdateTS: now, UpdateUserID: inviteAdminUserID,
	}
}

func (s *UserInviteRepositoryTestSuite) accountStatus(userName string) string {
	var status string
	s.Require().NoError(s.db.QueryRow(context.Background(),
		"SELECT account_status FROM app_user WHERE user_name = $1", userName,
	).Scan(&status))
	return status
}

// The invited login cannot be redeemed twice, becomes active with the new
// hash, and the redeeming address is recorded.
func (s *UserInviteRepositoryTestSuite) TestInviteThenRedeem() {
	ctx := context.Background()
	now := model.Now()
	created, err := s.userInviteRepository.CreateInvitedUser(ctx,
		s.invitee("RepoTestInvitee1"), testTenant, "hash-1", now+3600,
		inviteAdminUserID)
	s.Require().NoError(err)
	s.NotZero(created.AppUserID)
	s.Equal(model.AccountStatusInvited, created.AccountStatus)
	s.Equal(model.AccountStatusInvited, s.accountStatus("RepoTestInvitee1"))

	preview, err := s.userInviteRepository.RetrieveLiveInvite(ctx,
		"hash-1", now)
	s.Require().NoError(err)
	s.Equal("RepoTestInvitee1", preview.UserName)
	s.Equal("Ivy", preview.FirstName)

	userName, err := s.userInviteRepository.RedeemInvite(ctx, "hash-1",
		"argon-hash", "10.0.0.9")
	s.Require().NoError(err)
	s.Equal("RepoTestInvitee1", userName)
	s.Equal(model.AccountStatusActive, s.accountStatus("RepoTestInvitee1"))

	var pwHash, sourceIP string
	s.Require().NoError(s.db.QueryRow(ctx, `
		SELECT u.pw_hash, ui.redeemed_source_ip
		FROM app_user u JOIN user_invite ui ON ui.app_user_id = u.app_user_id
		WHERE u.user_name = 'RepoTestInvitee1'`).Scan(&pwHash, &sourceIP))
	s.Equal("argon-hash", pwHash)
	s.Equal("10.0.0.9", sourceIP)

	_, err = s.userInviteRepository.RedeemInvite(ctx, "hash-1", "again",
		"10.0.0.9")
	s.ErrorIs(err, ErrNotFound, "a link is single use")
	_, err = s.userInviteRepository.RetrieveLiveInvite(ctx, "hash-1", now)
	s.ErrorIs(err, ErrNotFound)
}

// Past expires_ts, neither public read can use it.
func (s *UserInviteRepositoryTestSuite) TestExpiredLinkIsDead() {
	ctx := context.Background()
	now := model.Now()
	_, err := s.userInviteRepository.CreateInvitedUser(ctx,
		s.invitee("RepoTestInvitee2"), testTenant, "hash-2", now-1,
		inviteAdminUserID)
	s.Require().NoError(err)

	_, err = s.userInviteRepository.RetrieveLiveInvite(ctx, "hash-2", now)
	s.ErrorIs(err, ErrNotFound)
	_, err = s.userInviteRepository.RedeemInvite(ctx, "hash-2",
		"argon-hash", "10.0.0.9")
	s.ErrorIs(err, ErrNotFound)
	s.Equal(model.AccountStatusInvited, s.accountStatus("RepoTestInvitee2"))
}

// A reissue revokes the old link, a revoke leaves nothing live, a second
// revoke is a conflict, and another tenant's id is not found.
func (s *UserInviteRepositoryTestSuite) TestReissueAndRevoke() {
	ctx := context.Background()
	now := model.Now()
	created, err := s.userInviteRepository.CreateInvitedUser(ctx,
		s.invitee("RepoTestInvitee3"), testTenant, "hash-3a", now+3600,
		inviteAdminUserID)
	s.Require().NoError(err)

	s.Require().NoError(s.userInviteRepository.ReissueInvite(ctx,
		created.AppUserID, testTenant, "hash-3b", now+3600,
		inviteAdminUserID))
	_, err = s.userInviteRepository.RetrieveLiveInvite(ctx, "hash-3a", now)
	s.ErrorIs(err, ErrNotFound, "the old link is revoked")
	_, err = s.userInviteRepository.RetrieveLiveInvite(ctx, "hash-3b", now)
	s.NoError(err)

	s.ErrorIs(s.userInviteRepository.ReissueInvite(ctx, created.AppUserID,
		otherTenant, "hash-3c", now+3600, inviteAdminUserID), ErrNotFound)
	s.ErrorIs(s.userInviteRepository.RevokeInvite(ctx, created.AppUserID,
		otherTenant, inviteAdminUserID), ErrNotFound)

	s.Require().NoError(s.userInviteRepository.RevokeInvite(ctx,
		created.AppUserID, testTenant, inviteAdminUserID))
	_, err = s.userInviteRepository.RetrieveLiveInvite(ctx, "hash-3b", now)
	s.ErrorIs(err, ErrNotFound)
	s.ErrorIs(s.userInviteRepository.RevokeInvite(ctx, created.AppUserID,
		testTenant, inviteAdminUserID), ErrConflict)
	s.Equal(model.AccountStatusInvited, s.accountStatus("RepoTestInvitee3"))
}

// An active login has no use for a link: reissuing for one is not found.
func (s *UserInviteRepositoryTestSuite) TestActiveLoginHasNoInvite() {
	ctx := context.Background()
	user := s.invitee("RepoTestInvitee4")
	user.PwHash = "argon-hash"
	created, err := NewUserRepository(s.db).CreateUser(ctx, user,
		testTenant)
	s.Require().NoError(err)
	s.ErrorIs(s.userInviteRepository.ReissueInvite(ctx, created.AppUserID,
		testTenant, "hash-4", model.Now()+3600, inviteAdminUserID),
		ErrNotFound)
}

// A taken user name is ErrDuplicate and leaves no invite behind.
func (s *UserInviteRepositoryTestSuite) TestDuplicateUserNameWritesNothing() {
	ctx := context.Background()
	now := model.Now()
	_, err := s.userInviteRepository.CreateInvitedUser(ctx,
		s.invitee("RepoTestInvitee5"), testTenant, "hash-5a", now+3600,
		inviteAdminUserID)
	s.Require().NoError(err)
	_, err = s.userInviteRepository.CreateInvitedUser(ctx,
		s.invitee("RepoTestInvitee5"), testTenant, "hash-5b", now+3600,
		inviteAdminUserID)
	s.ErrorIs(err, ErrDuplicate)
	var count int
	s.Require().NoError(s.db.QueryRow(ctx,
		"SELECT count(*) FROM user_invite WHERE token_hash = 'hash-5b'",
	).Scan(&count))
	s.Zero(count)
}
