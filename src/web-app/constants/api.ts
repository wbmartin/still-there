// Every API path the web app calls. Mirrors the routes in
// src/api-srv/cmd/server/main.go by hand -- the two lists are maintained
// separately on purpose (no generator, no build-time coupling); when a route
// is added there, add it here. apiJson takes one of these.
// tests/contract.test.ts fails if a path here is not registered there.
export const API_ENDPOINTS = {
  // Session
  AUTH: "/auth",
  REFRESH: "/refresh",
  LOGOFF: "/logoff",
  CHANGE_SECURITY_CONTEXT: "/changeSecurityContext",
  CHANGE_OWN_PASSWORD: "/changeOwnPassword",
  ME: "/me",

  // MFA
  MFA_SETUP: "/mfaSetup",
  MFA_VERIFY: "/mfaVerify",
  ADMIN_RESET_USER_MFA: "/admin/resetUserMFA",

  // Block / unblock a login (handler/user_block_handler.go)
  ADMIN_BLOCK_USER: "/admin/blockUser",
  ADMIN_UNBLOCK_USER: "/admin/unblockUser",

  // Users
  USER_LIST: "/userList",
  USER: "/user",
  USER_SELF_CREATE: "/userSelfCreate",
  CHANGE_OTHER_USER_PASSWORD: "/changeOtherUserPassword",
  ADD_USER_LIST_TO_GROUP: "/addUserListToGroup",
  REMOVE_USER_LIST_FROM_GROUP: "/removeUserListFromGroup",

  // Invite links (handler/user_invite_handler.go): admin half, then the
  // public half the invited person's browser calls with no session
  USER_INVITE: "/userInvite",
  USER_INVITE_REISSUE: "/userInviteReissue",
  USER_INVITE_REVOKE: "/userInviteRevoke",
  INVITE_CHECK: "/inviteCheck",
  INVITE_REDEEM: "/inviteRedeem",

  // User groups
  APP_USER_GROUP_LIST: "/appUserGroupList",
  APP_USER_GROUP: "/appUserGroup",

  // Grants
  GRANT_LIST: "/grantList",

  // Examples
  EXAMPLE_LIST: "/exampleList",
  EXAMPLE: "/example",

  // In-app issue reports (handler/issue_report_handler.go;
  // doc/150-issue-report-module.md)
  ISSUE_REPORT: "/issueReport",
  ISSUE_REPORT_LIST: "/issueReportList",
  ISSUE_REPORT_MESSAGE: "/issueReportMessage",
  ISSUE_REPORT_ACTION: "/issueReportAction",
  ISSUE_REPORT_ATTACHMENT: "/issueReportAttachment",

  // In-app help usage (handler/help_handler.go; doc/151-help-module.md)
  HELP_EVENT: "/helpEvent",
  HELP_INSIGHTS: "/helpInsights",
} as const;

export type ApiEndpoint = (typeof API_ENDPOINTS)[keyof typeof API_ENDPOINTS];
