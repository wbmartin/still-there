# schemas/

Zod schemas for forms. Today that is `loginSchema` (`pages/login.tsx`, via
`mantine-form-zod-resolver`). The entity edit pages validate inline; whether
they move to one schema per feature is PS0012's decision. Add a schema when a
form uses it, not before.
