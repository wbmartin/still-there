import { useEffect, useMemo, useState, type ReactNode } from "react";
import { Link } from "react-router-dom";
import {
  ActionIcon,
  Anchor,
  Badge,
  Button,
  Divider,
  Drawer,
  Group,
  List,
  ScrollArea,
  Stack,
  Text,
  TextInput,
  Title,
  UnstyledButton,
} from "@mantine/core";
import {
  IconArrowLeft,
  IconChartBar,
  IconHome,
  IconRoute,
  IconSearch,
} from "@tabler/icons-react";
import { ROUTES } from "../constants/routes";
import { grants } from "../constants/grants";
import { useAuthStore } from "../stores/auth-store";
import { useHelpStore } from "../stores/help-store";
import { logHelpEvent } from "./help-events";
import HelpMarkdownView from "./help-markdown-view";
import { markWhatsNewSeen, useHelpPreference } from "./help-preference";
import {
  GLOSSARY,
  TOPICS,
  WHATS_NEW,
  entryTitle,
  getEntry,
  getPageHelp,
  latestWhatsNewId,
  searchHelp,
  tourForRoute,
} from "./help-registry";
import { useHelpAudience } from "./use-help-audience";
import { useRouteKey } from "./use-route-key";

// How many "What's new" entries the home view lists.
const WHATS_NEW_SHOWN = 3;

function SectionLabel({ children }: { children: ReactNode }) {
  return (
    <Text size="xs" c="dimmed" tt="uppercase" fw={600}>
      {children}
    </Text>
  );
}

// A term or topic as a chip that opens it.
function EntryChip({ id }: { id: string }) {
  const openHelp = useHelpStore((state) => state.openHelp);
  const entry = getEntry(id);
  if (!entry) return null;
  return (
    <Badge
      component="button"
      type="button"
      variant="light"
      tt="none"
      style={{ cursor: "pointer" }}
      onClick={() => openHelp(id)}
    >
      {entryTitle(entry)}
    </Badge>
  );
}

function EntryLink({ id }: { id: string }) {
  const openHelp = useHelpStore((state) => state.openHelp);
  const entry = getEntry(id);
  if (!entry) return null;
  return (
    <Anchor
      component="button"
      type="button"
      size="sm"
      ta="left"
      onClick={() => openHelp(id)}
    >
      {entryTitle(entry)}
    </Anchor>
  );
}

function EntryView({ id, route }: { id: string; route: string }) {
  const entry = getEntry(id);
  useEffect(() => {
    if (getEntry(id)) logHelpEvent("entry_view", { route, entryId: id });
  }, [id, route]);
  if (!entry) {
    return <Text size="sm">That help entry does not exist.</Text>;
  }
  const chipIdList =
    entry.kind === "term"
      ? (entry.item.related ?? [])
      : [...(entry.item.terms ?? []), ...(entry.item.related ?? [])];
  return (
    <Stack gap="sm">
      <Title order={3}>{entryTitle(entry)}</Title>
      {entry.kind === "term" ? (
        <Text size="sm" fw={500}>
          {entry.item.short}
        </Text>
      ) : (
        <Text size="sm" c="dimmed">
          {entry.item.summary}
        </Text>
      )}
      <HelpMarkdownView body={entry.item.body} />
      {chipIdList.length > 0 && (
        <>
          <Text size="xs" c="dimmed" mt="xs">
            {entry.kind === "term" ? "Related" : "Terms and related help"}
          </Text>
          <Group gap={6}>
            {chipIdList.map((chipId) => (
              <EntryChip key={chipId} id={chipId} />
            ))}
          </Group>
        </>
      )}
    </Stack>
  );
}

// The newest entries, each marked New past what the user had seen when
// the panel opened.
function WhatsNew({ seenAtOpen }: { seenAtOpen: string }) {
  const openHelp = useHelpStore((state) => state.openHelp);
  if (WHATS_NEW.length === 0) return null;
  return (
    <Stack gap={6}>
      <SectionLabel>What&rsquo;s new</SectionLabel>
      {WHATS_NEW.slice(0, WHATS_NEW_SHOWN).map((news) => (
        <Stack key={news.id} gap={2}>
          <Group gap={6}>
            <Text size="sm" fw={600}>
              {news.title}
            </Text>
            {news.id > seenAtOpen && (
              <Badge size="xs" variant="light">
                New
              </Badge>
            )}
          </Group>
          <HelpMarkdownView body={news.body} size="xs" />
          {news.help && (
            <Anchor
              component="button"
              type="button"
              size="xs"
              ta="left"
              onClick={() => openHelp(news.help ?? null)}
            >
              Learn more
            </Anchor>
          )}
        </Stack>
      ))}
    </Stack>
  );
}

