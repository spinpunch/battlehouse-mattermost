// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package api4

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/mattermost/platform/app"
	"github.com/mattermost/platform/model"
	"github.com/mattermost/platform/utils"
)

func TestCreateUser(t *testing.T) {
	th := Setup().InitBasic()
	defer TearDown()
	Client := th.Client

	user := model.User{Email: GenerateTestEmail(), Nickname: "Corey Hulen", Password: "hello1", Username: GenerateTestUsername(), Roles: model.ROLE_SYSTEM_ADMIN.Id + " " + model.ROLE_SYSTEM_USER.Id}

	ruser, resp := Client.CreateUser(&user)
	CheckNoError(t, resp)

	Client.Login(user.Email, user.Password)

	if ruser.Nickname != user.Nickname {
		t.Fatal("nickname didn't match")
	}

	if ruser.Roles != model.ROLE_SYSTEM_USER.Id {
		t.Log(ruser.Roles)
		t.Fatal("did not clear roles")
	}

	CheckUserSanitization(t, ruser)

	_, resp = Client.CreateUser(ruser)
	CheckBadRequestStatus(t, resp)

	ruser.Id = ""
	ruser.Username = GenerateTestUsername()
	ruser.Password = "passwd1"
	_, resp = Client.CreateUser(ruser)
	CheckErrorMessage(t, resp, "store.sql_user.save.email_exists.app_error")
	CheckBadRequestStatus(t, resp)

	ruser.Email = GenerateTestEmail()
	ruser.Username = user.Username
	_, resp = Client.CreateUser(ruser)
	CheckErrorMessage(t, resp, "store.sql_user.save.username_exists.app_error")
	CheckBadRequestStatus(t, resp)

	ruser.Email = ""
	_, resp = Client.CreateUser(ruser)
	CheckErrorMessage(t, resp, "model.user.is_valid.email.app_error")
	CheckBadRequestStatus(t, resp)

	ruser.Email = GenerateTestEmail()
	ruser.Username = "1" + user.Username
	_, resp = Client.CreateUser(ruser)
	CheckErrorMessage(t, resp, "model.user.is_valid.username.app_error")
	CheckBadRequestStatus(t, resp)

	if r, err := Client.DoApiPost("/users", "garbage"); err == nil {
		t.Fatal("should have errored")
	} else {
		if r.StatusCode != http.StatusBadRequest {
			t.Log("actual: " + strconv.Itoa(r.StatusCode))
			t.Log("expected: " + strconv.Itoa(http.StatusBadRequest))
			t.Fatal("wrong status code")
		}
	}
}

