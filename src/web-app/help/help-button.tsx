import { ActionIcon, Indicator, Tooltip } from "@mantine/core";
import { IconHelp } from "@tabler/icons-react";
import { useHelpStore } from "../stores/help-store";
import { useWhatsNewUnread } from "./help-preference";

/**
 * The header's Help button: opens the panel on the page's help. A dot
 * shows while there is a "What's new" entry the user has not seen.
 */
export default function HelpButton() {
  const openHelp = useHelpStore((state) => state.openHelp);
  const unread = useWhatsNewUnread();
  const label = unread ? "Help (something new)" : "Help";
  return (
    <Tooltip label={`${label} — press ?`}>
      <Indicator disabled={!unread} size={8} offset={4} processing={false}>
        <ActionIcon
          variant="subtle"
          size="lg"
          aria-label={label}
          onClick={() => openHelp(null)}
        >
          <IconHelp size={20} />
        </ActionIcon>
      </Indicator>
    </Tooltip>
  );
}
