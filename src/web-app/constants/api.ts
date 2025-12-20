export const API_ENDPOINTS = {
	// User
	USER_LIST: '/userList',
	USER: '/user',
	USER_SELF_CREATE: '/userSelfCreate',
	MFA_SETUP: '/mfaSetup',
	MFA_VERIFY: '/mfaVerify',
	CHANGE_OWN_PASSWORD: '/changeOwnPassword',
	CHANGE_OTHER_USER_PASSWORD: '/changeOtherUserPassword',
	ADD_USER_LIST_TO_GROUP: '/addUserListToGroup',
	REMOVE_USER_LIST_FROM_GROUP: '/removeUserListFromGroup',

	// User groups
	APP_USER_GROUP_LIST: '/appUserGroupList',
	APP_USER_GROUP: '/appUserGroup',

	// Grants
	GRANT_LIST: '/grantList',

	// Examples
	EXAMPLE_LIST: '/exampleList',
	EXAMPLE: '/example',
} as const;