func TestGetUser(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer TearDown()
	Client := th.Client

	user := th.CreateUser()

	ruser, resp := Client.GetUser(user.Id, "")
	CheckNoError(t, resp)
	CheckUserSanitization(t, ruser)

	if ruser.Email != user.Email {
		t.Fatal("emails did not match")
	}

	ruser, resp = Client.GetUser(user.Id, resp.Etag)
	CheckEtag(t, ruser, resp)

	_, resp = Client.GetUser("junk", "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetUser(model.NewId(), "")
	CheckNotFoundStatus(t, resp)

	// Check against privacy config settings
	emailPrivacy := utils.Cfg.PrivacySettings.ShowEmailAddress
	namePrivacy := utils.Cfg.PrivacySettings.ShowFullName
	defer func() {
		utils.Cfg.PrivacySettings.ShowEmailAddress = emailPrivacy
		utils.Cfg.PrivacySettings.ShowFullName = namePrivacy
	}()
	utils.Cfg.PrivacySettings.ShowEmailAddress = false
	utils.Cfg.PrivacySettings.ShowFullName = false

	ruser, resp = Client.GetUser(user.Id, "")
	CheckNoError(t, resp)

	if ruser.Email != "" {
		t.Fatal("email should be blank")
	}
	if ruser.FirstName != "" {
		t.Fatal("first name should be blank")
	}
	if ruser.LastName != "" {
		t.Fatal("last name should be blank")
	}

	Client.Logout()
	_, resp = Client.GetUser(user.Id, "")
	CheckUnauthorizedStatus(t, resp)

	// System admins should ignore privacy settings
	ruser, resp = th.SystemAdminClient.GetUser(user.Id, resp.Etag)
	if ruser.Email == "" {
		t.Fatal("email should not be blank")
	}
	if ruser.FirstName == "" {
		t.Fatal("first name should not be blank")
	}
	if ruser.LastName == "" {
		t.Fatal("last name should not be blank")
	}
}

func TestGetUserByUsername(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer TearDown()
	Client := th.Client

	user := th.BasicUser

	ruser, resp := Client.GetUserByUsername(user.Username, "")
	CheckNoError(t, resp)
	CheckUserSanitization(t, ruser)

	if ruser.Email != user.Email {
		t.Fatal("emails did not match")
	}

	ruser, resp = Client.GetUserByUsername(user.Username, resp.Etag)
	CheckEtag(t, ruser, resp)

	_, resp = Client.GetUserByUsername(GenerateTestUsername(), "")
	CheckNotFoundStatus(t, resp)

	_, resp = Client.GetUserByUsername(model.NewRandomString(1), "")
	CheckBadRequestStatus(t, resp)

	// Check against privacy config settings
	emailPrivacy := utils.Cfg.PrivacySettings.ShowEmailAddress
	namePrivacy := utils.Cfg.PrivacySettings.ShowFullName
	defer func() {
		utils.Cfg.PrivacySettings.ShowEmailAddress = emailPrivacy
		utils.Cfg.PrivacySettings.ShowFullName = namePrivacy
	}()
	utils.Cfg.PrivacySettings.ShowEmailAddress = false
	utils.Cfg.PrivacySettings.ShowFullName = false

	ruser, resp = Client.GetUserByUsername(user.Username, "")
	CheckNoError(t, resp)

	if ruser.Email != "" {
		t.Fatal("email should be blank")
	}
	if ruser.FirstName != "" {
		t.Fatal("first name should be blank")
	}
	if ruser.LastName != "" {
		t.Fatal("last name should be blank")
	}

	Client.Logout()
	_, resp = Client.GetUserByUsername(user.Username, "")
	CheckUnauthorizedStatus(t, resp)

	// System admins should ignore privacy settings
	ruser, resp = th.SystemAdminClient.GetUserByUsername(user.Username, resp.Etag)
	if ruser.Email == "" {
		t.Fatal("email should not be blank")
	}
	if ruser.FirstName == "" {
		t.Fatal("first name should not be blank")
	}
	if ruser.LastName == "" {
		t.Fatal("last name should not be blank")
	}
}

func TestGetUserByEmail(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer TearDown()
	Client := th.Client

	user := th.CreateUser()

	ruser, resp := Client.GetUserByEmail(user.Email, "")
	CheckNoError(t, resp)
	CheckUserSanitization(t, ruser)

	if ruser.Email != user.Email {
		t.Fatal("emails did not match")
	}

	ruser, resp = Client.GetUserByEmail(user.Email, resp.Etag)
	CheckEtag(t, ruser, resp)

	_, resp = Client.GetUserByEmail(GenerateTestUsername(), "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetUserByEmail(GenerateTestEmail(), "")
	CheckNotFoundStatus(t, resp)

	// Check against privacy config settings
	emailPrivacy := utils.Cfg.PrivacySettings.ShowEmailAddress
	namePrivacy := utils.Cfg.PrivacySettings.ShowFullName
	defer func() {
		utils.Cfg.PrivacySettings.ShowEmailAddress = emailPrivacy
		utils.Cfg.PrivacySettings.ShowFullName = namePrivacy
	}()
	utils.Cfg.PrivacySettings.ShowEmailAddress = false
	utils.Cfg.PrivacySettings.ShowFullName = false

	ruser, resp = Client.GetUserByEmail(user.Email, "")
	CheckNoError(t, resp)

	if ruser.Email != "" {
		t.Fatal("email should be blank")
	}
	if ruser.FirstName != "" {
		t.Fatal("first name should be blank")
	}
	if ruser.LastName != "" {
		t.Fatal("last name should be blank")
	}

	Client.Logout()
	_, resp = Client.GetUserByEmail(user.Email, "")
	CheckUnauthorizedStatus(t, resp)

	// System admins should ignore privacy settings
	ruser, resp = th.SystemAdminClient.GetUserByEmail(user.Email, resp.Etag)
	if ruser.Email == "" {
		t.Fatal("email should not be blank")
	}
	if ruser.FirstName == "" {
		t.Fatal("first name should not be blank")
	}
	if ruser.LastName == "" {
		t.Fatal("last name should not be blank")
	}
}

func TestGetProfileImage(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer TearDown()
	Client := th.Client
	user := th.BasicUser

	data, resp := Client.GetProfileImage(user.Id, "")
	CheckNoError(t, resp)
	if data == nil || len(data) == 0 {
		t.Fatal("Should not be empty")
	}

	_, resp = Client.GetProfileImage(user.Id, resp.Etag)
	if resp.StatusCode != http.StatusNotModified {
		t.Fatal("Should have hit etag")
	}

	_, resp = Client.GetProfileImage("junk", "")
	CheckBadRequestStatus(t, resp)

	Client.Logout()
	_, resp = Client.GetProfileImage(user.Id, "")
	CheckUnauthorizedStatus(t, resp)

	_, resp = th.SystemAdminClient.GetProfileImage(user.Id, "")
	CheckNoError(t, resp)

	info := &model.FileInfo{Path: "/users/" + user.Id + "/profile.png"}
	if err := cleanupTestFile(info); err != nil {
		t.Fatal(err)
	}
}

func TestGetUsersByIds(t *testing.T) {
	th := Setup().InitBasic()
	Client := th.Client

	users, resp := Client.GetUsersByIds([]string{th.BasicUser.Id})
	CheckNoError(t, resp)

	if users[0].Id != th.BasicUser.Id {
		t.Fatal("returned wrong user")
	}
	CheckUserSanitization(t, users[0])

	_, resp = Client.GetUsersByIds([]string{})
	CheckBadRequestStatus(t, resp)

	users, resp = Client.GetUsersByIds([]string{"junk"})
	CheckNoError(t, resp)
	if len(users) > 0 {
		t.Fatal("no users should be returned")
	}

	users, resp = Client.GetUsersByIds([]string{"junk", th.BasicUser.Id})
	CheckNoError(t, resp)
	if len(users) != 1 {
		t.Fatal("1 user should be returned")
	}

	Client.Logout()
	_, resp = Client.GetUsersByIds([]string{th.BasicUser.Id})
	CheckUnauthorizedStatus(t, resp)
}

func TestUpdateUser(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer TearDown()
	Client := th.Client

	user := th.CreateUser()
	Client.Login(user.Email, user.Password)

	user.Nickname = "Joram Wilander"
	user.Roles = model.ROLE_SYSTEM_ADMIN.Id
	user.LastPasswordUpdate = 123

	ruser, resp := Client.UpdateUser(user)
	CheckNoError(t, resp)
	CheckUserSanitization(t, ruser)

	if ruser.Nickname != "Joram Wilander" {
		t.Fatal("Nickname did not update properly")
	}
	if ruser.Roles != model.ROLE_SYSTEM_USER.Id {
		t.Fatal("Roles should not have updated")
	}
	if ruser.LastPasswordUpdate == 123 {
		t.Fatal("LastPasswordUpdate should not have updated")
	}

	ruser.Id = "junk"
	_, resp = Client.UpdateUser(ruser)
	CheckBadRequestStatus(t, resp)

	ruser.Id = model.NewId()
	_, resp = Client.UpdateUser(ruser)
	CheckForbiddenStatus(t, resp)

	if r, err := Client.DoApiPut("/users/"+ruser.Id, "garbage"); err == nil {
		t.Fatal("should have errored")
	} else {
		if r.StatusCode != http.StatusBadRequest {
			t.Log("actual: " + strconv.Itoa(r.StatusCode))
			t.Log("expected: " + strconv.Itoa(http.StatusBadRequest))
			t.Fatal("wrong status code")
		}
	}

	Client.Logout()
	_, resp = Client.UpdateUser(user)
	CheckUnauthorizedStatus(t, resp)

	th.LoginBasic()
	_, resp = Client.UpdateUser(user)
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.UpdateUser(user)
	CheckNoError(t, resp)
}

func TestPatchUser(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer TearDown()
	Client := th.Client

	user := th.CreateUser()
	Client.Login(user.Email, user.Password)

	patch := &model.UserPatch{}

	patch.Nickname = new(string)
	*patch.Nickname = "Joram Wilander"
	patch.FirstName = new(string)
	*patch.FirstName = "Joram"
	patch.LastName = new(string)
	*patch.LastName = "Wilander"
	patch.Position = new(string)

	ruser, resp := Client.PatchUser(user.Id, patch)
	CheckNoError(t, resp)
	CheckUserSanitization(t, ruser)

	if ruser.Nickname != "Joram Wilander" {
		t.Fatal("Nickname did not update properly")
	}
	if ruser.FirstName != "Joram" {
		t.Fatal("FirstName did not update properly")
	}
	if ruser.LastName != "Wilander" {
		t.Fatal("LastName did not update properly")
	}
	if ruser.Position != "" {
		t.Fatal("Position did not update properly")
	}
	if ruser.Username != user.Username {
		t.Fatal("Username should not have updated")
	}

	_, resp = Client.PatchUser("junk", patch)
	CheckBadRequestStatus(t, resp)

	ruser.Id = model.NewId()
	_, resp = Client.PatchUser(model.NewId(), patch)
	CheckForbiddenStatus(t, resp)

	if r, err := Client.DoApiPut("/users/"+user.Id+"/patch", "garbage"); err == nil {
		t.Fatal("should have errored")
	} else {
		if r.StatusCode != http.StatusBadRequest {
			t.Log("actual: " + strconv.Itoa(r.StatusCode))
			t.Log("expected: " + strconv.Itoa(http.StatusBadRequest))
			t.Fatal("wrong status code")
		}
	}

	Client.Logout()
	_, resp = Client.PatchUser(user.Id, patch)
	CheckUnauthorizedStatus(t, resp)

	th.LoginBasic()
	_, resp = Client.PatchUser(user.Id, patch)
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.PatchUser(user.Id, patch)
	CheckNoError(t, resp)
}

func TestDeleteUser(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	Client := th.Client

	user := th.BasicUser
	th.LoginBasic()

	testUser := th.SystemAdminUser
	_, resp := Client.DeleteUser(testUser.Id)
	CheckForbiddenStatus(t, resp)

	Client.Logout()

	_, resp = Client.DeleteUser(user.Id)
	CheckUnauthorizedStatus(t, resp)

	Client.Login(testUser.Email, testUser.Password)

	user.Id = model.NewId()
	_, resp = Client.DeleteUser(user.Id)
	CheckNotFoundStatus(t, resp)

	user.Id = "junk"
	_, resp = Client.DeleteUser(user.Id)
	CheckBadRequestStatus(t, resp)

	_, resp = Client.DeleteUser(testUser.Id)
	CheckNoError(t, resp)

}

func TestUpdateUserRoles(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	Client := th.Client
	SystemAdminClient := th.SystemAdminClient

	_, resp := Client.UpdateUserRoles(th.SystemAdminUser.Id, model.ROLE_SYSTEM_USER.Id)
	CheckForbiddenStatus(t, resp)

	_, resp = SystemAdminClient.UpdateUserRoles(th.BasicUser.Id, model.ROLE_SYSTEM_USER.Id)
	CheckNoError(t, resp)

	_, resp = SystemAdminClient.UpdateUserRoles(th.BasicUser.Id, model.ROLE_SYSTEM_USER.Id+" "+model.ROLE_SYSTEM_ADMIN.Id)
	CheckNoError(t, resp)

	_, resp = SystemAdminClient.UpdateUserRoles(th.BasicUser.Id, "junk")
	CheckBadRequestStatus(t, resp)

	_, resp = SystemAdminClient.UpdateUserRoles("junk", model.ROLE_SYSTEM_USER.Id)
	CheckBadRequestStatus(t, resp)

	_, resp = SystemAdminClient.UpdateUserRoles(model.NewId(), model.ROLE_SYSTEM_USER.Id)
	CheckBadRequestStatus(t, resp)
}

func TestGetUsers(t *testing.T) {
	th := Setup().InitBasic()
	defer TearDown()
	Client := th.Client

	rusers, resp := Client.GetUsers(0, 60, "")
	CheckNoError(t, resp)
	for _, u := range rusers {
		CheckUserSanitization(t, u)
	}

	rusers, resp = Client.GetUsers(0, 60, resp.Etag)
	CheckEtag(t, rusers, resp)

	rusers, resp = Client.GetUsers(0, 1, "")
	CheckNoError(t, resp)
	if len(rusers) != 1 {
		t.Fatal("should be 1 per page")
	}

	rusers, resp = Client.GetUsers(1, 1, "")
	CheckNoError(t, resp)
	if len(rusers) != 1 {
		t.Fatal("should be 1 per page")
	}

	rusers, resp = Client.GetUsers(10000, 100, "")
	CheckNoError(t, resp)
	if len(rusers) != 0 {
		t.Fatal("should be no users")
	}

	// Check default params for page and per_page
	if _, err := Client.DoApiGet("/users", ""); err != nil {
		t.Fatal("should not have errored")
	}

	Client.Logout()
	_, resp = Client.GetUsers(0, 60, "")
	CheckUnauthorizedStatus(t, resp)
}

func TestGetUsersInTeam(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer TearDown()
	Client := th.Client
	teamId := th.BasicTeam.Id

	rusers, resp := Client.GetUsersInTeam(teamId, 0, 60, "")
	CheckNoError(t, resp)
	for _, u := range rusers {
		CheckUserSanitization(t, u)
	}

	rusers, resp = Client.GetUsersInTeam(teamId, 0, 60, resp.Etag)
	CheckEtag(t, rusers, resp)

	rusers, resp = Client.GetUsersInTeam(teamId, 0, 1, "")
	CheckNoError(t, resp)
	if len(rusers) != 1 {
		t.Fatal("should be 1 per page")
	}

	rusers, resp = Client.GetUsersInTeam(teamId, 1, 1, "")
	CheckNoError(t, resp)
	if len(rusers) != 1 {
		t.Fatal("should be 1 per page")
	}

	rusers, resp = Client.GetUsersInTeam(teamId, 10000, 100, "")
	CheckNoError(t, resp)
	if len(rusers) != 0 {
		t.Fatal("should be no users")
	}

	Client.Logout()
	_, resp = Client.GetUsersInTeam(teamId, 0, 60, "")
	CheckUnauthorizedStatus(t, resp)

	user := th.CreateUser()
	Client.Login(user.Email, user.Password)
	_, resp = Client.GetUsersInTeam(teamId, 0, 60, "")
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.GetUsersInTeam(teamId, 0, 60, "")
	CheckNoError(t, resp)
}

func TestGetUsersInChannel(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer TearDown()
	Client := th.Client
	channelId := th.BasicChannel.Id

	rusers, resp := Client.GetUsersInChannel(channelId, 0, 60, "")
	CheckNoError(t, resp)
	for _, u := range rusers {
		CheckUserSanitization(t, u)
	}

	rusers, resp = Client.GetUsersInChannel(channelId, 0, 1, "")
	CheckNoError(t, resp)
	if len(rusers) != 1 {
		t.Fatal("should be 1 per page")
	}

	rusers, resp = Client.GetUsersInChannel(channelId, 1, 1, "")
	CheckNoError(t, resp)
	if len(rusers) != 1 {
		t.Fatal("should be 1 per page")
	}

	rusers, resp = Client.GetUsersInChannel(channelId, 10000, 100, "")
	CheckNoError(t, resp)
	if len(rusers) != 0 {
		t.Fatal("should be no users")
	}

	Client.Logout()
	_, resp = Client.GetUsersInChannel(channelId, 0, 60, "")
	CheckUnauthorizedStatus(t, resp)

	user := th.CreateUser()
	Client.Login(user.Email, user.Password)
	_, resp = Client.GetUsersInChannel(channelId, 0, 60, "")
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.GetUsersInChannel(channelId, 0, 60, "")
	CheckNoError(t, resp)
}

func TestGetUsersNotInChannel(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer TearDown()
	Client := th.Client
	teamId := th.BasicTeam.Id
	channelId := th.BasicChannel.Id

	user := th.CreateUser()
	LinkUserToTeam(user, th.BasicTeam)

	rusers, resp := Client.GetUsersNotInChannel(teamId, channelId, 0, 60, "")
	CheckNoError(t, resp)
	for _, u := range rusers {
		CheckUserSanitization(t, u)
	}

	rusers, resp = Client.GetUsersNotInChannel(teamId, channelId, 0, 1, "")
	CheckNoError(t, resp)
	if len(rusers) != 1 {
		t.Log(len(rusers))
		t.Fatal("should be 1 per page")
	}

	rusers, resp = Client.GetUsersNotInChannel(teamId, channelId, 10000, 100, "")
	CheckNoError(t, resp)
	if len(rusers) != 0 {
		t.Fatal("should be no users")
	}

	Client.Logout()
	_, resp = Client.GetUsersNotInChannel(teamId, channelId, 0, 60, "")
	CheckUnauthorizedStatus(t, resp)

	Client.Login(user.Email, user.Password)
	_, resp = Client.GetUsersNotInChannel(teamId, channelId, 0, 60, "")
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.GetUsersNotInChannel(teamId, channelId, 0, 60, "")
	CheckNoError(t, resp)
}

func TestUpdateUserPassword(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer TearDown()
	Client := th.Client

	password := "newpassword1"
	pass, resp := Client.UpdateUserPassword(th.BasicUser.Id, th.BasicUser.Password, password)
	CheckNoError(t, resp)

	if !pass {
		t.Fatal("should have returned true")
	}

	_, resp = Client.UpdateUserPassword(th.BasicUser.Id, password, "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.UpdateUserPassword(th.BasicUser.Id, password, "junk")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.UpdateUserPassword("junk", password, password)
	CheckBadRequestStatus(t, resp)

	_, resp = Client.UpdateUserPassword(th.BasicUser.Id, "", password)
	CheckBadRequestStatus(t, resp)

	_, resp = Client.UpdateUserPassword(th.BasicUser.Id, "junk", password)
	CheckBadRequestStatus(t, resp)

	_, resp = Client.UpdateUserPassword(th.BasicUser.Id, password, th.BasicUser.Password)
	CheckNoError(t, resp)

	Client.Logout()
	_, resp = Client.UpdateUserPassword(th.BasicUser.Id, password, password)
	CheckUnauthorizedStatus(t, resp)

	th.LoginBasic2()
	_, resp = Client.UpdateUserPassword(th.BasicUser.Id, password, password)
	CheckForbiddenStatus(t, resp)

	th.LoginBasic()

	// Test lockout
	passwordAttempts := utils.Cfg.ServiceSettings.MaximumLoginAttempts
	defer func() {
		utils.Cfg.ServiceSettings.MaximumLoginAttempts = passwordAttempts
	}()
	utils.Cfg.ServiceSettings.MaximumLoginAttempts = 2

	// Fail twice
	_, resp = Client.UpdateUserPassword(th.BasicUser.Id, "badpwd", "newpwd")
	CheckBadRequestStatus(t, resp)
	_, resp = Client.UpdateUserPassword(th.BasicUser.Id, "badpwd", "newpwd")
	CheckBadRequestStatus(t, resp)

	// Should fail because account is locked out
	_, resp = Client.UpdateUserPassword(th.BasicUser.Id, th.BasicUser.Password, "newpwd")
	CheckErrorMessage(t, resp, "api.user.check_user_login_attempts.too_many.app_error")
	CheckForbiddenStatus(t, resp)

	// System admin can update another user's password
	adminSetPassword := "pwdsetbyadmin"
	pass, resp = th.SystemAdminClient.UpdateUserPassword(th.BasicUser.Id, "", adminSetPassword)
	CheckNoError(t, resp)

	if !pass {
		t.Fatal("should have returned true")
	}

	_, resp = Client.Login(th.BasicUser.Email, adminSetPassword)
	CheckNoError(t, resp)
}

func TestResetPassword(t *testing.T) {
	th := Setup().InitBasic()
	Client := th.Client

	Client.Logout()

	user := th.BasicUser

	// Delete all the messages before check the reset password
	utils.DeleteMailBox(user.Email)

	success, resp := Client.SendPasswordResetEmail(user.Email)
	CheckNoError(t, resp)
	if !success {
		t.Fatal("should have succeeded")
	}

	_, resp = Client.SendPasswordResetEmail("")
	CheckBadRequestStatus(t, resp)

	// Should not leak whether the email is attached to an account or not
	success, resp = Client.SendPasswordResetEmail("notreal@example.com")
	CheckNoError(t, resp)
	if !success {
		t.Fatal("should have succeeded")
	}

	var recovery *model.PasswordRecovery
	if result := <-app.Srv.Store.PasswordRecovery().Get(user.Id); result.Err != nil {
		t.Fatal(result.Err)
	} else {
		recovery = result.Data.(*model.PasswordRecovery)
	}

	// Check if the email was send to the right email address and the recovery key match
	if resultsMailbox, err := utils.GetMailBox(user.Email); err != nil && !strings.ContainsAny(resultsMailbox[0].To[0], user.Email) {
		t.Fatal("Wrong To recipient")
	} else {
		if resultsEmail, err := utils.GetMessageFromMailbox(user.Email, resultsMailbox[0].ID); err == nil {
			if !strings.Contains(resultsEmail.Body.Text, recovery.Code) {
				t.Log(resultsEmail.Body.Text)
				t.Log(recovery.Code)
				t.Fatal("Received wrong recovery code")
			}
		}
	}

	_, resp = Client.ResetPassword(recovery.Code, "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.ResetPassword(recovery.Code, "newp")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.ResetPassword("", "newpwd")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.ResetPassword("junk", "newpwd")
	CheckBadRequestStatus(t, resp)

	code := ""
	for i := 0; i < model.PASSWORD_RECOVERY_CODE_SIZE; i++ {
		code += "a"
	}

	_, resp = Client.ResetPassword(code, "newpwd")
	CheckBadRequestStatus(t, resp)

	success, resp = Client.ResetPassword(recovery.Code, "newpwd")
	CheckNoError(t, resp)
	if !success {
		t.Fatal("should have succeeded")
	}

	Client.Login(user.Email, "newpwd")
	Client.Logout()

	_, resp = Client.ResetPassword(recovery.Code, "newpwd")
	CheckBadRequestStatus(t, resp)

	authData := model.NewId()
	if result := <-app.Srv.Store.User().UpdateAuthData(user.Id, "random", &authData, "", true); result.Err != nil {
		t.Fatal(result.Err)
	}

	_, resp = Client.SendPasswordResetEmail(user.Email)
	CheckBadRequestStatus(t, resp)
}

func TestGetSessions(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer TearDown()
	Client := th.Client

	user := th.BasicUser

	Client.Login(user.Email, user.Password)

	sessions, resp := Client.GetSessions(user.Id, "")
	for _, session := range sessions {
		if session.UserId != user.Id {
			t.Fatal("user id does not match session user id")
		}
	}
	CheckNoError(t, resp)

	_, resp = Client.RevokeSession("junk", model.NewId())
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetSessions(th.BasicUser2.Id, "")
	CheckForbiddenStatus(t, resp)

	_, resp = Client.GetSessions(model.NewId(), "")
	CheckForbiddenStatus(t, resp)

	Client.Logout()
	_, resp = Client.GetSessions(th.BasicUser2.Id, "")
	CheckUnauthorizedStatus(t, resp)

	_, resp = th.SystemAdminClient.GetSessions(user.Id, "")
	CheckNoError(t, resp)

	_, resp = th.SystemAdminClient.GetSessions(th.BasicUser2.Id, "")
	CheckNoError(t, resp)

	_, resp = th.SystemAdminClient.GetSessions(model.NewId(), "")
	CheckNoError(t, resp)

}

func TestRevokeSessions(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer TearDown()
	Client := th.Client

	user := th.BasicUser
	Client.Login(user.Email, user.Password)
	sessions, _ := Client.GetSessions(user.Id, "")
	if len(sessions) == 0 {
		t.Fatal("sessions should exist")
	}
	for _, session := range sessions {
		if session.UserId != user.Id {
			t.Fatal("user id does not match session user id")
		}
	}
	session := sessions[0]

	_, resp := Client.RevokeSession(user.Id, model.NewId())
	CheckBadRequestStatus(t, resp)

	_, resp = Client.RevokeSession(th.BasicUser2.Id, model.NewId())
	CheckForbiddenStatus(t, resp)

	_, resp = Client.RevokeSession("junk", model.NewId())
	CheckBadRequestStatus(t, resp)

	status, resp := Client.RevokeSession(user.Id, session.Id)
	if status == false {
		t.Fatal("user session revoke unsuccessful")
	}
	CheckNoError(t, resp)

	Client.Logout()
	_, resp = Client.RevokeSession(user.Id, model.NewId())
	CheckUnauthorizedStatus(t, resp)

	_, resp = th.SystemAdminClient.RevokeSession(user.Id, model.NewId())
	CheckBadRequestStatus(t, resp)

	sessions, _ = th.SystemAdminClient.GetSessions(th.SystemAdminUser.Id, "")
	if len(sessions) == 0 {
		t.Fatal("sessions should exist")
	}
	for _, session := range sessions {
		if session.UserId != th.SystemAdminUser.Id {
			t.Fatal("user id does not match session user id")
		}
	}
	session = sessions[0]

	_, resp = th.SystemAdminClient.RevokeSession(th.SystemAdminUser.Id, session.Id)
	CheckNoError(t, resp)

}

func TestGetAudits(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer TearDown()
	Client := th.Client
	user := th.BasicUser

	audits, resp := Client.GetAudits(user.Id, 0, 100, "")
	for _, audit := range audits {
		if audit.UserId != user.Id {
			t.Fatal("user id does not match audit user id")
		}
	}
	CheckNoError(t, resp)

	_, resp = Client.GetAudits(th.BasicUser2.Id, 0, 100, "")
	CheckForbiddenStatus(t, resp)

	Client.Logout()
	_, resp = Client.GetAudits(user.Id, 0, 100, "")
	CheckUnauthorizedStatus(t, resp)

	_, resp = th.SystemAdminClient.GetAudits(user.Id, 0, 100, "")
	CheckNoError(t, resp)
}

func TestVerify(t *testing.T) {
	th := Setup().InitBasic()
	defer TearDown()
	Client := th.Client

	user := model.User{Email: GenerateTestEmail(), Nickname: "Darth Vader", Password: "hello1", Username: GenerateTestUsername(), Roles: model.ROLE_SYSTEM_ADMIN.Id + " " + model.ROLE_SYSTEM_USER.Id}

	ruser, resp := Client.CreateUser(&user)

	hashId := ruser.Id + utils.Cfg.EmailSettings.InviteSalt
	_, resp = Client.VerifyUserEmail(ruser.Id, hashId)
	CheckNoError(t, resp)

	hashId = ruser.Id + GenerateTestId()
	_, resp = Client.VerifyUserEmail(ruser.Id, hashId)
	CheckBadRequestStatus(t, resp)

	// Comment per request from Joram, he will investigate why it fail with a wrong status
	// hashId = ruser.Id+GenerateTestId()
	// _, resp = Client.VerifyUserEmail("", hashId)
	// CheckBadRequestStatus(t, resp)

	_, resp = Client.VerifyUserEmail(ruser.Id, "")
	CheckBadRequestStatus(t, resp)
}

func TestSetProfileImage(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer TearDown()
	Client := th.Client
	user := th.BasicUser

	data, err := readTestFile("test.png")
	if err != nil {
		t.Fatal(err)
	}

	ok, resp := Client.SetProfileImage(user.Id, data)
	if !ok {
		t.Fatal(resp.Error)
	}
	CheckNoError(t, resp)

	ok, resp = Client.SetProfileImage(model.NewId(), data)
	if ok {
		t.Fatal("Should return false, set profile image not allowed")
	}
	CheckForbiddenStatus(t, resp)

	// status code returns either forbidden or unauthorized
	// note: forbidden is set as default at Client4.SetProfileImage when request is terminated early by server
	Client.Logout()
	_, resp = Client.SetProfileImage(user.Id, data)
	if resp.StatusCode == http.StatusForbidden {
		CheckForbiddenStatus(t, resp)
	} else if resp.StatusCode == http.StatusUnauthorized {
		CheckUnauthorizedStatus(t, resp)
	} else {
		t.Fatal("Should have failed either forbidden or unauthorized")
	}

	_, resp = th.SystemAdminClient.SetProfileImage(user.Id, data)
	CheckNoError(t, resp)

	info := &model.FileInfo{Path: "users/" + user.Id + "/profile.png"}
	if err := cleanupTestFile(info); err != nil {
		t.Fatal(err)
	}
}
