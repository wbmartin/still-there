// Password utility functions
export function generateStrongPassword(): string {
  const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*";
  // Rejection sampling over a CSPRNG: skip values that would bias the modulo
  const limit = 256 - (256 % chars.length);
  const bytes = new Uint8Array(32);
  let password = "";
  while (password.length < 12) {
    crypto.getRandomValues(bytes);
    for (const b of bytes) {
      if (b < limit && password.length < 12) {
        password += chars.charAt(b % chars.length);
      }
    }
  }
  return password;
}