function HomeView({ route }: { route: string }) {
  const startTour = useHelpStore((state) => state.startTour);
  const closeHelp = useHelpStore((state) => state.closeHelp);
  const canReadInsights = useAuthStore((state) =>
    state.grantList.includes(grants.READ_USER),
  );
  const audience = useHelpAudience();
  const { preference, loaded } = useHelpPreference();
  // What had been seen when the panel opened: the New badges stay for
  // this viewing although the marker below moves on.
  const [seenAtOpen] = useState(preference.whatsNewSeen);
  const latest = latestWhatsNewId();
  useEffect(() => {
    if (loaded && latest && preference.whatsNewSeen < latest) {
      void markWhatsNewSeen(latest);
    }
  }, [loaded, latest, preference.whatsNewSeen]);

  const page = getPageHelp(route);
  const tour = tourForRoute(route);
  const pageTopicSet = new Set(page?.topics ?? []);
  const suggestedList = TOPICS.filter(
    (topic) =>
      !pageTopicSet.has(topic.id) && topic.audience?.includes(audience),
  ).slice(0, 3);
  const shownSet = new Set([
    ...pageTopicSet,
    ...suggestedList.map((topic) => topic.id),
  ]);
  const otherTopicList = TOPICS.filter((topic) => !shownSet.has(topic.id));

  return (
    <Stack gap="md">
      {page && (
        <Stack gap={6}>
          <SectionLabel>On this page</SectionLabel>
          <Title order={3}>{page.title}</Title>
          <Text size="sm">{page.purpose}</Text>
          {tour && (
            <Button
              variant="light"
              size="xs"
              leftSection={<IconRoute size={14} />}
              onClick={() => startTour(tour.id)}
              style={{ alignSelf: "flex-start" }}
            >
              Show me around this page
            </Button>
          )}
          {page.steps && page.steps.length > 0 && (
            <List size="sm" spacing={4}>
              {page.steps.map((step) => (
                <List.Item key={step}>{step}</List.Item>
              ))}
            </List>
          )}
          {page.terms && page.terms.length > 0 && (
            <Group gap={6} mt={4}>
              {page.terms.map((id) => (
                <EntryChip key={id} id={id} />
              ))}
            </Group>
          )}
          {page.topics && page.topics.length > 0 && (
            <Stack gap={2} mt={4}>
              {page.topics.map((id) => (
                <EntryLink key={id} id={id} />
              ))}
            </Stack>
          )}
        </Stack>
      )}
      {suggestedList.length > 0 && (
        <>
          <Divider />
          <Stack gap={2}>
            <SectionLabel>Suggested for you</SectionLabel>
            {suggestedList.map((topic) => (
              <EntryLink key={topic.id} id={topic.id} />
            ))}
          </Stack>
        </>
      )}
      {WHATS_NEW.length > 0 && (
        <>
          <Divider />
          <WhatsNew seenAtOpen={seenAtOpen} />
        </>
      )}
      {otherTopicList.length > 0 && (
        <>
          <Divider />
          <Stack gap={2}>
            <SectionLabel>{page ? "More help" : "Help topics"}</SectionLabel>
            {otherTopicList.map((topic) => (
              <EntryLink key={topic.id} id={topic.id} />
            ))}
          </Stack>
        </>
      )}
      {GLOSSARY.length > 0 && (
        <>
          <Divider />
          <Stack gap={6}>
            <SectionLabel>Glossary</SectionLabel>
            <Group gap={6}>
              {[...GLOSSARY]
                .sort((a, b) => a.term.localeCompare(b.term))
                .map((term) => (
                  <EntryChip key={term.id} id={term.id} />
                ))}
            </Group>
          </Stack>
        </>
      )}
      {canReadInsights && (
        <>
          <Divider />
          <Anchor
            component={Link}
            to={ROUTES.HELP_INSIGHTS}
            size="sm"
            onClick={closeHelp}
          >
            <Group gap={4} wrap="nowrap">
              <IconChartBar size={14} />
              Help insights: what people search for and read
            </Group>
          </Anchor>
        </>
      )}
    </Stack>
  );
}

