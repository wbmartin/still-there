import type { HelpTopic } from "../../help-types";

export default {
  id: "signing-in-with-mfa",
  title: "Signing in with MFA",
  summary:
    "Your password, then a six-digit code from your authenticator app. " +
    "The first time, you scan a QR code to set it up.",
  body: `1. Sign in with your username and password.
2. The first time, scan the QR code with an authenticator app on your
   phone (or type the key shown below it).
3. Enter the six-digit code the app shows. Codes change every 30 seconds;
   use the current one.

A code refused? Check the time on your phone is set automatically. Lost
the phone? Ask an administrator to **Reset MFA** for you.`,
  terms: ["mfa"],
  audience: ["everyone"],
} satisfies HelpTopic;
