# schemas/

Zod schemas for forms, used through `mantine-form-zod-resolver`:
`loginSchema` (`pages/login.tsx`) and the device monitoring forms (device,
band, contact, tenant setting, simulated text), each mirroring its Go
`*Problem` check. The template's entity edit pages still validate inline.
Add a schema when a form uses it, not before.
