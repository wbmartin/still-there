# Code Review: `src/web-app`

**Generated:** 2025-12-21

## Overall Assessment

The React web application is built on a very modern and well-chosen technology stack. It is well-structured, following common conventions, and demonstrates a good understanding of modern React patterns. The state management and API interaction layers are particularly strong. The primary areas for architectural improvement are in standardizing route protection and making data fetching more robust and scalable.

## Key Strengths (Best Practices)

-   **Modern Tooling:** The use of Vite, React 19, TypeScript, and Vitest provides a fast, efficient, and type-safe development environment that aligns with current industry best practices.
-   **Excellent Component Library:** The project is built on Mantine, a comprehensive and powerful component library. This provides a huge head start for building a consistent and feature-rich UI.
-   **Well-Structured State Management:** The use of Zustand with feature-sliced stores (`auth-store`, `app-user-store`) is a clean, scalable, and modern approach to managing global state. The stores correctly handle loading/error states and implement robust logic, such as clearing all data caches on logout.
-   **Centralized API Logic:** The `utils/api.ts` wrapper provides a single, clean point of entry for all API requests. It correctly abstracts away the logic for adding the CSRF token and setting cookie credentials, which is excellent for security and maintainability.
-   **Resilience and UX:** The use of a top-level `ErrorBoundary`, clear loading states (e.g., "Checking permissions..."), and user-friendly notifications for events like access denial shows a focus on creating a robust user experience.

## Primary Areas for Improvement

### 1. Route Protection Model

-   **Finding:** Route protection is handled imperatively by calling the `useAuthGuard` hook at the top of each page component that requires authentication or authorization.
-   **Impact:** While functional, this pattern is not as robust or explicit as a declarative, router-based approach. A developer could easily forget to add the hook to a new sensitive page, leaving it unprotected.
-   **Recommendation:** Refactor to a declarative model. Create a wrapper component (e.g., `<AuthGuard />` or `<ProtectedRoute />`) that contains the hook's logic and wraps sensitive routes directly within the `createBrowserRouter` configuration. This makes protection an explicit and non-negotiable part of the route definition itself and is a more common and scalable pattern.

### 2. Manual Data Fetching and Caching

-   **Finding:** Data fetching is handled manually inside `useEffect` hooks, and server state (like the list of users) is cached directly within Zustand stores. This involves manual management of loading states and "fetch-once" logic using `useRef`.
-   **Impact:** This pattern requires significant boilerplate and is less robust than dedicated libraries. It lacks features like automatic refetching on window focus, request de-duplication, and sophisticated cache invalidation, which have to be built by hand if needed.
-   **Recommendation:** Introduce a dedicated data-fetching library like **TanStack Query (React Query)**. It is the de-facto standard in the React ecosystem and would replace most `useEffect`-based fetching logic. It provides a robust cache out-of-the-box, simplifies state synchronization, and would be a major architectural improvement.

### 3. Client-Side Data Operations

-   **Finding:** The user list page fetches the *entire* list of users and then performs sorting and pagination on the client side.
-   **Impact:** This is a key architectural tradeoff. It provides a snappy UI for small datasets but does not scale to scenarios with thousands or millions of records, where the initial load would be unacceptably slow and memory-intensive.
-   **Recommendation:** This is an acceptable tradeoff for a "starter" application, but it should be recognized as a scalability limitation. The long-term solution is to implement server-side pagination, sorting, and filtering in the Go API and update the frontend to pass the necessary query parameters.

## Secondary Findings & Inconsistencies

-   **Hardcoded Configuration:** The API proxy URL in `vite.config.ts` is hardcoded (`http://localhost:8080`) and includes a comment that it requires manual editing. This should be loaded from an environment variable (`.env` file) for a better developer experience.
-   **Inconsistent Styling:** The project uses the powerful Mantine library but occasionally falls back to basic inline styles (`style={{...}}`). To fully leverage Mantine's theming and styling system, all styling should be done via Mantine props (`sx`, `style` functions) or layout components (`<Flex>`, `<Stack>`, etc.).
-   **Inconsistent Routing Definitions:** The router configuration in `App.tsx` mixes hardcoded path strings (e.g., `/mfa-setup`) with constants from `constants/routes.ts`. It should exclusively use the constants for better maintainability.
-   **Dead Code:** The `src/services` directory is empty, and the `use-examples-sync.ts` hook is disabled. These should be removed to reduce clutter and confusion.
