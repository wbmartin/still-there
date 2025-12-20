# React App
- MFA
  - investigate passcode as second factor
  -  QR code is too big
  
- Need to update all the 4 digit log codes to <12 digit codes to avoid collisions and increase clarity.


- react forms complaint - maybe should be using a library.
- user profile page
  - preferences 
  - upload avatar button.  
  
  
- Non Sensitive file storing -There will be an nginx server running on the server and we should use it to server static non sensitive files

- Check on all repo functions, they should be receving the security context from users jwt claims.
- page number on the user list page should be part of state so a refresh or link takes to same place.

- move the database connection string, JWT, and the TOTP code to a shrmple vault.







# API Server
- account lockout after failed login attempts, need to track failed login attempts and lock the account for a period of time after too many failed attempts.
- It is currently possible for someone with user update rights to add a group ID outside their context and gain access to another context.  The groups need to be checked to make sure they are within the users security context.
- need to compare https with http in the load test.


# shrmpl-chk and shrmpl-mon
need to add rust web server that listens to a port (may not need to be https) and receives health information, only in the query string. reads a cronitor type schedule and makes slack calls when things go wrong. needs to get slack webhook url from the shrmpl key server
also need to add a binary that checks website health (probably just status) and records the checkins to shrmpl-mon.

# you need to make a smug mug clone

# Starting prompt
make sure you look at my styles and quirks in the instructions. we are building a template/initial scaffold CRUD app in Go and React. in the doc file is an 810 file that describes how to construct a new entity, we're currently building the first entity, "example", and documenting what's important to facilitate future new entity creation for apps built on this scaffolding.
