import { useState } from "react";
import {
  Button,
  Drawer,
  Group,
  SegmentedControl,
  Stack,
  Text,
  TextInput,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useNavigate } from "react-router-dom";
import { useIssueReportStore } from "../stores/issue-report-store";
import {
  fileExtensionOf,
  ISSUE_MAX_DETAIL_LENGTH,
  ISSUE_MAX_SUMMARY_LENGTH,
  issueHtmlExtensionList,
  issueKindOptionList,
  type IssueKind,
} from "../types/issue-report-type";
import { ROUTES } from "../constants/routes";
import { MarkdownEditor } from "./markdown-editor";
import { IssueAttachmentPicker } from "./issue-attachment-picker";

/**
 * "Report an issue" (the issue-report module): Bug / Polish / Feature, a
 * summary, a markdown description and files, a capture of this page among
 * them. Opened from the header on any page; the page's address and the
 * browser go along to help reproduce it.
 */
export default function IssueReportDrawer() {
  const { drawerOpen, closeDrawer, createReport } = useIssueReportStore();
  const navigate = useNavigate();
  const [kind, setKind] = useState<IssueKind>("bug");
  const [summary, setSummary] = useState("");
  const [detail, setDetail] = useState("");
  const [fileList, setFileList] = useState<File[]>([]);
  const [saving, setSaving] = useState(false);

  const reset = () => {
    setKind("bug");
    setSummary("");
    setDetail("");
    setFileList([]);
  };

  const changeKind = (next: IssueKind) => {
    setKind(next);
    if (next !== "bug") return;
    // An HTML page is for polish and feature only.
    const keptList = fileList.filter(
      (file) => !issueHtmlExtensionList.includes(fileExtensionOf(file.name)),
    );
    if (keptList.length < fileList.length) {
      setFileList(keptList);
      notifications.show({
        color: "orange",
        title: "HTML page removed",
        message: "An HTML page may go on a polish or feature report.",
      });
    }
  };

  const submit = async () => {
    if (!summary.trim()) {
      notifications.show({
        color: "orange",
        title: "Add a summary",
        message: "One line saying what you found or would like.",
      });
      return;
    }
    setSaving(true);
    try {
      const created = await createReport({
        kind,
        summary: summary.trim(),
        detail,
        fileList,
      });
      reset();
      closeDrawer();
      notifications.show({
        color: "green",
        title: "Report sent",
        message: "Thank you. Follow it, and any replies, in Issue reports.",
      });
      void navigate(
        `${ROUTES.ISSUE_REPORT_LIST}?issueReportId=${created.issueReportId}`,
      );
    } catch (err) {
      notifications.show({
        color: "red",
        title: "The report was not sent",
        message: err instanceof Error ? err.message : "Unknown error",
      });
    } finally {
      setSaving(false);
    }
  };

  const hint = issueKindOptionList.find(
    (option) => option.value === kind,
  )?.hint;

  return (
    <Drawer
      opened={drawerOpen}
      onClose={() => (saving ? undefined : closeDrawer())}
      title="Report an issue"
      position="right"
      size="lg"
    >
      <Stack gap="md">
        <div>
          <SegmentedControl
            fullWidth
            value={kind}
            onChange={(next) => changeKind(next as IssueKind)}
            data={issueKindOptionList.map((option) => ({
              value: option.value,
              label: option.label,
            }))}
          />
          <Text size="xs" c="dimmed" mt={4}>
            {hint}
          </Text>
        </div>
        <TextInput
          label="Summary"
          required
          maxLength={ISSUE_MAX_SUMMARY_LENGTH}
          placeholder={
            kind === "feature"
              ? "e.g. Export the list to a spreadsheet"
              : "e.g. Save does nothing on the profile page"
          }
          value={summary}
          onChange={(event) => setSummary(event.currentTarget.value)}
        />
        <MarkdownEditor
          label="Description"
          value={detail}
          onChange={setDetail}
          maxLength={ISSUE_MAX_DETAIL_LENGTH}
          placeholder={
            kind === "bug"
              ? "What you did, what you expected, what happened instead."
              : "What would help, and why."
          }
        />
        <div>
          <Text size="sm" fw={500} mb={4}>
            Screenshots and files
          </Text>
          <IssueAttachmentPicker
            fileList={fileList}
            onChange={setFileList}
            allowHtml={kind !== "bug"}
          />
        </div>
        <Text size="xs" c="dimmed">
          The page you are on and your browser go with the report, to help
          reproduce it. Everyone in your workspace can see reports.
        </Text>
        <Group justify="flex-end">
          <Button variant="default" onClick={closeDrawer} disabled={saving}>
            Cancel
          </Button>
          <Button loading={saving} onClick={() => void submit()}>
            Send report
          </Button>
        </Group>
      </Stack>
    </Drawer>
  );
}
