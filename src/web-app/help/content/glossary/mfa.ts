import type { HelpTerm } from "../../help-types";

export default {
  id: "mfa",
  term: "MFA",
  short:
    "Multi-factor authentication: after your password, a six-digit code " +
    "from an authenticator app on your phone.",
  body: `On your first sign-in you scan a QR code with an authenticator
app (any app that shows TOTP codes). From then on each sign-in asks for
the six-digit code it shows at that moment.

Lost your phone, or the codes stopped working? An administrator can
**Reset MFA** on your user page; you scan a new code at your next
sign-in.`,
  aliases: [
    "two-factor",
    "2fa",
    "authenticator",
    "totp",
    "code",
    "multi-factor",
  ],
  related: ["signing-in-with-mfa"],
} satisfies HelpTerm;
