import { z } from "zod";

// The one schema in use. The edit pages validate inline today; a schema per
// feature is still an open call. Do not add speculative schemas here.
export const loginSchema = z.object({
  username: z.string().min(1, 'Username is required'),
  password: z.string().min(1, 'Password is required'),
});

export type LoginFormData = z.infer<typeof loginSchema>;
