import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter } from "react-router-dom";
import { MantineProvider } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import IssueReportDrawer from "../components/issue-report-drawer";
import { useIssueReportStore } from "../stores/issue-report-store";

// The Report drawer: closed, it renders nothing; open, a send with no
// summary asks for one and calls nothing; a send with one posts the report
// and closes the drawer.

vi.mock("@mantine/notifications", () => ({
  notifications: { show: vi.fn() },
}));

const json = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });

// React reads a controlled input's value through the native setter.
const typeInto = (input: HTMLInputElement, value: string) => {
  Object.getOwnPropertyDescriptor(
    HTMLInputElement.prototype,
    "value",
  )?.set?.call(input, value);
  input.dispatchEvent(new Event("input", { bubbles: true }));
};

// jsdom has no ResizeObserver; SegmentedControl's indicator asks for one.
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}

const buttonNamed = (name: string) =>
  Array.from(document.body.querySelectorAll("button")).find(
    (button) => button.textContent === name,
  );

describe("IssueReportDrawer", () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
    vi.stubGlobal("matchMedia", () => ({
      matches: false,
      addEventListener() {},
      removeEventListener() {},
    }));
    vi.stubGlobal("ResizeObserver", ResizeObserverStub);
    useIssueReportStore.getState().clear();
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    vi.unstubAllGlobals();
    vi.mocked(notifications.show).mockClear();
  });

  const render = () =>
    act(() =>
      root.render(
        <MantineProvider>
          <MemoryRouter>
            <IssueReportDrawer />
          </MemoryRouter>
        </MantineProvider>,
      ),
    );

  it("renders nothing while closed", () => {
    render();
    expect(document.body.textContent).not.toContain("Report an issue");
  });

  it("asks for a summary, then sends the report", async () => {
    const fetchMock = vi
      .fn<(input: string, init?: RequestInit) => Promise<Response>>()
      .mockResolvedValue(
        json(201, {
          data: {
            issueReportId: 12,
            kind: "bug",
            summary: "Save does nothing",
            lastActivityTs: 5,
            attachmentList: [],
            threadList: [],
          },
        }),
      );
    vi.stubGlobal("fetch", fetchMock);
    render();
    // The drawer mounts after its open transition.
    await act(async () => {
      useIssueReportStore.getState().openDrawer();
      await new Promise((resolve) => setTimeout(resolve, 400));
    });
    expect(document.body.textContent).toContain("Report an issue");
    expect(document.body.textContent).toContain("Capture screen");

    await act(async () => {
      buttonNamed("Send report")!.click();
      await Promise.resolve();
    });
    expect(fetchMock).not.toHaveBeenCalled();
    expect(vi.mocked(notifications.show).mock.calls[0][0].title).toBe(
      "Add a summary",
    );

    const summary = document.body.querySelector<HTMLInputElement>(
      "input[placeholder^='e.g.']",
    );
    expect(summary).not.toBeNull();
    act(() => typeInto(summary!, "Save does nothing"));
    await act(async () => {
      buttonNamed("Send report")!.click();
      await new Promise((resolve) => setTimeout(resolve, 0));
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/issueReport");
    expect((init?.body as FormData).get("summary")).toBe(
      "Save does nothing",
    );
    expect(useIssueReportStore.getState().drawerOpen).toBe(false);
    expect(useIssueReportStore.getState().reportList[0].issueReportId).toBe(
      12,
    );
  });
});
