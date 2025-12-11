# Critique of User and User Group Implementation vs 810 Doc

## Introduction

This document critiques the implementation of user and user group entities against the guidelines in doc/810-create-new-entity-type-notes.md. It identifies deviations and areas of compliance based on a comparison of the actual code to the documented patterns.

## Deviations

### Naming Conventions
- **Single-entity methods**: Use "Get" (e.g., `GetUserByUserName` in `user_repository.go`) instead of "Read" as suggested for both list and single operations in the doc. The doc states "Use 'Read' for list retrieval (e.g., `ReadUserList`) and single-entity retrieval (e.g., `ReadUser`)", but implementations use "Get" for single entities.

### Stores
- **user-groups-store**: Includes `readGrantList`, an extra function not mentioned in the doc's store guidelines.
- **Delete operations**: Use specific query param names like `?userId=` and `?userGroupId=` instead of the generic `?entityId=` suggested in the doc.

### Pages
- **user-group-list-page**: Lacks pagination, unlike `user-list-page` which includes pagination. The doc suggests pagination for list pages ("Table with pagination, add/edit/delete buttons").

### Handlers and Repositories
- **user_handler**: Includes `ReadGrantList`, an extra method not in the doc.
- **user_repository**: Includes `GetUserGrantList`, an extra method not in the doc.

### Other
- **API endpoints**: Use `/userList`, `/userGroupList` etc., compliant, but delete uses specific params as noted.

## Compliant Areas

- **CRUD methods**: All required Create, Read, Update, Delete methods are present in handlers and repositories.
- **Types**: Separate `User`/`ApiUserModel` and `UserGroup`/`ApiUserGroupModel` interfaces, matching the doc.
- **Query params**: Edit pages use query params for IDs, compliant with the critical note in the doc.
- **Grant naming**: Updated to `GrantUserCreate`, `GrantUserBlock`, etc., following CRUD verbs as per doc.
- **Routes**: Use query params, no path params, compliant.
- **Store naming**: Singular (e.g., `useUserStore`), compliant.
- **Function comments**: Present (e.g., `//============================^FUNCTION LINE ^============================`), compliant.

## Conclusion

The user and user group implementations are largely compliant with the 810 doc, with deviations primarily in extra methods for grants, inconsistent pagination, and minor naming variations (e.g., "Get" vs "Read" for single entities). The core CRUD functionality, types, and routing follow the patterns well. Future entities should consider the extras as optional enhancements and ensure consistent pagination.</content>
<parameter name="filePath">doc/811-create-critique.md