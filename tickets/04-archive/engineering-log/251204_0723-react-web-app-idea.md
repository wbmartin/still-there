> Archived from `doc/archive/210-React-web-app-idea.md` on 2026-09-21 by PS0006; point-in-time, not maintained.

I want to create a new app starting with vite's create to update all the libraries and approaches I used in a previous project to turn it into a base for future projects. First step is to draft the technical spec.


I definitely want to use mantine for the UI library. I want to use RTK for client side state management.

There is a good baseline of what I want to create here: /Users/brandon/Code/ProtoStack-ReactGo/examples/snipper2

I want to start fresh though.  for this first round I want you to create
- Login
- Dashboard (will just have a count of examples)
- Example List Page
- Example Edit Page
- Admin Page
  - User adminstration
  - Group Administration
- Maintenance Page (will be empty)
- help Page - can have the excavator icon as a placeholder


It should have a left navigation bar that can expand out.  Instead of Pathways, I want you to call it "proto"

You'll need to create the components (tsx partials), features(slice and type), pages (tsx pages), services(server interactions).

Things we'll need to do in a second interation:
- Multi Factor Login
- Multi Factor onboarding

## Questions and Suggestions

### Suggestions for Improvement
- Update all dependencies to their latest stable versions (e.g., React 18, Mantine 8.x, RTK 2.x) for security and performance.
  - Yes please
- Adopt a consistent folder structure like the example: components/, pages/, services/, features/ (for RTK slices), hooks/.
  - In a followup you clarified you're talking about the hooks and features.  note some features may not need a type. 
- Implement RTK Query for all API interactions, following the example's snipper-api-svc pattern.
  - check this again, I think they are consistent.

- Add global error handling with Mantine's notifications and error boundaries.
  - good suggestion
- Include loading states and skeletons using Mantine's LoadingOverlay for better UX.
  - good suggestion
- Use Mantine's theme provider for consistent styling and potential dark mode.
 - good suggestion
- Add environment-based config (e.g., Vite env vars) for API URLs and feature flags.
  - good suggestion
- Ensure responsive design across all pages, leveraging Mantine's responsive utilities.
 - good suggestion
- Add basic E2E tests with Playwright or keep Vitest for unit/integration tests.
  - good suggestion
- Implement routing guards using the example's auth patterns (e.g., unauth-redirect hook).
- For the left nav, adapt the example's left-nav-bar.tsx but rename "Pathways" to "Proto" and make it expandable.

### Questions
- What authentication mechanism should we use (e.g., JWT, session-based)? How will it integrate with the existing Go API server?
  - this should be obvious with all the work we did on JWT in the go app.  These two are intended to be complimentary so you need to review the go app to see how this fits together
- For the Example pages, what fields and functionality should the list and edit forms include? (Referencing the example_model in the API.)
  - normal input fields, text boxes, this is an example page to showcase how it would work.
- What specific features are needed for User Administration and Group Administration in the Admin page?
  - Add User
  - Delete(soft) User
  - Add Group to User
  - Remove Group From User
  - Add user_grant to Group
  - remove user_grant from Group
  - list members of group
- How should the security_context from the database be handled in the frontend (e.g., tenant isolation)?
  - just like it is in the example snipper 2 app.
- Any specific design mockups or branding guidelines beyond the "proto" name and excavator icon?
  - there are some examples in the examples/assets folder.  Just use these as placholders, don't design around them
- Should we include internationalization (i18n) support from the start?
  - no
- For MFA in future iterations, any preferred library or approach (e.g., TOTP, SMS)?
  - TOTP
- How to handle routing guards for authenticated vs. public pages?
   - I think the "  unauthRedirect(location.pathname + location.search);" around line 59 of examples/snipper2/src/pages/pathway-batch-list-page.tsx

## Additional Questions and Clarifications

