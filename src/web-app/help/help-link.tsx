import type { ReactNode } from "react";
import { Anchor } from "@mantine/core";
import { useHelpStore } from "../stores/help-store";
import { useHelpEnabled } from "./help-preference";

/**
 * An inline link that opens the Help panel on a term or topic. With help
 * off it is its text alone. `id` must exist in help/content/ (checked by
 * tests/help-content.test.ts).
 */
export default function HelpLink({
  id,
  children,
}: {
  id: string;
  children: ReactNode;
}) {
  const openHelp = useHelpStore((state) => state.openHelp);
  if (!useHelpEnabled()) return <>{children}</>;
  return (
    <Anchor
      component="button"
      type="button"
      fz="inherit"
      onClick={() => openHelp(id)}
    >
      {children}
    </Anchor>
  );
}
