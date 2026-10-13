import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import {
  Alert,
  Anchor,
  Badge,
  Box,
  Button,
  Container,
  Group,
  Image,
  Loader,
  Modal,
  Paper,
  SegmentedControl,
  Stack,
  Text,
  Textarea,
  Title,
  UnstyledButton,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconPaperclip, IconPlus } from "@tabler/icons-react";
import {
  issueAttachmentUrl,
  useIssueReportStore,
} from "../stores/issue-report-store";
import { useProfileStore } from "../stores/profile-store";
import {
  ISSUE_MAX_DETAIL_LENGTH,
  ISSUE_MAX_NOTE_LENGTH,
  issueIsClosed,
  issueKindColor,
  issueKindLabel,
  type IssueAction,
  type IssueAttachment,
  type IssueReportDetail,
} from "../types/issue-report-type";
import { MarkdownView } from "../components/markdown-view";
import { MarkdownEditor } from "../components/markdown-editor";
import { IssueAttachmentPicker } from "../components/issue-attachment-picker";
import useAuthGuard from "../hooks/use-auth-guard";
import { grants } from "../constants/grants";
import { reportError } from "../utils/report-error";

// The issue-report module's page (doc/150-issue-report-module.md): the
// tenant's reports, and one report's conversation (?issueReportId=N).

// Staff replies arrive with the relay's sync, every couple of minutes.
const DETAIL_REFRESH_MS = 60_000;

const formatWhen = (epochSeconds: number): string =>
  new Date(epochSeconds * 1000).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });

const stateColor = (state: string): string => {
  if (issueIsClosed(state)) return "gray";
  if (state === "fixing" || state === "reverify") return "yellow";
  if (state === "reopened") return "orange";
  return "blue";
};

const errorText = (err: unknown, fallback: string): string =>
  err instanceof Error ? err.message : fallback;

function AttachmentRow({ itemList }: { itemList: IssueAttachment[] }) {
  if (itemList.length === 0) return null;
  return (
    <Group gap="xs" mt="xs">
      {itemList.map((attachment) => {
        const url = issueAttachmentUrl(attachment.issueReportAttachmentId);
        return attachment.contentType.startsWith("image/") ? (
          <Anchor
            key={attachment.issueReportAttachmentId}
            href={url}
            target="_blank"
            rel="noopener noreferrer"
          >
            <Image
              src={url}
              alt={attachment.filename}
              title={attachment.filename}
              h={90}
              w="auto"
              maw={160}
              fit="cover"
              radius="sm"
              style={{
                border: "1px solid var(--mantine-color-default-border)",
              }}
            />
          </Anchor>
        ) : (
          <Button
            key={attachment.issueReportAttachmentId}
            component="a"
            href={url}
            variant="default"
            size="xs"
            leftSection={<IconPaperclip size={12} />}
          >
            {attachment.filename}
          </Button>
        );
      })}
    </Group>
  );
}

