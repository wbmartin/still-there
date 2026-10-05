// Field help sits behind an info icon beside the label rather than as grey
// text under it, so a form's fields stay in line and the help is there when
// wanted. Write help for the person using the screen, not spec references.
import type { ReactNode } from "react";
import { Tooltip } from "@mantine/core";
import { IconInfoCircle } from "@tabler/icons-react";

// The info icon; hovering or focusing it shows `help`. `about` names what
// it explains, for screen readers. It takes focus so a keyboard user can
// open it too.
export function HelpTip({ help, about }: { help: ReactNode; about: string }) {
  return (
    <Tooltip
      multiline
      w={300}
      label={help}
      withinPortal
      events={{ hover: true, focus: true, touch: true }}
    >
      <span
        role="img"
        aria-label={`About ${about}`}
        tabIndex={0}
        style={{ display: "inline-flex", verticalAlign: "middle" }}
      >
        <IconInfoCircle
          size={16}
          color="var(--mantine-color-dimmed)"
          aria-hidden
        />
      </span>
    </Tooltip>
  );
}

// A label with its help beside it, for an input's `label` in place of
// `description`. Empty `help` renders the bare label, so a hint that only
// sometimes applies can be passed conditionally.
export function LabelWithHelp({
  label,
  help,
}: {
  label: string;
  help?: ReactNode;
}) {
  if (!help) return <>{label}</>;
  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: 4 }}>
      {label}
      <HelpTip help={help} about={label} />
    </span>
  );
}
