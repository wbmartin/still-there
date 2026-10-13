import { ROUTES } from "../../../constants/routes";
import type { PageHelp } from "../../help-types";

export default {
  route: ROUTES.LOGIN,
  title: "Sign in",
  purpose:
    "Sign in with your username and password, then the code from your " +
    "authenticator app.",
  steps: [
    "Too many wrong tries lock the login for a few minutes; wait and " +
      "try again.",
    "Forgot your password? Ask an administrator for a new invite link " +
      "or a new password.",
  ],
  terms: ["mfa"],
  topics: ["signing-in-with-mfa"],
} satisfies PageHelp;
