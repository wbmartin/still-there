import { Badge } from "@mantine/core";
import type { ContactStatus } from "../types/contact-type";

// A contact's status as the watchdog left it: invited until they text Y,
// opted out once they text STOP.
const STATUS_BADGE: Record<ContactStatus, { label: string; color: string }> =
  {
    invited: { label: "Invited (waiting for Y)", color: "yellow" },
    active: { label: "Active", color: "green" },
    opted_out: { label: "Opted out", color: "red" },
  };

export function ContactStatusBadge({ status }: { status: ContactStatus }) {
  const badge = STATUS_BADGE[status] ?? { label: status, color: "gray" };
  return (
    <Badge color={badge.color} variant="light">
      {badge.label}
    </Badge>
  );
}
