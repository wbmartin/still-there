// A list's filter row stays on screen while its rows scroll: the rows are
// what the user reads, the filter is what they read them with. No template
// list page has a filter row yet; a page that adds one wraps it in this.
import { Box } from "@mantine/core";
import type { ReactNode } from "react";

// Sticky against the page scroll, offset by the shell header plus the
// page's pinned header block (--app-page-header-height, written by
// hooks/use-pinned-page-header.ts), so the bar lands directly beneath both.
// The background is the body colour, not transparent: rows showing through
// the bar are worse than no sticky bar. The negative top margin cancels the
// top padding so adding the wrapper does not shift the page. zIndex 2 sits
// under the pinned header (3) and clears table content, but stays below
// Mantine's overlay range, so a Select dropdown opened from the bar still
// renders above it.
export function StickyFilters({ children }: { children: ReactNode }) {
  return (
    <Box
      style={{
        position: "sticky",
        top:
          "calc(var(--app-shell-header-offset, 0px)" +
          " + var(--app-page-header-height, 0px))",
        zIndex: 2,
        background: "var(--mantine-color-body)",
        paddingTop: "var(--mantine-spacing-xs)",
        paddingBottom: "var(--mantine-spacing-xs)",
        marginTop: "calc(var(--mantine-spacing-xs) * -1)",
      }}
    >
      {children}
    </Box>
  );
}
