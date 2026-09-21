import { createHmac } from "node:crypto";

// RFC 6238 TOTP (SHA-1, 6 digits, 30 s) from a base32 secret — the same
// arithmetic an authenticator app does, so the e2e suite can complete the
// second factor without a dependency.

const BASE32 = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";

const base32Decode = (secret: string): Buffer => {
  const clean = secret.toUpperCase().replace(/=+$/, "").replace(/\s/g, "");
  let bits = "";
  for (const char of clean) {
    const index = BASE32.indexOf(char);
    if (index < 0) throw new Error(`not base32: ${char}`);
    bits += index.toString(2).padStart(5, "0");
  }
  const bytes: number[] = [];
  for (let i = 0; i + 8 <= bits.length; i += 8) {
    bytes.push(parseInt(bits.slice(i, i + 8), 2));
  }
  return Buffer.from(bytes);
};

export const totpCode = (secret: string, now = Date.now()): string => {
  const counter = Buffer.alloc(8);
  counter.writeBigUInt64BE(BigInt(Math.floor(now / 1000 / 30)));
  const digest = createHmac("sha1", base32Decode(secret))
    .update(counter)
    .digest();
  const offset = digest[digest.length - 1] & 0x0f;
  const code =
    ((digest[offset] & 0x7f) << 24) |
    (digest[offset + 1] << 16) |
    (digest[offset + 2] << 8) |
    digest[offset + 3];
  return String(code % 1_000_000).padStart(6, "0");
};
