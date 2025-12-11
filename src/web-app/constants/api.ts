export const API_ENDPOINTS = {
	// User
	USER_LIST: '/userList',
	USER: '/user',
	ADD_USER_LIST_TO_GROUP: '/addUserListToGroup',
	REMOVE_USER_LIST_FROM_GROUP: '/removeUserListFromGroup',

	// User groups
	USER_GROUP_LIST: '/userGroupList',
	USER_GROUP: '/userGroup',

	// Grants
	GRANT_LIST: '/grantList',

	// Examples
	EXAMPLE_LIST: '/exampleList',
	EXAMPLE: '/example',
} as const;