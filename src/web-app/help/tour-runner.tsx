import { useCallback, useEffect, useMemo, useState } from "react";
import { Button, Group, Paper, Portal, Text } from "@mantine/core";
import { useHelpStore } from "../stores/help-store";
import { logHelpEvent } from "./help-events";
import { markTourDone, useHelpPreference } from "./help-preference";
import { TOURS, tourForRoute } from "./help-registry";
import type { TourStep } from "./help-types";
import { useRouteKey } from "./use-route-key";

const CARD_WIDTH = 320;
const CARD_HEIGHT = 170;
const OUTLINE = "3px solid var(--mantine-primary-color-filled)";

const findTarget = (step: TourStep): HTMLElement | null =>
  document.querySelector<HTMLElement>(
    `[data-tour="${CSS.escape(step.target)}"]`,
  );

/**
 * Runs the active page tour: outlines each step's element (its data-tour
 * attribute) and shows a card beside it with Back / Next / Skip. A step
 * whose element is not on the page is passed over. Keyboard: right arrow
 * next, left arrow back, Escape skips. Leaving the page ends the tour
 * without marking it done. A tour with autoStart starts once on the
 * user's first visit, when its page has rendered a target.
 */
export default function TourRunner() {
  const route = useRouteKey();
  const activeTourId = useHelpStore((state) => state.activeTourId);
  const index = useHelpStore((state) => state.tourStep);
  const startTour = useHelpStore((state) => state.startTour);
  const setTourStep = useHelpStore((state) => state.setTourStep);
  const endTour = useHelpStore((state) => state.endTour);
  const { preference, loaded } = useHelpPreference();
  const tour = useMemo(
    () => TOURS.find((candidate) => candidate.id === activeTourId),
    [activeTourId],
  );
  const [rect, setRect] = useState<DOMRect | null>(null);
  const toursDone = preference.toursDone.join(" ");

  // First visit: start this page's auto-start tour once a target is up.
  useEffect(() => {
    const pageTour = tourForRoute(route);
    if (!loaded || !pageTour?.autoStart || activeTourId) return;
    if (toursDone.split(" ").includes(pageTour.id)) return;
    const timer = setTimeout(() => {
      if (pageTour.steps.some((step) => findTarget(step))) {
        startTour(pageTour.id);
      }
    }, 1200);
    return () => clearTimeout(timer);
  }, [route, loaded, toursDone, activeTourId, startTour]);

  // A tour belongs to its page.
  useEffect(() => {
    if (tour && tour.route !== route) endTour();
  }, [route, tour, endTour]);

  const stepList = useMemo(() => (tour ? tour.steps : []), [tour]);
  const present = useCallback(
    (at: number) =>
      at >= 0 && at < stepList.length && findTarget(stepList[at]) !== null,
    [stepList],
  );

  const finish = useCallback(
    (completed: boolean) => {
      if (tour) {
        void markTourDone(tour.id);
        logHelpEvent(completed ? "tour_complete" : "tour_skip", {
          route,
          entryId: tour.id,
        });
      }
      endTour();
    },
    [tour, route, endTour],
  );

  const go = useCallback(
    (direction: 1 | -1, from: number) => {
      let next = from + direction;
      while (next >= 0 && next < stepList.length && !present(next)) {
        next += direction;
      }
      if (next >= stepList.length) finish(from >= 0);
      else if (next >= 0) setTourStep(next);
    },
    [stepList, present, finish, setTourStep],
  );

  // A new tour starts at -1: move to the first step on the page.
  useEffect(() => {
    if (tour && index === -1) go(1, -1);
  }, [tour, index, go]);

  // Outline the step's element and follow it as the page moves.
  useEffect(() => {
    const step = tour ? stepList[index] : undefined;
    const element = step ? findTarget(step) : null;
    if (!element) return;
    element.scrollIntoView({ block: "center", behavior: "smooth" });
    const previousOutline = element.style.outline;
    const previousOffset = element.style.outlineOffset;
    element.style.outline = OUTLINE;
    element.style.outlineOffset = "4px";
    const update = () => setRect(element.getBoundingClientRect());
    const frame = requestAnimationFrame(update);
    window.addEventListener("scroll", update, true);
    window.addEventListener("resize", update);
    const timer = setInterval(update, 300);
    return () => {
      element.style.outline = previousOutline;
      element.style.outlineOffset = previousOffset;
      cancelAnimationFrame(frame);
      window.removeEventListener("scroll", update, true);
      window.removeEventListener("resize", update);
      clearInterval(timer);
    };
  }, [tour, index, stepList]);

  useEffect(() => {
    if (!tour) return;
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") finish(false);
      else if (event.key === "ArrowRight") go(1, index);
      else if (event.key === "ArrowLeft") go(-1, index);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [tour, index, go, finish]);

  if (!tour || !rect || !stepList[index]) return null;
  const step = stepList[index];
  const onPageList = stepList.filter((_, at) => present(at));
  const position = onPageList.indexOf(step) + 1;
  const below = rect.bottom + CARD_HEIGHT + 12 < window.innerHeight;
  const top = below
    ? rect.bottom + 12
    : Math.max(12, rect.top - 12 - CARD_HEIGHT);
  const left = Math.min(
    Math.max(12, rect.left),
    window.innerWidth - CARD_WIDTH - 12,
  );

  return (
    <Portal>
      <Paper
        role="dialog"
        aria-label={`${tour.title}: ${step.title}`}
        shadow="lg"
        withBorder
        p="md"
        radius="md"
        w={CARD_WIDTH}
        style={{ position: "fixed", top, left, zIndex: 400 }}
        data-testid="help-tour-card"
      >
        <Text size="xs" c="dimmed" mb={2}>
          {tour.title} &middot; {position} of {onPageList.length}
        </Text>
        <Text fw={600} size="sm" mb={4}>
          {step.title}
        </Text>
        <Text size="sm">{step.body}</Text>
        <Group justify="space-between" mt="sm">
          <Button
            variant="subtle"
            size="compact-sm"
            color="gray"
            onClick={() => finish(false)}
          >
            Skip tour
          </Button>
          <Group gap={6}>
            {position > 1 && (
              <Button
                variant="default"
                size="compact-sm"
                onClick={() => go(-1, index)}
              >
                Back
              </Button>
            )}
            <Button size="compact-sm" onClick={() => go(1, index)}>
              {position === onPageList.length ? "Done" : "Next"}
            </Button>
          </Group>
        </Group>
      </Paper>
    </Portal>
  );
}
