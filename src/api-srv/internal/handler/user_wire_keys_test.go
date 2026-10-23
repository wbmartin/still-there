package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"stillthere-api/internal/auth"
	"stillthere-api/internal/model"
)

// POST /user reads the raw password from "password" and stores only its
// Argon2id hash; the old "pwHash" key no longer reaches the handler.
func TestCreateUser_ReadsThePasswordKey(t *testing.T) {
	const password = "wire-key-pass1!"
	s := testServices(t)
	token := sessionToken(t, s, "alice", 1, "session-abc",
		[]int{model.GrantUserCreate})

	t.Run("password → created with its hash", func(t *testing.T) {
		repo := new(MockUserRepository)
		repo.On("CreateUser", mock.Anything,
			mock.MatchedBy(func(user *model.AppUserModel) bool {
				return user.UserName == "bob" &&
					auth.CheckPasswordHash(password, user.PwHash)
			}), 1).
			Return(&model.AppUserModel{AppUserID: 9, UserName: "bob"}, nil)
		handler := NewUserHandler(s, repo, new(MockAppUserGroupRepository))

		w := httptest.NewRecorder()
		handler.CreateUser(w, sessionRequest(http.MethodPost, "/user",
			`{"userName":"bob","password":"`+password+`",`+
				`"firstName":"Bob","lastName":"B","email":"b@example.com",`+
				`"appUserGroupMembershipList":[]}`,
			token, "session-abc"))
		assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		assert.NotContains(t, w.Body.String(), password)
		repo.AssertExpectations(t)
	})

	t.Run("old pwHash key → 400, nothing stored", func(t *testing.T) {
		repo := new(MockUserRepository)
		handler := NewUserHandler(s, repo, new(MockAppUserGroupRepository))

		w := httptest.NewRecorder()
		handler.CreateUser(w, sessionRequest(http.MethodPost, "/user",
			`{"userName":"bob","pwHash":"`+password+`",`+
				`"firstName":"Bob","lastName":"B","email":"b@example.com",`+
				`"appUserGroupMembershipList":[]}`,
			token, "session-abc"))
		assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		repo.AssertExpectations(t)
	})
}

// The group-membership routes read userIdList and appUserGroupId; the old
// userIds / groupId keys decode to nothing and are refused before the
// repository.
func TestUserListGroupRoutes_ReadTheListKeys(t *testing.T) {
	s := testServices(t)
	token := sessionToken(t, s, "alice", 1, "session-abc",
		[]int{model.GrantUserGroupUpdate})
	routeList := []struct {
		name       string
		repoMethod string
		call       func(*UserHandler, http.ResponseWriter, *http.Request)
	}{
		{"/addUserListToGroup", "AddUserListToGroup",
			(*UserHandler).AddUserListToGroupHandler},
		{"/removeUserListFromGroup", "RemoveUserListFromGroup",
			(*UserHandler).RemoveUserListFromGroupHandler},
	}
	for _, route := range routeList {
		t.Run(route.name+" new keys reach the repository",
			func(t *testing.T) {
				repo := new(MockUserRepository)
				groupRepo := new(MockAppUserGroupRepository)
				groupRepo.On("RetrieveAppUserGroupListByIDList",
					mock.Anything, []int{7}, 1).
					Return([]*model.AppUserGroupModel{
						{AppUserGroupID: 7, SecurityContext: 1}}, nil)
				repo.On("RetrieveUserList", mock.Anything, 1, []int{5},
					float64(0)).
					Return([]*model.AppUserModel{
						{AppUserID: 5, SecurityContext: 1}}, nil)
				repo.On(route.repoMethod, mock.Anything, []int{5}, 7,
					[]float64{12.5}, 1, 1).
					Return([]*model.AppUserModel{
						{AppUserID: 5, UserName: "bob"}}, nil)
				handler := NewUserHandler(s, repo, groupRepo)

				w := httptest.NewRecorder()
				route.call(handler, w, sessionRequest(http.MethodPost,
					route.name,
					`{"userIdList":[5],"appUserGroupId":7,`+
						`"updateTsList":[12.5]}`,
					token, "session-abc"))
				assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
				repo.AssertExpectations(t)
				groupRepo.AssertExpectations(t)
			})

		t.Run(route.name+" old keys → 400", func(t *testing.T) {
			repo := new(MockUserRepository)
			groupRepo := new(MockAppUserGroupRepository)
			handler := NewUserHandler(s, repo, groupRepo)

			w := httptest.NewRecorder()
			route.call(handler, w, sessionRequest(http.MethodPost,
				route.name,
				`{"userIds":[5],"groupId":7,"updateTsList":[12.5]}`,
				token, "session-abc"))
			assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			assert.Contains(t, w.Body.String(), "userIdList")
			repo.AssertExpectations(t)
			groupRepo.AssertExpectations(t)
		})
	}
}
