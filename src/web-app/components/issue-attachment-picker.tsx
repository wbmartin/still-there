import { useEffect, useState, type DragEvent } from "react";
import {
  ActionIcon,
  Button,
  FileButton,
  Group,
  Image,
  Paper,
  Stack,
  Text,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconCamera, IconPaperclip, IconX } from "@tabler/icons-react";
import {
  ISSUE_MAX_FILE_COUNT,
  issueFileExtensionList,
  issueFileProblem,
  issueHtmlExtensionList,
} from "../types/issue-report-type";

const formatBytes = (byteCount: number): string =>
  byteCount < 1024 * 1024
    ? `${Math.max(1, Math.round(byteCount / 1024))} KB`
    : `${(byteCount / 1024 / 1024).toFixed(1)} MB`;

/**
 * A picture of this tab's page as it is now, without the report drawer,
 * notifications or anything marked `data-capture-ignore`. Only this app's
 * page, so the browser asks no permission. html-to-image loads on first
 * use, so it stays out of the main bundle.
 */
const capturePage = async (): Promise<File> => {
  const { toBlob } = await import("html-to-image");
  const keep = (node: HTMLElement) =>
    !(
      node.classList?.contains("mantine-Drawer-root") ||
      node.classList?.contains("mantine-Notifications-root") ||
      node.dataset?.captureIgnore !== undefined
    );
  const blob = await toBlob(document.body, {
    filter: keep,
    // Web fonts may be cross-origin; the picture uses system fonts.
    skipFonts: true,
    pixelRatio: Math.min(window.devicePixelRatio || 1, 2),
    width: window.innerWidth,
    height: window.innerHeight,
    backgroundColor:
      getComputedStyle(document.body).backgroundColor || "#ffffff",
    style: {
      transform: `translate(${-window.scrollX}px, ${-window.scrollY}px)`,
    },
  });
  if (!blob) throw new Error("The page could not be drawn.");
  const stamp = new Date().toISOString().replace(/[:.]/g, "-").slice(0, 19);
  return new File([blob], `screenshot-${stamp}.png`, { type: "image/png" });
};

// A data URL read from the file: nothing to revoke, so a remount (React's
// StrictMode runs effects twice in development) cannot leave a dead URL.
function Thumbnail({ file }: { file: File }) {
  const [loaded, setLoaded] = useState<{ file: File; url: string }>();
  useEffect(() => {
    const reader = new FileReader();
    reader.onload = () => {
      if (typeof reader.result === "string") {
        setLoaded({ file, url: reader.result });
      }
    };
    reader.readAsDataURL(file);
    return () => reader.abort();
  }, [file]);
  return loaded?.file === file ? (
    <Image src={loaded.url} w={56} h={36} fit="cover" radius="xs" alt="" />
  ) : null;
}

/** Files on a report or reply: pick, drop, or capture this page. */
export function IssueAttachmentPicker({
  fileList,
  onChange,
  allowHtml = false,
}: {
  fileList: File[];
  onChange: (fileList: File[]) => void;
  /** Polish and feature reports may carry an example HTML page. */
  allowHtml?: boolean;
}) {
  const [capturing, setCapturing] = useState(false);
  const [dragging, setDragging] = useState(false);
  const full = fileList.length >= ISSUE_MAX_FILE_COUNT;

  const add = (incomingList: File[]) => {
    const nextList = [...fileList];
    for (const file of incomingList) {
      const problem = issueFileProblem(file, nextList, allowHtml);
      if (problem) {
        notifications.show({
          color: "orange",
          title: "Not attached",
          message: problem,
        });
        continue;
      }
      nextList.push(file);
    }
    onChange(nextList);
  };

  const capture = async () => {
    setCapturing(true);
    try {
      add([await capturePage()]);
    } catch (err) {
      notifications.show({
        color: "red",
        title: "Could not capture the screen",
        message:
          err instanceof Error
            ? err.message
            : "Take a screenshot with your computer and attach it.",
      });
    } finally {
      setCapturing(false);
    }
  };

  const onDrop = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    setDragging(false);
    add(Array.from(event.dataTransfer.files));
  };

  const accept = [
    ...issueFileExtensionList,
    ...(allowHtml ? issueHtmlExtensionList : []),
  ].join(",");

  return (
    <Stack gap="xs">
      <Paper
        withBorder
        p="sm"
        style={{
          borderStyle: "dashed",
          background: dragging
            ? "var(--mantine-color-default-hover)"
            : undefined,
        }}
        onDragOver={(event) => {
          event.preventDefault();
          setDragging(true);
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={onDrop}
      >
        <Group gap="xs" justify="center">
          <Button
            variant="light"
            size="xs"
            leftSection={<IconCamera size={14} />}
            loading={capturing}
            disabled={full}
            onClick={() => void capture()}
          >
            Capture screen
          </Button>
          <FileButton onChange={add} accept={accept} multiple>
            {(props) => (
              <Button
                {...props}
                variant="default"
                size="xs"
                leftSection={<IconPaperclip size={14} />}
                disabled={full}
              >
                Attach files
              </Button>
            )}
          </FileButton>
          <Text size="xs" c="dimmed">
            or drop them here. Up to {ISSUE_MAX_FILE_COUNT}, 10 MB each:
            images, PDF, TXT, LOG, CSV
            {allowHtml ? ", or an example HTML page" : ""}.
          </Text>
        </Group>
      </Paper>
      {fileList.length > 0 && (
        <Stack gap={6}>
          {fileList.map((file, index) => (
            <Paper key={`${file.name}-${index}`} withBorder p={6}>
              <Group gap="sm" wrap="nowrap" justify="space-between">
                <Group gap="sm" wrap="nowrap" style={{ minWidth: 0 }}>
                  {file.type.startsWith("image/") && (
                    <Thumbnail file={file} />
                  )}
                  <Text size="sm" truncate>
                    {file.name}
                  </Text>
                  <Text size="xs" c="dimmed" style={{ flex: "none" }}>
                    {formatBytes(file.size)}
                  </Text>
                </Group>
                <ActionIcon
                  variant="subtle"
                  color="gray"
                  aria-label={`Remove ${file.name}`}
                  onClick={() =>
                    onChange(fileList.filter((_, at) => at !== index))
                  }
                >
                  <IconX size={14} />
                </ActionIcon>
              </Group>
            </Paper>
          ))}
        </Stack>
      )}
    </Stack>
  );
}
