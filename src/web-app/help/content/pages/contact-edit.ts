import { ROUTES } from "../../../constants/routes";
import type { PageHelp } from "../../help-types";

export default {
  route: ROUTES.CONTACT_EDIT,
  title: "Edit a contact",
  purpose:
    "One person texted about a device: their name, number and which " +
    "events they are texted about.",
  steps: [
    "A US number can be typed any way, 704.860.9088 included; the page " +
      "shows the form it is saved in.",
    "A new contact is texted an invite and gets nothing else until they " +
      "reply Y.",
    "Once they have replied their number cannot change: add a new contact.",
  ],
  terms: ["contact-status"],
  topics: ["setting-up-a-device", "alarm-life"],
} satisfies PageHelp;
