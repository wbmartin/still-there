import { ActionIcon, Anchor, Group, Popover, Text } from "@mantine/core";
import { IconHelpCircle } from "@tabler/icons-react";
import { useHelpStore } from "../stores/help-store";
import { entrySummary, entryTitle, getEntry } from "./help-registry";
import { useHelpEnabled } from "./help-preference";

/**
 * A small "?" beside a field label or heading. Click (or Enter / Space)
 * shows the term's definition or the topic's summary; "Learn more" opens
 * the panel on it. Nothing with help off. For a one-line hint that needs
 * no panel entry, components/help-tip.tsx is the lighter choice. `id`
 * must exist in help/content/ (checked by tests/help-content.test.ts).
 */
export default function FieldHelp({
  id,
  label,
}: {
  id: string;
  label?: string;
}) {
  const openHelp = useHelpStore((state) => state.openHelp);
  const helpEnabled = useHelpEnabled();
  const entry = getEntry(id);
  if (!helpEnabled || !entry) return null;
  const title = entryTitle(entry);
  return (
    <Popover width={300} position="top" withArrow shadow="md">
      <Popover.Target>
        <ActionIcon
          variant="subtle"
          color="gray"
          size="sm"
          radius="xl"
          aria-label={label ?? `Help: ${title}`}
          style={{ verticalAlign: "middle" }}
        >
          <IconHelpCircle size={16} />
        </ActionIcon>
      </Popover.Target>
      <Popover.Dropdown>
        <Text size="sm" fw={600} mb={4}>
          {title}
        </Text>
        <Text size="sm">{entrySummary(entry)}</Text>
        <Group justify="flex-end" mt={6}>
          <Anchor
            component="button"
            type="button"
            size="sm"
            onClick={() => openHelp(id)}
          >
            Learn more
          </Anchor>
        </Group>
      </Popover.Dropdown>
    </Popover>
  );
}