function ReportDetail({
  issueReportId,
  onChanged,
}: {
  issueReportId: number;
  onChanged: () => void;
}) {
  const { readReport, createMessage, createAction } = useIssueReportStore();
  const [report, setReport] = useState<IssueReportDetail | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [body, setBody] = useState("");
  const [fileList, setFileList] = useState<File[]>([]);
  const [sending, setSending] = useState(false);
  // The action stays set while its modal closes, so the modal's words do
  // not change during the close transition.
  const [action, setAction] = useState<IssueAction>("withdraw");
  const [actionOpen, setActionOpen] = useState(false);
  const [note, setNote] = useState("");

  // Loads the report, then again every minute while it is shown. A
  // different id shows the loader until its own report arrives (the check
  // below the hook line), and an answer for an id the page has left is
  // dropped.
  useEffect(() => {
    let current = true;
    const load = () => {
      readReport(issueReportId)
        .then((loaded) => {
          if (!current) return;
          setReport(loaded);
          setLoadError(null);
        })
        .catch((err: unknown) => {
          if (current) {
            setLoadError(errorText(err, "The report could not be loaded"));
          }
        });
    };
    load();
    const timer = window.setInterval(load, DETAIL_REFRESH_MS);
    return () => {
      current = false;
      window.clearInterval(timer);
    };
  }, [readReport, issueReportId]);

  //============================^ HOOK LINE ^====================

  if (loadError) return <Alert color="red">{loadError}</Alert>;
  if (!report || report.issueReportId !== issueReportId) return <Loader />;

  const send = async () => {
    if (!body.trim() && fileList.length === 0) return;
    setSending(true);
    try {
      setReport(await createMessage(report.issueReportId, body, fileList));
      setBody("");
      setFileList([]);
      onChanged();
    } catch (err) {
      notifications.show({
        color: "red",
        title: "The reply was not sent",
        message: errorText(err, "Unknown error"),
      });
    } finally {
      setSending(false);
    }
  };

  const openAction = (next: IssueAction) => {
    setAction(next);
    setActionOpen(true);
  };

  const confirmAction = async () => {
    setSending(true);
    try {
      setReport(await createAction(report.issueReportId, action, note));
      setActionOpen(false);
      setNote("");
      onChanged();
    } catch (err) {
      notifications.show({
        color: "red",
        title: "The report was not updated",
        message: errorText(err, "Unknown error"),
      });
    } finally {
      setSending(false);
    }
  };

  const closed = issueIsClosed(report.state);

  return (
    <Stack gap="md">
      <Paper withBorder p="lg">
        <Group gap="xs" mb={6}>
          <Badge color={issueKindColor(report.kind)} variant="light">
            {issueKindLabel(report.kind)}
          </Badge>
          <Badge color={stateColor(report.state)} variant="filled">
            {report.status}
          </Badge>
          {report.relayState === "failed" && (
            <Badge
              color="orange"
              variant="outline"
              title="Sending again automatically"
            >
              Not delivered yet
            </Badge>
          )}
        </Group>
        <Title order={3} style={{ overflowWrap: "anywhere" }}>
          {report.summary}
        </Title>
        <Text size="xs" c="dimmed" mt={4}>
          {report.reporterName} · {formatWhen(report.createTs)}
          {report.pageUrl ? ` · on ${report.pageUrl}` : ""}
        </Text>
        {report.statusNote && (
          <Alert color="gray" mt="sm" p="xs">
            <Text size="sm">{report.statusNote}</Text>
          </Alert>
        )}
        <Box mt="md">
          {report.detail ? (
            <MarkdownView text={report.detail} />
          ) : (
            <Text size="sm" c="dimmed">
              No description.
            </Text>
          )}
        </Box>
        <AttachmentRow itemList={report.attachmentList} />
        <Group justify="flex-end" mt="md">
          {closed ? (
            <Button
              variant="default"
              size="xs"
              onClick={() => openAction("reopen")}
            >
              Reopen
            </Button>
          ) : (
            <Button
              variant="subtle"
              color="gray"
              size="xs"
              onClick={() => openAction("withdraw")}
            >
              Withdraw
            </Button>
          )}
        </Group>
      </Paper>

      <Title order={4}>Conversation</Title>
      {report.threadList.length === 0 && (
        <Text size="sm" c="dimmed">
          No replies yet. Replies from the support team appear here.
        </Text>
      )}
      {report.threadList.map((message) => (
        <Paper
          key={message.issueReportMessageId}
          withBorder
          p="sm"
          bg={
            message.authorKind === "staff"
              ? "var(--mantine-color-blue-light)"
              : undefined
          }
        >
          <Group justify="space-between" mb={4}>
            <Text size="sm" fw={600}>
              {message.authorName ||
                (message.authorKind === "staff" ? "Support" : "Someone")}
              {message.authorKind === "staff" && (
                <Badge ml={6} size="xs" variant="light">
                  Support
                </Badge>
              )}
            </Text>
            <Text size="xs" c="dimmed">
              {formatWhen(message.createTs)}
              {message.relayState === "failed" ? " · not delivered yet" : ""}
            </Text>
          </Group>
          <MarkdownView text={message.body} />
          <AttachmentRow itemList={message.attachmentList} />
        </Paper>
      ))}

      <Paper withBorder p="sm">
        <MarkdownEditor
          label="Reply"
          value={body}
          onChange={setBody}
          minRows={3}
          maxLength={ISSUE_MAX_DETAIL_LENGTH}
          placeholder="Add detail or answer a question"
        />
        <Box mt="xs">
          <IssueAttachmentPicker
            fileList={fileList}
            onChange={setFileList}
            allowHtml={report.kind !== "bug"}
          />
        </Box>
        <Group justify="flex-end" mt="sm">
          <Button
            loading={sending}
            disabled={!body.trim() && fileList.length === 0}
            onClick={() => void send()}
          >
            Send reply
          </Button>
        </Group>
      </Paper>

      <Modal
        opened={actionOpen}
        onClose={() => (sending ? undefined : setActionOpen(false))}
        title={
          action === "withdraw" ? "Withdraw this report" : "Reopen this report"
        }
        centered
        size="sm"
      >
        <Stack gap="sm">
          <Text size="sm">
            {action === "withdraw"
              ? "Closes the report: it is no longer needed, or you found " +
                "the answer."
              : "Use this if it is not fixed after all, or it came back."}
          </Text>
          <Textarea
            label="Note (optional)"
            value={note}
            onChange={(event) => setNote(event.currentTarget.value)}
            autosize
            minRows={2}
            maxLength={ISSUE_MAX_NOTE_LENGTH}
          />
          <Group justify="flex-end">
            <Button
              variant="subtle"
              onClick={() => setActionOpen(false)}
              disabled={sending}
            >
              Cancel
            </Button>
            <Button loading={sending} onClick={() => void confirmAction()}>
              {action === "withdraw" ? "Withdraw" : "Reopen"}
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}

type ShowFilter = "open" | "closed" | "all";

export default function IssueReportListPage() {
  const authState = useAuthGuard([grants.LOGON]);
  const moduleOn = useProfileStore(
    (state) => state.profile?.issueReportsEnabled,
  );
  const { reportList, listLoaded, readReportList, openDrawer } =
    useIssueReportStore();
  const [searchParams, setSearchParams] = useSearchParams();
  const [show, setShow] = useState<ShowFilter>("open");
  const [loadError, setLoadError] = useState<string | null>(null);
  const hasFetched = useRef(false);
  const selectedId = Number(searchParams.get("issueReportId")) || null;

  const refresh = useCallback(() => {
    readReportList()
      .then(() => setLoadError(null))
      .catch((err: unknown) => {
        reportError(err, "issue report list");
        setLoadError(errorText(err, "The reports could not be loaded"));
      });
  }, [readReportList]);

  useEffect(() => {
    if (authState === "YES" && moduleOn && !hasFetched.current) {
      hasFetched.current = true;
      refresh();
    }
  }, [authState, moduleOn, refresh]);

  //============================^ HOOK LINE ^====================

  if (authState === "CHECKING" || moduleOn === undefined) {
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
  if (!moduleOn) {
    return (
      <Container size={1280} px={32} py={32} mx="auto">
        <Title order={1} mb="lg">
          Issue reports
        </Title>
        <Alert color="gray">Issue reports are not enabled in this app.</Alert>
      </Container>
    );
  }

  const shownList = reportList.filter((report) =>
    show === "all"
      ? true
      : show === "open"
        ? !issueIsClosed(report.state)
        : issueIsClosed(report.state),
  );

  return (
    <Container size={1280} px={32} py={32} mx="auto">
      <Group justify="space-between" mb="lg">
        <Title order={1}>Issue reports</Title>
        <Button leftSection={<IconPlus size={16} />} onClick={openDrawer}>
          Report an issue
        </Button>
      </Group>
      <Text size="sm" c="dimmed" mb="md">
        Bugs, polish and feature requests from your workspace, and the
        support team&apos;s replies.
      </Text>
      {loadError && (
        <Alert color="red" mb="md">
          {loadError}
        </Alert>
      )}
      <Group align="flex-start" gap="lg" wrap="wrap">
        <Stack gap="xs" w={340} maw="100%" style={{ flex: "none" }}>
          <SegmentedControl
            size="xs"
            value={show}
            onChange={(next) => setShow(next as ShowFilter)}
            data={[
              { value: "open", label: "Open" },
              { value: "closed", label: "Closed" },
              { value: "all", label: "All" },
            ]}
          />
          {!listLoaded && !loadError && <Loader size="sm" />}
          {listLoaded && shownList.length === 0 && (
            <Text size="sm" c="dimmed">
              {show === "open" ? "No open reports." : "Nothing here."}
            </Text>
          )}
          {shownList.map((report) => (
            <UnstyledButton
              key={report.issueReportId}
              onClick={() =>
                setSearchParams({
                  issueReportId: String(report.issueReportId),
                })
              }
            >
              <Paper
                withBorder
                p="sm"
                style={{
                  borderColor:
                    report.issueReportId === selectedId
                      ? "var(--mantine-primary-color-filled)"
                      : undefined,
                }}
              >
                <Group gap={6} mb={4}>
                  <Badge
                    size="xs"
                    color={issueKindColor(report.kind)}
                    variant="light"
                  >
                    {issueKindLabel(report.kind)}
                  </Badge>
                  <Badge
                    size="xs"
                    color={stateColor(report.state)}
                    variant="outline"
                  >
                    {report.status}
                  </Badge>
                </Group>
                <Text
                  size="sm"
                  fw={500}
                  lineClamp={2}
                  style={{ overflowWrap: "anywhere" }}
                >
                  {report.summary}
                </Text>
                <Text size="xs" c="dimmed" mt={2}>
                  {report.reporterName} · {formatWhen(report.lastActivityTs)}
                  {report.messageCount > 0
                    ? ` · ${report.messageCount} ` +
                      (report.messageCount === 1 ? "message" : "messages")
                    : ""}
                </Text>
              </Paper>
            </UnstyledButton>
          ))}
        </Stack>
        <Box style={{ flex: 1, minWidth: 280 }}>
          {selectedId ? (
            <ReportDetail issueReportId={selectedId} onChanged={refresh} />
          ) : (
            <Paper withBorder p="xl">
              <Text c="dimmed" size="sm">
                Pick a report to see its conversation, or report a new one.
              </Text>
            </Paper>
          )}
        </Box>
      </Group>
    </Container>
  );
}
