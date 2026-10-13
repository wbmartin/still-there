import { useEffect, useState } from "react";
import {
  Alert,
  Container,
  Group,
  Loader,
  Paper,
  SegmentedControl,
  SimpleGrid,
  Stack,
  Text,
  Title,
} from "@mantine/core";
import useAuthGuard from "../hooks/use-auth-guard";
import { API_ENDPOINTS } from "../constants/api";
import { grants } from "../constants/grants";
import {
  entryTitle,
  getEntry,
  getTour,
} from "../help/help-registry";
import { useHelpEnabled } from "../help/help-preference";
import type { ApiResponse } from "../types/api";
import type { HelpInsightCount, HelpInsights } from "../types/help-type";
import { apiJson } from "../utils/api";
import { reportError } from "../utils/report-error";

const REQUIRED_GRANT_LIST = [grants.READ_USER];

const dayOptionList = [
  { value: "7", label: "7 days" },
  { value: "30", label: "30 days" },
  { value: "90", label: "90 days" },
];

// An id as its title; an id the content no longer has stays as it is.
const entryLabel = (id: string) => {
  const entry = getEntry(id);
  return entry ? entryTitle(entry) : id;
};
const tourLabel = (id: string) => getTour(id)?.title ?? id;

function CountList({
  title,
  rowList,
  label,
}: {
  title: string;
  rowList: HelpInsightCount[];
  label: (key: string) => string;
}) {
  return (
    <Paper withBorder p="sm">
      <Text size="sm" fw={600} mb={4}>
        {title}
      </Text>
      {rowList.length === 0 ? (
        <Text size="xs" c="dimmed">
          None in this period.
        </Text>
      ) : (
        <Stack gap={2}>
          {rowList.map((row) => (
            <Group key={row.key} justify="space-between" wrap="nowrap">
              <Text size="xs" lineClamp={1}>
                {label(row.key)}
              </Text>
              <Text size="xs" c="dimmed">
                {row.count}
              </Text>
            </Group>
          ))}
        </Stack>
      )}
    </Paper>
  );
}

/**
 * What the tenant's users search the help for without finding it, which
 * help they read and which tours they finish or skip: where to write help
 * next (doc/151-help-module.md). Counts only, never who. User admins
 * (READ_USER); the server checks it too.
 */
function HelpInsightsPage() {
  const authState = useAuthGuard(REQUIRED_GRANT_LIST);
  const helpEnabled = useHelpEnabled();
  const [days, setDays] = useState("30");
  const [insights, setInsights] = useState<HelpInsights | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (authState !== "YES" || !helpEnabled) return;
    let cancelled = false;
    apiJson<ApiResponse<HelpInsights>>(
      `${API_ENDPOINTS.HELP_INSIGHTS}?days=${days}`,
    )
      .then((response) => {
        if (cancelled) return;
        setError(null);
        setInsights(response.data);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        reportError(err, "loading help insights");
        setError(
          err instanceof Error ? err.message : "Failed to load help insights",
        );
      });
    return () => {
      cancelled = true;
    };
  }, [authState, helpEnabled, days]);

  //============================^ HOOK LINE ^====================
  if (authState === "CHECKING") {
    return (
      <Container size={1280} px={32} py={32} mx="auto" ta="center">
        <Loader />
      </Container>
    );
  }
  if (authState !== "YES") {
    return (
      <Container size={1280} px={32} py={32} mx="auto">
        <Alert color="red">Authentication required.</Alert>
      </Container>
    );
  }

  return (
    <Container size={1280} px={32} py={32} mx="auto">
      <Group justify="space-between" align="flex-end" mb="lg">
        <Title order={1}>Help insights</Title>
        {helpEnabled && (
          <SegmentedControl
            size="xs"
            value={days}
            onChange={setDays}
            data={dayOptionList}
            aria-label="Period"
          />
        )}
      </Group>
      {!helpEnabled ? (
        <Alert color="gray">
          Help is not turned on for this app, so there is nothing to show.
        </Alert>
      ) : (
        <Stack gap="md">
          <Text size="sm" c="dimmed">
            What people search the help for without finding it, which help
            they read, and which page tours they finish or skip: where to
            write help next. Searches are shown as typed and may include
            anything people typed.
          </Text>
          {error && <Alert color="red">{error}</Alert>}
          {!insights && !error && <Loader size="sm" />}
          {insights && (
            <SimpleGrid cols={{ base: 1, sm: 2 }}>
              <CountList
                title="Searches with no results"
                rowList={insights.noResultSearchList}
                label={(key) => `“${key}”`}
              />
              <CountList
                title="Most-read help"
                rowList={insights.entryViewList}
                label={entryLabel}
              />
              <CountList
                title="Tours finished"
                rowList={insights.tourCompleteList}
                label={tourLabel}
              />
              <CountList
                title="Tours skipped"
                rowList={insights.tourSkipList}
                label={tourLabel}
              />
            </SimpleGrid>
          )}
        </Stack>
      )}
    </Container>
  );
}

export default HelpInsightsPage;
