// Every API path the web app calls. Mirrors the routes in
// src/api-srv/cmd/server/main.go by hand -- the two lists are maintained
// separately on purpose (no generator, no build-time coupling); when a route
// is added there, add it here. apiRequest/apiJson take one of these.
export const API_ENDPOINTS = {
  // Session
  AUTH: "/auth",
  REFRESH: "/refresh",
  LOGOFF: "/logoff",
  CHANGE_SECURITY_CONTEXT: "/changeSecurityContext",
  CHANGE_OWN_PASSWORD: "/changeOwnPassword",

  // MFA
  MFA_SETUP: "/mfaSetup",
  MFA_VERIFY: "/mfaVerify",
  ADMIN_RESET_USER_MFA: "/admin/resetUserMFA",

  // Users
  USER_LIST: "/userList",
  USER: "/user",
  USER_SELF_CREATE: "/userSelfCreate",
  CHANGE_OTHER_USER_PASSWORD: "/changeOtherUserPassword",
  ADD_USER_LIST_TO_GROUP: "/addUserListToGroup",
  REMOVE_USER_LIST_FROM_GROUP: "/removeUserListFromGroup",

  // User groups
  APP_USER_GROUP_LIST: "/appUserGroupList",
  APP_USER_GROUP: "/appUserGroup",

  // Grants
  GRANT_LIST: "/grantList",

  // Examples
  EXAMPLE_LIST: "/exampleList",
  EXAMPLE: "/example",
} as const;

export type ApiEndpoint = (typeof API_ENDPOINTS)[keyof typeof API_ENDPOINTS];
