// The invite link (B035): `<origin>/invite#<token>`.
//
// The token rides in the fragment. A browser never sends the fragment to a
// server, so the token is in no access log and no `Referer`; the invite page
// reads it from `location.hash` and posts it in a request body. The API
// returns the token once, when it is issued, and stores only its hash: a
// lost link is replaced (POST /userInviteReissue), never read back.

import { ROUTES } from "../constants/routes";

/** The token's one shape: 32 random bytes, hex (auth.NewInviteToken). */
const INVITE_TOKEN_PATTERN = /^[0-9a-f]{64}$/;

/** The link an admin copies and sends. */
export const buildInviteLink = (origin: string, token: string): string =>
  `${origin}${ROUTES.INVITE}#${token}`;

/** The token in `location.hash` (`#<token>`), or null when there is none of
 *  the right shape: a link truncated by a mail client, or an address typed
 *  by hand. The API would refuse it anyway; this lets the page say so
 *  without a round trip. */
export const readInviteToken = (hash: string): string | null => {
  const token = hash.startsWith("#") ? hash.slice(1) : hash;
  return INVITE_TOKEN_PATTERN.test(token) ? token : null;
};

/** "3 Oct 2026, 14:05" for an epoch-seconds expiry, in the viewer's zone. */
export const formatInviteExpiry = (expiresTs: number): string =>
  new Date(expiresTs * 1000).toLocaleString(undefined, {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
