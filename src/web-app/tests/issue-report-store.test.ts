import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  captureIssueContext,
  issueAttachmentUrl,
  useIssueReportStore,
} from "../stores/issue-report-store";
import {
  issueFileProblem,
  issueIsClosed,
  type IssueReportDetail,
} from "../types/issue-report-type";
import { clearAllCaches } from "../utils/cache-utils";
import { clearCsrfToken, setCsrfToken } from "../utils/api";

// The issue-report store's requests, against the Go side's names: the
// form fields CreateIssueReport and CreateIssueReportMessage read
// (handler/issue_report_handler.go) and the json tags of
// IssueReportActionRequest (model/issue_report_model.go). Go drops a form
// field or a body key it does not know, so a misspelt one fails nowhere
// but here.

const detail = (
  issueReportId: number,
  lastActivityTs: number,
): IssueReportDetail => ({
  issueReportId,
  securityContext: 2,
  kind: "bug",
  summary: `report ${issueReportId}`,
  state: "open",
  status: "Received",
  reporterName: "Alice Example",
  relayState: "local",
  lastActivityTs,
  messageCount: 0,
  createTs: 1,
  createUserId: 7,
  updateTs: lastActivityTs,
  updateUserId: 7,
  attachmentList: [],
  threadList: [],
});

