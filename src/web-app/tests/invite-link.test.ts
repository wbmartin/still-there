import { describe, expect, it } from "vitest";
import {
  buildInviteLink,
  formatInviteExpiry,
  readInviteToken,
} from "../utils/invite-link";

// B035: the token rides in the URL fragment, never the path or query, so no
// server log or Referer sees it.

const token = "ab".repeat(32);

describe("invite link", () => {
  it("puts the token in the fragment of /invite", () => {
    expect(buildInviteLink("https://app.example", token)).toBe(
      `https://app.example/invite#${token}`,
    );
  });

  it("reads back the token it built", () => {
    const link = new URL(buildInviteLink("https://app.example", token));
    expect(readInviteToken(link.hash)).toBe(token);
    expect(readInviteToken(token)).toBe(token);
  });

  it.each([
    ["empty", ""],
    ["bare hash", "#"],
    ["truncated", `#${token.slice(0, 40)}`],
    ["upper case", `#${token.toUpperCase()}`],
    ["trailing junk", `#${token}x`],
  ])("refuses a %s fragment", (_label, hash) => {
    expect(readInviteToken(hash)).toBeNull();
  });

  it("formats an epoch-seconds expiry with its year", () => {
    expect(formatInviteExpiry(Date.UTC(2026, 9, 3, 12) / 1000)).toContain(
      "2026",
    );
  });
});
