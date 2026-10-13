// The issue-report module (doc/150-issue-report-module.md). Mirrors the
// json tags of src/api-srv/internal/model/issue_report_model.go by hand.

export type IssueKind = "bug" | "polish" | "feature";

export const issueKindOptionList: {
  value: IssueKind;
  label: string;
  hint: string;
}[] = [
  { value: "bug", label: "Bug", hint: "Something is broken or wrong" },
  {
    value: "polish",
    label: "Polish",
    hint: "Works, but could look or read better",
  },
  { value: "feature", label: "Feature", hint: "Something new you would like" },
];

export const issueKindLabel = (kind: string): string =>
  issueKindOptionList.find((option) => option.value === kind)?.label ?? kind;

export const issueKindColor = (kind: string): string =>
  kind === "bug" ? "red" : kind === "polish" ? "yellow" : "teal";

export interface IssueAttachment {
  issueReportAttachmentId: number;
  issueReportId: number;
  issueReportMessageId?: number;
  filename: string;
  contentType: string;
  sizeBytes: number;
  createTs: number;
}

export interface IssueMessage {
  issueReportMessageId: number;
  issueReportId: number;
  authorKind: "reporter" | "staff";
  authorName?: string;
  body: string;
  action?: "withdraw" | "reopen";
  relayState: string;
  createTs: number;
  attachmentList: IssueAttachment[];
}

export interface IssueReport {
  issueReportId: number;
  securityContext: number;
  kind: IssueKind;
  summary: string;
  detail?: string;
  pageUrl?: string;
  state: string;
  status: string;
  statusNote?: string;
  reporterName: string;
  relayState: "pending" | "sent" | "failed" | "local";
  lastSyncedTs?: number;
  lastActivityTs: number;
  messageCount: number;
  createTs: number;
  createUserId: number;
  updateTs: number;
  updateUserId: number;
}

export interface IssueReportDetail extends IssueReport {
  attachmentList: IssueAttachment[];
  threadList: IssueMessage[];
}

export type IssueAction = "withdraw" | "reopen";

/** Closed or won't fix: the reporter may reopen it, not withdraw it. */
export const issueIsClosed = (state: string): boolean =>
  state === "closed" || state === "wont_fix";

// The API's limits (model.IssueMax*): five files, 10 MB each, 18 MB in all.
export const ISSUE_MAX_FILE_COUNT = 5;
export const ISSUE_MAX_FILE_BYTES = 10 * 1024 * 1024;
export const ISSUE_MAX_TOTAL_BYTES = 18 * 1024 * 1024;
export const ISSUE_MAX_SUMMARY_LENGTH = 500;
export const ISSUE_MAX_DETAIL_LENGTH = 20000;
export const ISSUE_MAX_NOTE_LENGTH = 2000;

export const issueFileExtensionList = [
  ".png",
  ".jpg",
  ".jpeg",
  ".gif",
  ".webp",
  ".pdf",
  ".txt",
  ".log",
  ".csv",
];
/** An example page (a mock-up) may go on polish and feature, never bug. */
export const issueHtmlExtensionList = [".html", ".htm"];

export const fileExtensionOf = (name: string): string => {
  const dot = name.lastIndexOf(".");
  return dot < 0 ? "" : name.slice(dot).toLowerCase();
};

/**
 * Why a file cannot join a report's attachments, or null when it can. The
 * API checks again, from the bytes; this saves the user a refused send.
 */
export const issueFileProblem = (
  file: File,
  attachedList: File[],
  allowHtml: boolean,
): string | null => {
  const extension = fileExtensionOf(file.name);
  const isHtml = issueHtmlExtensionList.includes(extension);
  if (!issueFileExtensionList.includes(extension) && !(allowHtml && isHtml)) {
    return isHtml
      ? `${file.name}: an HTML page may go on a polish or feature report, ` +
          "not on a bug."
      : `${file.name}: images, PDF, TXT, LOG or CSV only` +
          `${allowHtml ? ", or an HTML page" : ""}.`;
  }
  if (file.size > ISSUE_MAX_FILE_BYTES) {
    return `${file.name} is over 10 MB.`;
  }
  if (attachedList.length >= ISSUE_MAX_FILE_COUNT) {
    return `Attach at most ${ISSUE_MAX_FILE_COUNT} files.`;
  }
  const attachedBytes = attachedList.reduce(
    (total, attached) => total + attached.size,
    0,
  );
  if (attachedBytes + file.size > ISSUE_MAX_TOTAL_BYTES) {
    return "Keep the files under 18 MB altogether.";
  }
  return null;
};
