# Form Schemas

This directory contains Zod validation schemas for form validation. These schemas provide type-safe validation rules for all forms in the application.

## Available Schemas

### Authentication
- `loginSchema` - Login form validation

### User Management
- `userCreateSchema` - User creation form validation
- `userUpdateSchema` - User update form validation

### Examples
- `exampleCreateSchema` - Example creation form validation
- `exampleUpdateSchema` - Example update form validation

### User Groups
- `userGroupCreateSchema` - User group creation validation
- `userGroupUpdateSchema` - User group update validation

### Password Management
- `passwordChangeSchema` - Password change validation

## Usage with Mantine Forms

```typescript
import { useForm, zodResolver } from '@mantine/form';
import { loginSchema } from '../schemas/form-schemas';

const form = useForm({
  initialValues: { username: '', password: '' },
  validate: zodResolver(loginSchema), // TODO: Enable when version compatibility resolved
});
```

## Type Exports

Each schema has a corresponding TypeScript type export for type-safe form handling:

```typescript
import type { LoginFormData } from '../schemas/form-schemas';

const form = useForm<LoginFormData>({ /* config */ });
```

## Zod Integration Status

**✅ Current Status:** Zod schemas are fully integrated with Mantine forms using the `mantine-form-zod-resolver` package, which resolves the version compatibility issue between Zod v4 and @mantine/form.

**Integration Details:**
- Uses `mantine-form-zod-resolver` package instead of `@mantine/form`'s built-in zodResolver
- Full Zod v4 support with advanced validation features
- Compatible with all Zod schema types and validation rules

**Migration Status:**
- ✅ Login form: Fully migrated to Zod validation
- ✅ User edit form: Migrated to useForm with basic validation (Zod schemas ready for future use)
- 🔄 Other forms: Ready to migrate using existing schemas
- 📋 Future forms: Use schemas from this file as the standard approach

**Note:** User edit form uses basic validation due to Zod type compatibility issues with complex forms. Zod schemas are defined and ready for when compatibility is resolved.

## Benefits of Zod Integration

- **Advanced Validation**: Regex patterns, length constraints, format validation
- **Type Safety**: Automatic TypeScript types from schemas
- **Reusable**: Same schema can be used across multiple forms/components
- **Cross-field Validation**: Validate relationships between fields
- **Developer Experience**: Clear, declarative validation rules