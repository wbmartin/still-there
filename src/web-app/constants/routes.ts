export const ROUTES = {
  // User management
  APP_USER_LIST: 'appUserList',
  APP_USER_EDIT: 'appUserEdit', // Base path; append ?id or ?new

  // User group management
  USER_GROUP_LIST: 'userGroupList',
  USER_GROUP_EDIT: 'userGroupEdit', // Base path; append ?id

   // Other
   DASHBOARD: 'dashboard',
   EXAMPLE_LIST: 'exampleList',
   EXAMPLE_EDIT: 'example', // Base path; append ?exampleId
   LOGIN: 'login',
} as const;