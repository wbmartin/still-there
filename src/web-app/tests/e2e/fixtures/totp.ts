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

// The API accepts a code once: a step at or below the last one it accepted
// for the user is a replay. freshTotpCode hands out the oldest step in the
// window (now's, one either side) newer than the last one used for the
// secret, skipping the step before now's near a boundary, and waits for the
// next step only when all three are spent.
const lastStepBySecret = new Map<string, number>();

export const freshTotpCode = async (secret: string): Promise<string> => {
  for (;;) {
    const nowSeconds = Date.now() / 1000;
    const current = Math.floor(nowSeconds / 30);
    for (const step of [current - 1, current, current + 1]) {
      if (step === current - 1 && nowSeconds % 30 > 27) continue;
      if (step > (lastStepBySecret.get(secret) ?? -1)) {
        lastStepBySecret.set(secret, step);
        return totpCode(secret, step * 30 * 1000);
      }
    }
    const waitMs = (30 - (nowSeconds % 30) + 0.5) * 1000;
    await new Promise((resolve) => setTimeout(resolve, waitMs));
  }
};
