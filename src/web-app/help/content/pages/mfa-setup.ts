import { ROUTES } from "../../../constants/routes";
import type { PageHelp } from "../../help-types";

export default {
  route: ROUTES.MFA_SETUP,
  title: "Set up MFA",
  purpose:
    "Link an authenticator app to your login: scan the QR code, then " +
    "enter the six-digit code it shows.",
  steps: [
    "Can't scan? Copy the key under the code into the app instead.",
  ],
  terms: ["mfa"],
  topics: ["signing-in-with-mfa"],
} satisfies PageHelp;
