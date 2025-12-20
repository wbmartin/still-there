# React App
- MFA
  - MFAreset for admin. 
  - investigate passcode as second factor
  
- Need to update all the 4 digit log codes to <12 digit codes to avoid collisions and increase clarity.


- react forms complaint - maybe should be using a library.
- user profile page
  - preferences 
  - upload avatar button.  
  
  
- Non Sensitive file storing -There will be an nginx server running on the server and we should use it to server static non sensitive files

- Check on all repo functions, they should be receving the security context from users jwt claims.
- page number on the user list page should be part of state so a refresh or link takes to same place.

- move the database connection string, JWT, and the TOTP code to a shrmple vault.



We need to create a user self-create function and MFA onboarding flow.  on the login page there should be a link to create a new user.  the user will enter their email address and desired password (twice and the web-app will need to verify it matches).  In the future we will send a confirmation email so leave a TODO for that in the api-server.  The dialog should check with the server to see if a username is available.  We can just add this as an error code from the user self create endpoint. (In the future we may do async to check if it's available before submitting.)   

Then we need to add the MFA onboarding flow.  After the user is created in the self-create flow, they should be taken to a page that shows them a QR code and a secret key for TOTP.  They should be required to enter a current TOTP code to confirm they have set it up correctly before being allowed to proceed.  
- We will need to add an endpoint to the api-server to support this as well.  
- The user should then be taken to their dashboard.  
- We will need to add a TOTP seed token to the app_user table.  
- We will need endpoints to generate the QR code.  

Note:There is existing Auth2 code that was started, but may have been deleted. If the user is created from the user admin panel they may have the mfa_exempt set.  Users that have mfa_exempt set should not be required to do the MFA onboarding flow.  We will need to add a way for admin to reset a users MFA settings so they can go through the onboarding flow again if they lose their device.  This should probably be a button on the user edit page in the admin panel.   We should also add a button to generate a strong password that the user can use if they don't want to come up with one themselves.  The generated password should be displayed in a way that the user can easily copy it to their clipboard.  If a user was created in the admin panel and doesn't have MFA exempt set they should be taken to the MFA onboarding flow the first time they log in as well.  There will need to be a Security Context Page as well since we'll need to let the security_context admin specify which groups will be assigned for self create.  We should create a default "New User" group that only has the logon page.  The admin will need assign privil  eges to this group as needed or add them to more groups to get their access levels set up.  help me plan.






# API Server
- TOTP implementation, open to do in auth_handler.go also lots of impacts to the react app. Needs to be part of the login flow, and also need to have a way to onboard users with TOTP, and reset TOTP for users that lost their device. Admin should be able to reset TOTP for users.
- account lockout after failed login attempts, need to track failed login attempts and lock the account for a period of time after too many failed attempts.
- It is currently possible for someone with user update rights to add a group ID outside their context and gain access to another context.  The groups need to be checked to make sure they are within the users security context.
- need to compare https with http in the load test.
- need to toggle https in the config file.


# shrmpl-chk and shrmpl-mon
need to add rust web server that listens to a port (may not need to be https) and receives health information, only in the query string. reads a cronitor type schedule and makes slack calls when things go wrong. needs to get slack webhook url from the shrmpl key server
also need to add a binary that checks website health (probably just status) and records the checkins to shrmpl-mon.

# you need to make a smug mug clone




# Starting prompt
make sure you look at my styles and quirks in the instructions. we are building a template/initial scaffold CRUD app in Go and React. in the doc file is an 810 file that describes how to construct a new entity, we're currently building the first entity, "example", and documenting what's important to facilitate future new entity creation for apps built on this scaffolding.
