import { create } from "zustand";
import type { ApiResponse } from "../types/api";
import type {
  IssueAction,
  IssueKind,
  IssueReport,
  IssueReportDetail,
} from "../types/issue-report-type";
import { apiJson, jsonBody } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";
import { registerCacheClearer } from "../utils/cache-utils";

// The issue-report module (doc/150-issue-report-module.md): the tenant's
// reports and the Report drawer's open state. The drawer is global, opened
// from the header on any page. Body keys are the Go form field names and
// json tags (handler/issue_report_handler.go, model/issue_report_model.go).

export interface NewIssueReport {
  kind: IssueKind;
  summary: string;
  detail: string;
  fileList: File[];
}

/**
 * Where the report was made from, to help reproduce it. `site` (the host)
 * comes first: a UAT and a production install can report to the same
 * tracker, and it is how triage tells them apart.
 */
export const captureIssueContext = (): string =>
  JSON.stringify({
    site: window.location.host,
    page_url: window.location.pathname + window.location.search,
    browser: navigator.userAgent,
    viewport: `${window.innerWidth}x${window.innerHeight}`,
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    language: navigator.language,
  });

/**
 * The URL that serves one attachment. Used as an <img src> or a download
 * link, so the browser fetches it with the session cookie; it is a GET, so
 * no CSRF header is needed, and it is not an API call through apiJson.
 */
export const issueAttachmentUrl = (issueReportAttachmentId: number): string =>
  `/api${API_ENDPOINTS.ISSUE_REPORT_ATTACHMENT}` +
  `?issueReportAttachmentId=${issueReportAttachmentId}`;

const appendFileList = (form: FormData, fileList: File[]) => {
  for (const file of fileList) form.append("files", file, file.name);
};

// The report, moved to the top of the list (it was just active).
const withReport = (
  reportList: IssueReport[],
  detail: IssueReportDetail,
): IssueReport[] =>
  [
    detail,
    ...reportList.filter(
      (report) => report.issueReportId !== detail.issueReportId,
    ),
  ].sort((left, right) => right.lastActivityTs - left.lastActivityTs);

interface IssueReportStore {
  drawerOpen: boolean;
  openDrawer: () => void;
  closeDrawer: () => void;

  reportList: IssueReport[];
  listLoaded: boolean;
  readReportList: () => Promise<IssueReport[]>;
  readReport: (issueReportId: number) => Promise<IssueReportDetail>;
  createReport: (report: NewIssueReport) => Promise<IssueReportDetail>;
  createMessage: (
    issueReportId: number,
    body: string,
    fileList: File[],
  ) => Promise<IssueReportDetail>;
  createAction: (
    issueReportId: number,
    action: IssueAction,
    note: string,
  ) => Promise<IssueReportDetail>;
  clear: () => void;
}

export const useIssueReportStore = create<IssueReportStore>((set, get) => ({
  drawerOpen: false,
  openDrawer: () => set({ drawerOpen: true }),
  closeDrawer: () => set({ drawerOpen: false }),

  reportList: [],
  listLoaded: false,

  readReportList: async () => {
    const response = await apiJson<ApiResponse<IssueReport[]>>(
      API_ENDPOINTS.ISSUE_REPORT_LIST,
    );
    set({ reportList: response.data, listLoaded: true });
    return response.data;
  },

  readReport: async (issueReportId) => {
    const response = await apiJson<ApiResponse<IssueReportDetail>>(
      `${API_ENDPOINTS.ISSUE_REPORT}?issueReportId=${issueReportId}`,
    );
    return response.data;
  },

  createReport: async ({ kind, summary, detail, fileList }) => {
    const form = new FormData();
    form.set("kind", kind);
    form.set("summary", summary);
    if (detail.trim()) form.set("detail", detail);
    form.set("pageUrl", window.location.pathname + window.location.search);
    form.set("context", captureIssueContext());
    appendFileList(form, fileList);
    // No Content-Type: the browser writes the multipart boundary.
    const response = await apiJson<ApiResponse<IssueReportDetail>>(
      API_ENDPOINTS.ISSUE_REPORT,
      { method: "POST", body: form },
    );
    set({ reportList: withReport(get().reportList, response.data) });
    return response.data;
  },

  createMessage: async (issueReportId, body, fileList) => {
    const form = new FormData();
    form.set("issueReportId", String(issueReportId));
    form.set("body", body);
    appendFileList(form, fileList);
    const response = await apiJson<ApiResponse<IssueReportDetail>>(
      API_ENDPOINTS.ISSUE_REPORT_MESSAGE,
      { method: "POST", body: form },
    );
    set({ reportList: withReport(get().reportList, response.data) });
    return response.data;
  },

  createAction: async (issueReportId, action, note) => {
    const response = await apiJson<ApiResponse<IssueReportDetail>>(
      API_ENDPOINTS.ISSUE_REPORT_ACTION,
      { method: "POST", ...jsonBody({ issueReportId, action, note }) },
    );
    set({ reportList: withReport(get().reportList, response.data) });
    return response.data;
  },

  clear: () => set({ drawerOpen: false, reportList: [], listLoaded: false }),
}));

registerCacheClearer(() => useIssueReportStore.getState().clear());
