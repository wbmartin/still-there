import { ROUTES } from "../../../constants/routes";
import type { PageHelp } from "../../help-types";

export default {
  route: ROUTES.MFA_VERIFY,
  title: "Enter your code",
  purpose:
    "The second step of signing in: the six-digit code your " +
    "authenticator app shows now.",
  terms: ["mfa"],
  topics: ["signing-in-with-mfa"],
} satisfies PageHelp;
