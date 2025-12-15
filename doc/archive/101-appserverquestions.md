# 101 - App Server Questions

Based on my analysis of the specs in `doc/100-Golang-app-server.md`, the existing `examples/go-app-srv` code, and the `examples/auth/auth.sql` schema, I have several questions that need clarification before proceeding with building the starter app in `src/api-srv`.

## Infrastructure and Dependencies

1. **shrmpl-srv Infrastructure**: The specs reference `https://github.com/wbmartin/shrmpl-srv-preview` for the infrastructure. Can you provide access to this repository or clarify what components (shrmpl-kv, shrmpl-log) are available? Do we need to implement our own Redis-like client for shrmpl-kv?

this is an open github repo, you should have access. there are example clients here: https://github.com/wbmartin/shrmpl-srv-preview/tree/main/examples/go


2. **Go Version**: The specs mention using Go 1.24, but the current examples use an older version. Should we update to 1.24, and are there any specific features from 1.24 that are required?

No specific requirements, but I'd like to stay current.  let's upgrade any older examples.

3. **External Dependencies**: The specs mention minimal dependencies but require shrmpl-kv and shrmpl-log. Are these services already running in your environment, or do we need to set them up as part of the project?

We will need to create a script in the bin folder to download the previously compiled executables in https://github.com/wbmartin/shrmpl-srv-preview/tree/main/dist/mac

## Database Schema Alignment

4. **Schema Evolution**: The existing `auth.sql` has a basic schema, but the specs require additional fields and tables (like the note table). Should we extend the existing schema or create a completely new one? How should we handle the transition?

Everything in the example folder is an example.  You should create a new sql/010-initialcreate.sql file to include that auth table and the pseudo code example that I started.

5. **User Table Structure**: The specs show `app_user` table with `user_group_membership_list` as `integer[]`, but the existing auth.sql doesn't have this field. Should we update the existing schema to match the specs?

you don't need to update the examples, just the newly created 010-initialcreate.sql file

6. **Security Context**: The specs emphasize multi-tenancy with `security_context` fields. In the existing examples, is there already a concept of security contexts implemented, or do we need to add this from scratch?
yes, almost every table will have a security context that must be checked  for every select, update, insert, delete by the app server.  the one table exception I can name is the app_user_grant because those apply to all security_contexts (tenants).  If it improves consistency, you can add a security_context and we'll set it to 0 to signal it's a system level setting.


## Authentication and Security

7. **JWT Claims Structure**: The specs define a detailed JWT claims structure with grants map by security_context. The existing auth handler creates a simple JWT. Do we need to completely replace the existing auth system?

use the more detailed jwt.

8. **Password Hashing**: The existing code uses bcrypt. The specs don't specify password hashing - should we continue using bcrypt or implement something else?

bcrypt.

9. **Session Management**: The specs mention CSRF tokens compared against session ID last 3 characters. How should sessions be managed in shrmpl-kv? Are sessions persistent or just for CSRF validation?

CSRF tokens won't interact with shrmpl-kv.  the X-CSRF header will be compared to the last 3 digits of the 'ses' claim in the jwt

## RBAC and Grants

10. **Grant IDs**: The specs define specific grant IDs (1001-1004 for notes, 8001 for password change, 9001-9003 for admin). Should these match exactly, or can we adjust them?

you can adjust them, but you should have a good reason.  the 1000 - 7000 series is for normal business logic.  the 8000 is for user level admin privileges and the 9000 series is for admin privileges.

11. **Grant Implementation**: The specs show a complex RBAC system with grants per security_context. The existing auth.sql has basic grants. Do we need to implement the full RBAC system as described?

yes full rbac.

12. **User Group Membership**: How should `user_group_membership_list` be populated and used? Is it a list of user_group_ids that get resolved to grants?

user_group_membership_list will be a list of integers corresponding to user_group.user_group_id.  to get the grants, you would pull in all of the user_grant.user_grant_id listed in the user_group.user_grant_list

## Application Structure

13. **Project Location**: Should the new starter app be built in `src/api-srv` as mentioned, or should it replace/improve the existing `examples/go-app-srv`?
new starter app be built in `src/api-srv`

14. **Configuration**: The specs show an .env file structure. Should we replace the existing JSON config in `loc-dev-config.json` with this .env approach?
examples/go-app-srv/loc-dev-config.json is an example.  You should create a new version in the top level etc folder.  I plan to delete the examples folder after we get this running.

15. **Logging**: The specs mention shrmpl-log TCP connection. Do we have a Go client for this, or do we need to implement one based on the Rust example mentioned?
check the https://github.com/wbmartin/shrmpl-srv-preview/blob/main/examples/go/main.go

## CRUD Operations

16. **Note Entity**: The specs define a note table with specific fields. Should this be the primary example entity, or do you want different entities?
this entity should cover everything.

17. **Optimistic Locking**: The specs detail complex optimistic locking with follow-up queries for 409/410 responses. Is this level of complexity required for the starter, or can we simplify?
keep it.

18. **Soft Deletes**: The specs require soft deletes with `delete_ts` and `delete_user_id`. Should all deletes be soft deletes in this system?

Yes, assum soft deletes.

## Development Workflow

19. **Testing**: The specs mention running lint and typecheck commands. What are the specific commands/tools you use (e.g., golangci-lint, go vet)?
You tell me if we need them.  this might be a vestige requirment that I should remove.

20. **Build Process**: How should the application be built and deployed? Are there specific Docker or deployment requirements?
I'm trying to do this without docker.  my plan is to just run this in go.  I've considered the scratch container to capture these, but feel like I can implement a simple distribution method.

21. **Migration Path**: Since you mentioned this is an evolution from the existing examples, how much of the existing code should be preserved vs. completely rewritten?

Almost all of it should be rewritten to accomodate the new requirements.  However keep in mind much of it was written with this same approach in mind.

## Priorities

22. **Scope**: Given the complexity of the full specs, should we start with a minimal viable version (just auth and one CRUD endpoint) and iterate, or implement the full system at once?

Let's start with auth and CRUD.  We'll tackle user management in a subsequent effort.

23. **Timeline**: What's the timeline for this project? Should we focus on getting a working starter quickly or invest time in the full production-ready features?</content>
<parameter name="filePath">doc/101-appserverquestions.md

I want full production-ready features.  we can take our time and do multiple iterations to get there.  don't add a bunch of bloat without checking.
