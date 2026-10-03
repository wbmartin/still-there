import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MantineProvider, Table } from "@mantine/core";
import { SortTh, type SortDirection } from "../components/sort-th";
import { HelpTip, LabelWithHelp } from "../components/help-tip";

// SortTh and LabelWithHelp are the shared leaves of the list and edit
// pages. What a page copy can break unseen: a header that is not a button
// (keyboard users cannot sort), aria-sort on the wrong column, and help
// text that no longer reaches a screen reader.

describe("SortTh and HelpTip", () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
    // jsdom has no matchMedia; MantineProvider reads the colour scheme.
    vi.stubGlobal("matchMedia", () => ({
      matches: false,
      addEventListener() {},
      removeEventListener() {},
    }));
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    vi.unstubAllGlobals();
  });

  const renderHeader = (
    sortColumn: string,
    sortDirection: SortDirection,
    onSort: (column: string) => void,
  ) =>
    act(() =>
      root.render(
        <MantineProvider>
          <Table>
            <Table.Thead>
              <Table.Tr>
                <SortTh
                  label="Username"
                  column="userName"
                  sortColumn={sortColumn}
                  sortDirection={sortDirection}
                  onSort={onSort}
                />
                <SortTh
                  label="Email"
                  column="email"
                  sortColumn={sortColumn}
                  sortDirection={sortDirection}
                  onSort={onSort}
                />
              </Table.Tr>
            </Table.Thead>
          </Table>
        </MantineProvider>,
      ),
    );

  const headerList = () => [...container.querySelectorAll("th")];

  it("puts the label in a button that calls onSort", () => {
    const onSort = vi.fn();
    renderHeader("userName", "asc", onSort);
    const button = headerList()[1].querySelector("button");
    // A <button> is focusable and Enter/Space activate it natively; the
    // e2e entity-pages spec presses Enter in a real browser.
    expect(button?.textContent).toBe("Email");
    expect(button?.type).toBe("button");
    act(() => button!.click());
    expect(onSort).toHaveBeenCalledWith("email");
  });

  it("marks only the sorted column with aria-sort", () => {
    renderHeader("userName", "asc", () => {});
    expect(headerList()[0].getAttribute("aria-sort")).toBe("ascending");
    expect(headerList()[1].hasAttribute("aria-sort")).toBe(false);

    renderHeader("email", "desc", () => {});
    expect(headerList()[0].hasAttribute("aria-sort")).toBe(false);
    expect(headerList()[1].getAttribute("aria-sort")).toBe("descending");
  });

  it("gives the help icon a name and focus", () => {
    act(() =>
      root.render(
        <MantineProvider>
          <HelpTip help="Shown on hover" about="JSON" />
        </MantineProvider>,
      ),
    );
    const icon = container.querySelector('[role="img"]');
    expect(icon?.getAttribute("aria-label")).toBe("About JSON");
    expect(icon?.getAttribute("tabindex")).toBe("0");
  });

  it("renders a bare label when there is no help", () => {
    act(() =>
      root.render(<LabelWithHelp label="UUID" />),
    );
    expect(container.innerHTML).toBe("UUID");
  });
});
