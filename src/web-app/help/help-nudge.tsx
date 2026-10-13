import type { ReactNode } from "react";
import { Alert, Group, Text } from "@mantine/core";
import { IconBulb } from "@tabler/icons-react";
import HelpLink from "./help-link";
import {
  dismissNudge,
  useHelpEnabled,
  useHelpPreference,
} from "./help-preference";

/**
 * A tip a page shows when its state suggests the user could use one (the
 * page decides when to render it). Dismissing it is remembered for the
 * user; it stays hidden until their preference has loaded, so a
 * dismissed tip never flashes. Nothing with help off. `help`, when given,
 * must exist in help/content/ (checked by tests/help-content.test.ts).
 */
export default function Nudge({
  id,
  help,
  title,
  children,
}: {
  // Stable id, the shape of a help id: what dismissing remembers.
  id: string;
  help?: string;
  title: string;
  children: ReactNode;
}) {
  const helpEnabled = useHelpEnabled();
  const { preference, loaded } = useHelpPreference();
  if (!helpEnabled || !loaded || preference.dismissed.includes(id)) {
    return null;
  }
  return (
    <Alert
      variant="light"
      icon={<IconBulb size={16} />}
      title={title}
      withCloseButton
      closeButtonLabel="Dismiss tip"
      onClose={() => void dismissNudge(id)}
      mb="md"
    >
      <Text size="sm">{children}</Text>
      {help && (
        <Group mt={4}>
          <HelpLink id={help}>Learn more</HelpLink>
        </Group>
      )}
    </Alert>
  );
}