function SearchResultList({
  query,
  onOpen,
}: {
  query: string;
  onOpen: (id: string) => void;
}) {
  const route = useRouteKey();
  const resultList = useMemo(() => searchHelp(query), [query]);
  const trimmed = query.trim();

  // A search that finds nothing is a gap in the help: recorded once the
  // user stops typing (the Help insights page).
  useEffect(() => {
    if (trimmed.length < 3 || resultList.length > 0) return;
    const timer = setTimeout(
      () => logHelpEvent("search_no_result", { route, query: trimmed }),
      1500,
    );
    return () => clearTimeout(timer);
  }, [trimmed, resultList.length, route]);

  if (resultList.length === 0) {
    return (
      <Text size="sm" c="dimmed">
        Nothing in the help matches &ldquo;{trimmed}&rdquo;. Try another
        word.
      </Text>
    );
  }
  return (
    <Stack gap={4}>
      {resultList.map((result) => (
        <UnstyledButton
          key={result.id}
          onClick={() => onOpen(result.id)}
          p="xs"
          style={{ borderRadius: 6 }}
        >
          <Group gap={6} wrap="nowrap" align="flex-start">
            <Badge size="xs" variant="light" mt={3}>
              {result.kind === "term" ? "Term" : "Topic"}
            </Badge>
            <div>
              <Text size="sm" fw={600}>
                {result.title}
              </Text>
              <Text size="xs" c="dimmed">
                {result.summary}
              </Text>
            </div>
          </Group>
        </UnstyledButton>
      ))}
    </Stack>
  );
}

/**
 * The Help drawer: "On this page" for the current route, search across
 * the glossary and topics, and a reader for any entry. Opened by the
 * header's Help button, the ? key, a Term / FieldHelp / HelpLink, or a
 * `?help=<id>` link (help-shell.tsx). `askSlot` is the seam for an
 * assistant that answers a typed question; the template passes none, so
 * no such UI exists until an app adds one.
 */
export default function HelpPanel({
  askSlot,
}: {
  askSlot?: (query: string, route: string) => ReactNode;
}) {
  const opened = useHelpStore((state) => state.opened);
  const entryId = useHelpStore((state) => state.entryId);
  const historyList = useHelpStore((state) => state.historyList);
  const openHelp = useHelpStore((state) => state.openHelp);
  const back = useHelpStore((state) => state.back);
  const goHome = useHelpStore((state) => state.goHome);
  const closeHelp = useHelpStore((state) => state.closeHelp);
  const route = useRouteKey();
  const [query, setQuery] = useState("");
  const searching = query.trim() !== "";

  const close = () => {
    setQuery("");
    closeHelp();
  };

  return (
    <Drawer
      opened={opened}
      onClose={close}
      position="right"
      size="md"
      title={
        <Group gap="xs">
          {(entryId || searching) && (
            <>
              <ActionIcon
                variant="subtle"
                aria-label="Back"
                onClick={() => {
                  if (searching) setQuery("");
                  else if (historyList.length > 0) back();
                  else goHome();
                }}
              >
                <IconArrowLeft size={16} />
              </ActionIcon>
              <ActionIcon
                variant="subtle"
                aria-label="Help home"
                onClick={() => {
                  setQuery("");
                  goHome();
                }}
              >
                <IconHome size={16} />
              </ActionIcon>
            </>
          )}
          <Text fw={600}>Help</Text>
        </Group>
      }
      scrollAreaComponent={ScrollArea.Autosize}
    >
      <Stack gap="md">
        <TextInput
          placeholder="Search help"
          leftSection={<IconSearch size={14} />}
          value={query}
          onChange={(event) => setQuery(event.currentTarget.value)}
          aria-label="Search help"
          data-autofocus
        />
        {searching && askSlot?.(query, route)}
        {searching ? (
          <SearchResultList
            query={query}
            onOpen={(id) => {
              setQuery("");
              openHelp(id);
            }}
          />
        ) : entryId ? (
          <EntryView id={entryId} route={route} />
        ) : (
          <HomeView route={route} />
        )}
        {!searching && entryId && (
          <Button
            variant="subtle"
            size="xs"
            onClick={goHome}
            style={{ alignSelf: "flex-start" }}
          >
            Help for this page
          </Button>
        )}
      </Stack>
    </Drawer>
  );
}
