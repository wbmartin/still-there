import { z } from 'zod';

// Login form schema
export const loginSchema = z.object({
  username: z.string().min(1, 'Username is required'),
  password: z.string().min(1, 'Password is required'),
});

// User form schemas
export const userCreateSchema = z.object({
  userName: z.string()
    .min(3, 'Username must be at least 3 characters')
    .max(50, 'Username must be less than 50 characters')
    .regex(/^[a-zA-Z0-9_-]+$/, 'Username contains invalid characters'),
  firstName: z.string().min(1, 'First name is required'),
  lastName: z.string().min(1, 'Last name is required'),
  email: z.string().email('Invalid email address'),
  appUserGroupMembershipList: z.array(z.number()),
  mfaExempt: z.boolean(),
  pwHash: z.string(),
});

export const userUpdateSchema = z.object({
  userName: z.string()
    .min(3, 'Username must be at least 3 characters')
    .max(50, 'Username must be less than 50 characters')
    .regex(/^[a-zA-Z0-9_-]+$/, 'Username contains invalid characters'),
  firstName: z.string().min(1, 'First name is required'),
  lastName: z.string().min(1, 'Last name is required'),
  email: z.string().email('Invalid email address'),
  appUserGroupMembershipList: z.array(z.number()),
  mfaExempt: z.boolean(),
  pwHash: z.string(),
  updateTs: z.string(), // Form uses string, API converts to number
});

// Example form schemas
export const exampleCreateSchema = z.object({
  exampleText: z.string().min(1, 'Example text is required'),
  exampleInt: z.number(),
  exampleFloat: z.number(),
});

export const exampleUpdateSchema = z.object({
  exampleId: z.number(),
  exampleText: z.string().min(1, 'Example text is required'),
  exampleInt: z.number(),
  exampleFloat: z.number(),
  updateTs: z.number(),
});

// User group schemas
export const userGroupCreateSchema = z.object({
  groupName: z.string().min(1, 'Group name is required'),
});

export const userGroupUpdateSchema = z.object({
  appUserGroupID: z.number(),
  groupName: z.string().min(1, 'Group name is required'),
  updateTs: z.number(),
});

// Password change schema
export const passwordChangeSchema = z.object({
  oldPassword: z.string().min(1, 'Current password is required'),
  newPassword: z.string()
    .min(8, 'Password must be at least 8 characters')
    .regex(/^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/,
      'Password must contain at least one uppercase letter, one lowercase letter, and one number'),
});

// Type exports for use with useForm
export type LoginFormData = z.infer<typeof loginSchema>;
export type UserCreateFormData = z.infer<typeof userCreateSchema>;
export type UserUpdateFormData = z.infer<typeof userUpdateSchema>;
export type ExampleCreateFormData = z.infer<typeof exampleCreateSchema>;
export type ExampleUpdateFormData = z.infer<typeof exampleUpdateSchema>;
export type UserGroupCreateFormData = z.infer<typeof userGroupCreateSchema>;
export type UserGroupUpdateFormData = z.infer<typeof userGroupUpdateSchema>;
export type PasswordChangeFormData = z.infer<typeof passwordChangeSchema>;