const recordFetch = (data: unknown) => {
  const fetchMock = vi.fn<
    (input: string, init?: RequestInit) => Promise<Response>
  >(() =>
    Promise.resolve(
      new Response(JSON.stringify({ data }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    ),
  );
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
};

describe("issue report store requests", () => {
  beforeEach(() => {
    useIssueReportStore.getState().clear();
    setCsrfToken("abc");
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    clearCsrfToken();
  });

  it("readReportList reads /issueReportList into the store", async () => {
    const fetchMock = recordFetch([detail(1, 10)]);
    await useIssueReportStore.getState().readReportList();
    expect(fetchMock.mock.calls[0][0]).toBe("/api/issueReportList");
    const state = useIssueReportStore.getState();
    expect(state.listLoaded).toBe(true);
    expect(state.reportList.map((report) => report.issueReportId)).toEqual(
      [1],
    );
  });

  it("readReport reads ?issueReportId=", async () => {
    const fetchMock = recordFetch(detail(4, 10));
    const loaded = await useIssueReportStore.getState().readReport(4);
    expect(fetchMock.mock.calls[0][0]).toBe(
      "/api/issueReport?issueReportId=4",
    );
    expect(loaded.issueReportId).toBe(4);
  });

  it("createReport posts the form fields the handler reads", async () => {
    useIssueReportStore.setState({ reportList: [detail(1, 10)] });
    const fetchMock = recordFetch(detail(2, 20));
    const shot = new File(["png"], "shot.png", { type: "image/png" });

    await useIssueReportStore.getState().createReport({
      kind: "bug",
      summary: "Save does nothing",
      detail: "Steps",
      fileList: [shot],
    });

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/issueReport");
    expect(init?.method).toBe("POST");
    // The browser writes the multipart boundary: no Content-Type set.
    const headers = new Headers(init?.headers);
    expect(headers.get("Content-Type")).toBeNull();
    expect(headers.get("X-CSRF-Token")).toBe("abc");
    const form = init?.body as FormData;
    expect([...new Set(form.keys())].sort()).toEqual(
      ["context", "detail", "files", "kind", "pageUrl", "summary"],
    );
    expect(form.get("kind")).toBe("bug");
    expect(form.get("summary")).toBe("Save does nothing");
    expect((form.get("files") as File).name).toBe("shot.png");
    expect(JSON.parse(form.get("context") as string)).toHaveProperty(
      "page_url",
    );
    // The new report leads the list (it is the most recently active).
    expect(
      useIssueReportStore.getState().reportList.map((r) => r.issueReportId),
    ).toEqual([2, 1]);
  });

  it("createReport leaves an empty description out", async () => {
    const fetchMock = recordFetch(detail(2, 20));
    await useIssueReportStore.getState().createReport({
      kind: "feature",
      summary: "Export",
      detail: "   ",
      fileList: [],
    });
    const form = fetchMock.mock.calls[0][1]?.body as FormData;
    expect(form.has("detail")).toBe(false);
    expect(form.has("files")).toBe(false);
  });

  it("createMessage posts issueReportId, body and files", async () => {
    const fetchMock = recordFetch(detail(3, 30));
    const note = new File(["words"], "notes.txt", { type: "text/plain" });
    await useIssueReportStore.getState().createMessage(3, "More", [note]);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/issueReportMessage");
    const form = init?.body as FormData;
    expect([...new Set(form.keys())].sort()).toEqual(
      ["body", "files", "issueReportId"],
    );
    expect(form.get("issueReportId")).toBe("3");
  });

  it("createAction posts the IssueReportActionRequest json tags",
    async () => {
      const fetchMock = recordFetch(detail(3, 30));
      await useIssueReportStore
        .getState()
        .createAction(3, "withdraw", "Found it");
      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/issueReportAction");
      expect(JSON.parse(init?.body as string)).toEqual({
        issueReportId: 3,
        action: "withdraw",
        note: "Found it",
      });
    });

  it("is cleared with every other cache on sign-out", () => {
    useIssueReportStore.setState({
      drawerOpen: true,
      reportList: [detail(1, 10)],
      listLoaded: true,
    });
    clearAllCaches();
    const state = useIssueReportStore.getState();
    expect(state.drawerOpen).toBe(false);
    expect(state.reportList).toEqual([]);
    expect(state.listLoaded).toBe(false);
  });
});

describe("issue report helpers", () => {
  it("issueAttachmentUrl is the attachment route", () => {
    expect(issueAttachmentUrl(8)).toBe(
      "/api/issueReportAttachment?issueReportAttachmentId=8",
    );
  });

  it("captureIssueContext names the site first, then the page", () => {
    const context = JSON.parse(captureIssueContext()) as Record<
      string,
      unknown
    >;
    expect(Object.keys(context)[0]).toBe("site");
    expect(context.site).toBe(window.location.host);
    expect(typeof context.page_url).toBe("string");
    expect(typeof context.browser).toBe("string");
  });

  it("issueIsClosed is closed or won't fix", () => {
    expect(issueIsClosed("closed")).toBe(true);
    expect(issueIsClosed("wont_fix")).toBe(true);
    expect(issueIsClosed("reopened")).toBe(false);
  });

  it("issueFileProblem mirrors the API's limits", () => {
    const file = (name: string, size = 3) =>
      new File([new Uint8Array(size)], name);
    expect(issueFileProblem(file("shot.png"), [], false)).toBeNull();
    expect(issueFileProblem(file("SHOT.PNG"), [], false)).toBeNull();
    expect(issueFileProblem(file("run.exe"), [], false)).toMatch(
      /images, PDF/,
    );
    expect(issueFileProblem(file("mockup.html"), [], false)).toMatch(
      /not on a bug/,
    );
    expect(issueFileProblem(file("mockup.html"), [], true)).toBeNull();
    expect(
      issueFileProblem(file("big.pdf", 10 * 1024 * 1024 + 1), [], false),
    ).toMatch(/over 10 MB/);
    const fiveList = Array.from({ length: 5 }, (_, at) => file(`${at}.txt`));
    expect(issueFileProblem(file("six.txt"), fiveList, false)).toMatch(
      /at most 5/,
    );
    const twoBigList = [
      file("a.pdf", 9 * 1024 * 1024),
      file("b.pdf", 9 * 1024 * 1024),
    ];
    expect(issueFileProblem(file("c.txt", 10), twoBigList, false)).toMatch(
      /18 MB/,
    );
  });
});

describe("captureIssueContext", () => {
  it("says what the reporter's screen and tab looked like", () => {
    const context = JSON.parse(captureIssueContext()) as Record<
      string,
      unknown
    >;
    for (const key of [
      "screen",
      "zoom_estimate_pct",
      "device_pixel_ratio",
      "touch",
      "app_version",
      "app_built_at",
      "tab_loaded_at",
      "tab_age_minutes",
    ]) {
      expect(context).toHaveProperty(key);
    }
    expect(typeof context.app_version).toBe("string");
  });

  it("reports how long ago the tab loaded the app", () => {
    // A tab left open for three days.
    const later = Date.now() + 3 * 24 * 60 * 60 * 1000;
    const context = JSON.parse(captureIssueContext(later)) as {
      tab_age_minutes: number;
    };
    expect(context.tab_age_minutes).toBeGreaterThanOrEqual(3 * 24 * 60);
  });
});

