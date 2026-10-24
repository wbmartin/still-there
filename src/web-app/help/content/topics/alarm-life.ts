import type { HelpTopic } from "../../help-types";

export default {
  id: "alarm-life",
  title: "How an alarm runs",
  summary:
    "Raised, then acknowledged or turned off, then resolved when the band " +
    "or base station is back; who is texted at each step.",
  body: `- **Raised**: a band went absent or the base station stopped
  checking in. Every contact is texted, and again every reminder interval.
- **Acknowledged**: someone replied OK, or pressed **Acknowledge** on
  [Alerts](/alertList). Everyone is told who; then one reminder a day at
  the daily reminder time while it stays open.
- **Turned off**: **Turn off** on Alerts. Everyone is told who; no more
  reminders.
- **Resolved**: the band was seen again (or the base station checked in).
  Everyone is told alerting is re-activated.

The Alerts page refreshes itself every 30 seconds. Reminder timing is on
[Settings](/tenantSetting).`,
  terms: ["band", "base-station"],
  related: ["setting-up-a-device"],
  audience: ["everyone"],
} satisfies HelpTopic;
