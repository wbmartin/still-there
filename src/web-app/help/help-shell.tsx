import { useEffect } from "react";
import { useSearchParams } from "react-router-dom";
import "@mantine/spotlight/styles.css";
import { useHelpStore } from "../stores/help-store";
import CommandPalette from "./command-palette";
import HelpPanel from "./help-panel";
import { getEntry } from "./help-registry";
import TourRunner from "./tour-runner";

// Whether a key press belongs to something the user is typing in.
const isTypingTarget = (target: EventTarget | null): boolean => {
  if (!(target instanceof HTMLElement)) return false;
  return (
    target.isContentEditable ||
    ["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName)
  );
};

/**
 * The help module's always-there parts, loaded on first use and mounted
 * only while signed in with help on (app.tsx): the Help panel, the tour
 * runner, the command palette, the ? key and the `?help=<id>` deep link
 * (doc/151-help-module.md).
 */
export default function HelpShell() {
  const openHelp = useHelpStore((state) => state.openHelp);
  const closeHelp = useHelpStore((state) => state.closeHelp);
  const [searchParams, setSearchParams] = useSearchParams();
  const deepLink = searchParams.get("help");

  // ?help=<id> opens the panel on that entry (its home when the id is
  // unknown), then leaves the URL without it.
  useEffect(() => {
    if (deepLink === null) return;
    openHelp(getEntry(deepLink) ? deepLink : null);
    setSearchParams(
      (previous) => {
        const next = new URLSearchParams(previous);
        next.delete("help");
        return next;
      },
      { replace: true },
    );
  }, [deepLink, openHelp, setSearchParams]);

  // ? anywhere but a text field toggles the panel.
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key !== "?" || event.metaKey || event.ctrlKey) return;
      if (event.altKey || isTypingTarget(event.target)) return;
      event.preventDefault();
      if (useHelpStore.getState().opened) closeHelp();
      else openHelp(null);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [openHelp, closeHelp]);

  return (
    <>
      <HelpPanel />
      <TourRunner />
      <CommandPalette />
    </>
  );
}
