// Absolute paths, so a consumer never has to remember the leading slash.
// Edit pages take their id as a query string: `${ROUTES.EXAMPLE_EDIT}?exampleId=N`
// (or `=new`).
export const ROUTES = {
  // User management
  APP_USER_LIST: "/appUserList",
  APP_USER_EDIT: "/appUserEdit",

  // User group management
  APP_USER_GROUP_LIST: "/appUserGroupList",
  APP_USER_GROUP_EDIT: "/appUserGroupEdit",

  // Other
  DASHBOARD: "/dashboard",
  EXAMPLE_LIST: "/exampleList",
  EXAMPLE_EDIT: "/example",
  LOGIN: "/login",
  USER_PREFERENCES: "/userPreferences",
  MFA_SETUP: "/mfa-setup",
  // An enrolled user's second step: the same page, no QR code and no call
  // to /mfaSetup.
  MFA_VERIFY: "/mfa-verify",
  // An invite link's landing page: `/invite#<token>` (utils/invite-link.ts).
  INVITE: "/invite",
} as const;
