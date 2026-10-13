import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter } from "react-router-dom";
import { MantineProvider } from "@mantine/core";
import {
  helpLinkIdList,
  parseHelpInline,
  parseHelpMarkdown,
  routeLinkList,
} from "../help/help-markdown";
import HelpMarkdownView from "../help/help-markdown-view";
import { useHelpStore } from "../stores/help-store";

// The help markdown subset: what it parses, and that nothing in a body
// becomes markup or a link off the app.

describe("parseHelpMarkdown", () => {
  it("parses paragraphs, bullets and numbered lists", () => {
    const blockList = parseHelpMarkdown(
      "One line\nand its next line.\n\n- first\n- second\n\n1. a\n2. b",
    );
    expect(blockList.map((block) => block.kind)).toEqual([
      "paragraph",
      "bullet-list",
      "number-list",
    ]);
    expect(blockList[0]).toEqual({
      kind: "paragraph",
      inlineList: [{ kind: "text", text: "One line and its next line." }],
    });
  });

  it("joins an indented line to the list item before it", () => {
    const [block] = parseHelpMarkdown(
      "- a long item\n  that wraps\n- the next",
    );
    expect(block).toEqual({
      kind: "bullet-list",
      itemList: [
        [{ kind: "text", text: "a long item that wraps" }],
        [{ kind: "text", text: "the next" }],
      ],
    });
  });

  it("reads bold, help links and route links", () => {
    expect(
      parseHelpInline("See **this**, [roles](help:grant) and [it](/x?a=1)."),
    ).toEqual([
      { kind: "text", text: "See " },
      { kind: "bold", text: "this" },
      { kind: "text", text: ", " },
      { kind: "help-link", label: "roles", id: "grant" },
      { kind: "text", text: " and " },
      { kind: "route-link", label: "it", to: "/x?a=1" },
      { kind: "text", text: "." },
    ]);
  });

  it("leaves a link off the app, or a malformed one, as text", () => {
    for (const text of [
      "[x](https://example.com)",
      "[x](//example.com/path)",
      "[x](javascript:alert(1))",
      "[x](help:Bad_Id)",
    ]) {
      expect(parseHelpInline(text), text).toEqual([{ kind: "text", text }]);
    }
  });

  it("lists a body's help and route links for the gates", () => {
    const body = "[a](help:mfa) and\n\n- [b](/login)\n- [c](help:grant)";
    expect(helpLinkIdList(body)).toEqual(["mfa", "grant"]);
    expect(routeLinkList(body)).toEqual(["/login"]);
  });

  it("is nothing for an empty body", () => {
    expect(parseHelpMarkdown("  \n ")).toEqual([]);
  });
});

describe("HelpMarkdownView", () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
    vi.stubGlobal("matchMedia", () => ({
      matches: false,
      addEventListener() {},
      removeEventListener() {},
    }));
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    useHelpStore.getState().clear();
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    vi.unstubAllGlobals();
  });

  const render = (body: string) =>
    act(() =>
      root.render(
        <MantineProvider>
          <MemoryRouter>
            <HelpMarkdownView body={body} />
          </MemoryRouter>
        </MantineProvider>,
      ),
    );

  it("shows markup in a body as text, never as elements", () => {
    render('<script>alert(1)</script> <img src=x onerror="alert(2)">');
    expect(container.querySelector("script")).toBeNull();
    expect(container.querySelector("img")).toBeNull();
    expect(container.textContent).toContain("<script>alert(1)</script>");
  });

  it("renders a route link as an in-app link and a help link as a button", () => {
    render("Go [home](/dashboard) or read [roles](help:grant).");
    const link = container.querySelector("a");
    expect(link?.getAttribute("href")).toBe("/dashboard");
    const button = [...container.querySelectorAll("button")].find(
      (element) => element.textContent === "roles",
    );
    expect(button).toBeDefined();
    act(() => button?.click());
    expect(useHelpStore.getState()).toMatchObject({
      opened: true,
      entryId: "grant",
    });
  });
});
