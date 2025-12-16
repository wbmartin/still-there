export const ROUTES = {
  // User management
  APP_USER_LIST: 'appUserList',
  APP_USER_EDIT: 'appUserEdit', // Base path; append ?id or ?new

  // User group management
  APP_USER_GROUP_LIST: 'appUserGroupList',
  APP_USER_GROUP_EDIT: 'appUserGroupEdit', // Base path; append ?id

   // Other
   DASHBOARD: 'dashboard',
   EXAMPLE_LIST: 'exampleList',
   EXAMPLE_EDIT: 'example', // Base path; append ?exampleId
   LOGIN: 'login',
   USER_PREFERENCES: 'userPreferences',
} as const;