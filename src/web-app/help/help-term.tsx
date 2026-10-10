import { useRef, useState, type ReactNode } from "react";
import { Anchor, Group, Popover, Text } from "@mantine/core";
import { useHelpStore } from "../stores/help-store";
import { getTerm } from "./help-registry";
import { useHelpEnabled } from "./help-preference";

/**
 * A glossary term in running text: a dotted underline; hover or keyboard
 * focus shows its one-line definition, Enter or "Learn more" opens the
 * Help panel on it, Escape closes. With help off it is its text alone.
 * `id` must exist in help/content/glossary.ts (checked by
 * tests/help-content.test.ts).
 */
export default function Term({
  id,
  children,
}: {
  id: string;
  children?: ReactNode;
}) {
  const term = getTerm(id);
  const openHelp = useHelpStore((state) => state.openHelp);
  const helpEnabled = useHelpEnabled();
  const [opened, setOpened] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  if (!term || !helpEnabled) return <>{children ?? term?.term}</>;

  // A short delay both ways, so the pointer can cross into the card.
  const setOpenedSoon = (next: boolean) => {
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => setOpened(next), 150);
  };

  return (
    <Popover
      opened={opened}
      onChange={setOpened}
      width={300}
      position="top"
      withArrow
      shadow="md"
    >
      <Popover.Target>
        <span
          tabIndex={0}
          role="button"
          aria-label={`${term.term}: what is this?`}
          onMouseEnter={() => setOpenedSoon(true)}
          onMouseLeave={() => setOpenedSoon(false)}
          onFocus={() => setOpenedSoon(true)}
          onBlur={() => setOpenedSoon(false)}
          onKeyDown={(event) => {
            if (event.key === "Escape") setOpened(false);
            if (event.key === "Enter") openHelp(id);
          }}
          style={{
            textDecoration: "underline dotted",
            textUnderlineOffset: 3,
            cursor: "help",
          }}
        >
          {children ?? term.term}
        </span>
      </Popover.Target>
      <Popover.Dropdown
        onMouseEnter={() => setOpenedSoon(true)}
        onMouseLeave={() => setOpenedSoon(false)}
      >
        <Text size="sm" fw={600} mb={4}>
          {term.term}
        </Text>
        <Text size="sm">{term.short}</Text>
        <Group justify="flex-end" mt={6}>
          <Anchor
            component="button"
            type="button"
            size="sm"
            onClick={() => {
              setOpened(false);
              openHelp(id);
            }}
          >
            Learn more
          </Anchor>
        </Group>
      </Popover.Dropdown>
    </Popover>
  );
}
