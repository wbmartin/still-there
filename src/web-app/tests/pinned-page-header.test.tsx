import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, useRef } from "react";
import { createRoot, type Root } from "react-dom/client";
import {
  PAGE_HEADER_HEIGHT_VARIABLE,
  usePinnedPageHeader,
} from "../hooks/use-pinned-page-header";

// The pinned header itself is CSS (index.css, checked by the e2e
// app-shell spec). What can break unseen is the measurement a page's
// sticky rows sit under: the hook must find the same block the CSS pins and keep its
// height current as pages change.

// jsdom has no layout and no ResizeObserver: heights come from a stub keyed
// on the element's data-height, and the observer only needs to exist.
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}

function Page({ children }: { children: React.ReactNode }) {
  const pageRef = useRef<HTMLDivElement>(null);
  usePinnedPageHeader(pageRef);
  return (
    <div className="app-page" ref={pageRef}>
      {children}
    </div>
  );
}

const ListPage = () => (
  <div>
    <div data-height="47.2">
      <h1>Examples</h1>
      <button>Add Example</button>
    </div>
    <table data-height="900" />
  </div>
);

const NoTitlePage = () => (
  <div>
    <p data-height="20">Nothing to pin</p>
  </div>
);

describe("usePinnedPageHeader", () => {
  let container: HTMLDivElement;
  let root: Root;
  const pageHeight = () =>
    container
      .querySelector<HTMLElement>(".app-page")!
      .style.getPropertyValue(PAGE_HEADER_HEIGHT_VARIABLE);

  beforeEach(() => {
    vi.stubGlobal("ResizeObserver", ResizeObserverStub);
    vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
    vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(
      function (this: Element) {
        const height = Number(this.getAttribute("data-height") ?? 0);
        return { height } as DOMRect;
      },
    );
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("writes the title block's height, rounded up", () => {
    act(() => root.render(<Page><ListPage /></Page>));
    expect(pageHeight()).toBe("48px");
  });

  it("re-measures when the routed page changes", async () => {
    act(() => root.render(<Page><ListPage /></Page>));
    expect(pageHeight()).toBe("48px");

    act(() => root.render(<Page><NoTitlePage /></Page>));
    // MutationObserver callbacks run as a microtask after the commit.
    await act(async () => {});
    expect(pageHeight()).toBe("0px");

    act(() => root.render(<Page><ListPage /></Page>));
    await act(async () => {});
    expect(pageHeight()).toBe("48px");
  });

  it("matches a bare <h1> as the block (edit pages)", () => {
    act(() =>
      root.render(
        <Page>
          <div>
            <h1 data-height="40">Edit Example</h1>
            <form data-height="600" />
          </div>
        </Page>,
      ),
    );
    expect(pageHeight()).toBe("40px");
  });
});