- For the Example Edit form, should the int/float arrays be edited as comma-separated strings or with dynamic add/remove inputs?
- How should the security_context be handled in the frontend (e.g., stored in state, sent in API requests)? Reviewing the example, it seems included in auth response but not explicitly used in components—confirm if tenant filtering is needed in UI.
- For branding, use examples/assets/proto-basic-logo-rectangle.jpg and proto-basic-logo-square.jpg as placeholders in the nav and login.
- Auth integration: Frontend will use RTK Query mutations for login (POST /auth), with JWT set in httpOnly cookie by server; subsequent API calls should include the cookie automatically. For MFA, prepare for temp token flow in future iteration.

## Technical Specification

### Overview
This document specifies the technical implementation of a clean, modern React web application using Vite, TypeScript, Mantine UI, RTK Query for API interactions, and React Router 7 for routing. The app serves as a base for future projects, integrating with the existing Go API server for authentication, user management, and CRUD operations on examples.

### Architecture
- **Framework**: React 18 with TypeScript
- **Build Tool**: Vite
- **UI Library**: Mantine 8.x with theme provider and responsive design
- **State Management**: Redux Toolkit (RTK) for local state, RTK Query for API state
- **Routing**: React Router 7 with protected routes using custom hooks
- **Styling**: Mantine components with CSS modules for custom styles
- **Testing**: Vitest for unit/integration tests, Playwright for E2E
- **Linting/Formatting**: ESLint with TypeScript rules, Prettier

### Project Structure
```
src/
├── components/          # Reusable UI components (e.g., forms, tables)
├── pages/               # Page components (Login, Dashboard, etc.)
├── features/            # RTK slices and types (auth, example, etc.)
├── services/            # API service functions and RTK Query base
├── hooks/               # Custom React hooks (auth guards, etc.)
├── utils/               # Utility functions
├── types/               # Shared TypeScript types
├── App.tsx              # Main app component with routing
├── main.tsx             # Entry point
├── store.ts             # Redux store configuration
└── index.css            # Global styles
```

### Authentication & Authorization
- **Login Flow**: Username/password POST to /auth; JWT stored in httpOnly cookie
- **MFA Preparation**: Support temp token flow for TOTP in future iteration
- **Session Management**: RTK slice tracks auth state (isAuthenticated, user, grants)
- **Routing Guards**: `unauthRedirect` hook redirects unauthenticated users to /login
- **Grants**: User permissions stored in state for UI conditional rendering

### Pages & Features
- **Login**: Form with username/password, Mantine notifications for errors
- **Dashboard**: Display example count from API
- **Example List**: Table with pagination, sorting, search; actions for edit/delete
- **Example Edit**: Form with fields for example_text (TextInput), example_int (NumberInput), example_float (NumberInput), arrays (Textarea for comma-separated values)
- **Admin**:
  - User Admin: List users, add user, soft delete user
  - Group Admin: Add/remove users from groups, manage group grants
- **Maintenance**: Empty placeholder page
- **Help**: Placeholder with excavator icon

### API Integration
- **Base API**: RTK Query setup with base URL from env, automatic cookie inclusion
- **Endpoints**: Mutations for auth, queries/mutations for examples, users, groups
- **Error Handling**: Global error boundaries, Mantine notifications for API errors
- **Loading States**: Mantine LoadingOverlay for async operations

### UI/UX
- **Navigation**: Expandable left sidebar with "Proto" branding, using proto-basic-logo-rectangle.jpg
- **Responsive**: Mantine responsive utilities for mobile compatibility
- **Theme**: Mantine theme provider with potential dark mode
- **Forms**: Mantine form hooks with validation
- **Tables**: Mantine Table with sorting/pagination

### Security
- **CSRF**: Token from auth response stored in localStorage, sent in headers
- **Security Context**: Included in API requests for tenant isolation (per example app)
- **Input Validation**: Client-side with Mantine forms, server-side via API

### Development & Deployment
- **Environment Config**: Vite env vars for API base URL, feature flags
- **Testing**: Unit tests for components/slices, E2E for critical flows
- **Build**: Optimized production build with Vite
- **Dependencies**: Latest stable versions, updated from example baseline

### Future Iterations
- MFA Login: TOTP verification flow
- MFA Onboarding: Setup TOTP for users
