# React App
- when the user changes context, all cache has to be invalidted
- table sorting
- add error boundary
- lots of unknown users coming into the api server logs
- need to add MFA onboarding for users, 
- MFAreset for admin. 
- redo the auth2 handler opencode deleted
- need to add password reset for admin, maybe a modal on the user edit page.
- Need to add password change for users


- adding a user needs to hash the password and should require the admin to enter it twice to confirm. 
- add a button to generate a good password, let you view the passwords as you type it. may be nice to let it copy the password to the clipboard.

- page number on the user list page should be part of state so a refresh or link takes to same place.

- react forms complaint - maybe should be using a library.
- user profile page
  - preferences 
  - modal user can change their own password.  
  - upload avatar button.  
  
  
  - Non Sensitive file storing -There will be an nginx server running on the server and we should use it to server static non sensitive files

- Check on all repo functions, they should be receving the security context from users jwt claims.

# API Server
- TOTP implementation, open to do in auth_handler.go also lots of impacts to the react app. Needs to be part of the login flow, and also need to have a way to onboard users with TOTP, and reset TOTP for users that lost their device. Admin should be able to reset TOTP for users.
- database change user_group -> app_user_group
- account lockout after failed login attempts, need to track failed login attempts and lock the account for a period of time after too many failed attempts.


# database
- table user_group needs to be renamed to app_user_group and all references to it in code and sql need to be updated.

# passkeys
You need to make a passkey login system for the api server and react app.

# shrmpl-chk and shrmpl-mon
need to add rust web server that listens to a port (may not need to be https) and receives health information, only in the query string. reads a cronitor type schedule and makes slack calls when things go wrong. needs to get slack webhook url from the shrmpl key server
also need to add a binary that checks website health (probably just status) and records the checkins to shrmpl-mon.

# you need to make a smug mug clone









# Starting prompt
make sure you look at my styles and quirks in the instructions. we are building a template CRUD app in Go and React. in the doc file is an 811 file that describes how to construct a new entity, we're currently building the first entity, "example", and documenting what's important to facilitate future new entity create.
