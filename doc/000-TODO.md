# React App

-several userAdmin page needs to map user groups  to group names, etc
- add error boundary
- lots of unknown users coming into the api server logs
- need to add MFA onboarding for users, reset for admin
- need to add password reset for admin, maybe a modal on the user edit page.
- Need to add password change for users

- need to make the admin pages only visible to admin users.  This may also mean we need to duplicate the list of grants in the react app so we can check them there.  We'll keep the endpoing to readGrantList, but we probably won't use it


- adding a user needs to hash the password and should require the admin to enter it twice to confirm. should also have a button to generate a good password, let you view the passwords as you type it. may be nice to let it copy the password to the clipboard.

- page number on the user list page should be part of state so a refresh or link takes to same place.


- user profile page with preferences page with items per page, modal user can change their own password.

- Check on all repo functions, they should be receving the security context from users jwt claims.

# API Server
- TOTP implementation, open to do in auth_handler.go also lots of impacts to the react app. Needs to be part of the login flow, and also need to have a way to onboard users with TOTP, and reset TOTP for users that lost their device. Admin should be able to reset TOTP for users.
- database change user_group -> app_user_group


# Starting prompt
make sure you look at my styles and quirks in the instructions. we are building a template CRUD app in Go and React. in the doc file is an 811 file that describes how to construct a new entity, we're currently building the first entity, "example", and documenting what's important to facilitate future new entity create. the last thing you did was create a critque in doc/811.
