import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter } from "react-router-dom";
import { MantineProvider } from "@mantine/core";
import HelpPanel from "../help/help-panel";
import Term from "../help/help-term";
import FieldHelp from "../help/field-help";
import { latestWhatsNewId } from "../help/help-registry";
import { ROUTES } from "../constants/routes";
import { grants } from "../constants/grants";
import { useAuthStore } from "../stores/auth-store";
import { useHelpStore } from "../stores/help-store";
import { useProfileStore } from "../stores/profile-store";
import type { Profile } from "../types/profile-type";

// The Help panel and the inline pieces, rendered: the page's help, search,
// an entry, the "What's new" marker, and nothing at all with help off.

const profileOf = (helpEnabled: boolean): Profile => ({
  appUserId: 7,
  userName: "alice",
  firstName: "Alice",
  lastName: "Tester",
  email: "alice@example.com",
  securityContext: 2,
  preferenceJsonb: {},
  updateTs: 10,
  allowMfaExemptViaApi: false,
  issueReportsEnabled: false,
  helpEnabled,
});

describe("help UI", () => {
  let container: HTMLDivElement;
  let root: Root;
  let requestList: { url: string; body: unknown }[];

  beforeEach(() => {
    vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
    vi.stubGlobal("matchMedia", () => ({
      matches: false,
      addEventListener() {},
      removeEventListener() {},
    }));
    // jsdom has neither; the drawer's scroll area and popovers use them.
    vi.stubGlobal(
      "ResizeObserver",
      class {
        observe() {}
        unobserve() {}
        disconnect() {}
      },
    );
    vi.useFakeTimers({ shouldAdvanceTime: true });
    requestList = [];
    vi.stubGlobal(
      "fetch",
      vi.fn((url: string, init?: RequestInit) => {
        const body = init?.body
          ? (JSON.parse(init.body as string) as Record<string, unknown>)
          : undefined;
        requestList.push({ url, body });
        const data =
          url === "/api/me"
            ? { ...profileOf(true), ...body, updateTs: 11 }
            : { recorded: true };
        return Promise.resolve(
          new Response(JSON.stringify({ data }), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        );
      }),
    );
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    useHelpStore.getState().clear();
    useAuthStore.setState({ grantList: [grants.LOGON] });
    useProfileStore.setState({ profile: profileOf(true) });
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    useProfileStore.getState().clear();
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  const renderAt = async (route: string, element: React.ReactNode) => {
    await act(async () => {
      root.render(
        <MantineProvider>
          <MemoryRouter initialEntries={[route]}>{element}</MemoryRouter>
        </MantineProvider>,
      );
      await vi.advanceTimersByTimeAsync(500);
    });
  };

  const bodyText = () => document.body.textContent ?? "";

  const typeSearch = async (text: string) => {
    const input = document.body.querySelector<HTMLInputElement>(
      'input[aria-label="Search help"]',
    );
    expect(input).not.toBeNull();
    await act(async () => {
      // The prototype's setter, so React's change tracking sees a change.
      Reflect.set(HTMLInputElement.prototype, "value", text, input);
      input?.dispatchEvent(new Event("input", { bubbles: true }));
      await vi.advanceTimersByTimeAsync(2000);
    });
  };

  it("opens on the page's own help, and marks what's new seen", async () => {
    act(() => useHelpStore.getState().openHelp(null));
    await renderAt(ROUTES.EXAMPLE_LIST, <HelpPanel />);
    expect(bodyText()).toContain("On this page");
    expect(bodyText()).toContain("Every example in this security context");
    expect(bodyText()).toContain("Show me around this page");
    expect(bodyText()).toContain("What’s new");
    expect(bodyText()).not.toContain("Help insights:");
    const put = requestList.find((request) => request.url === "/api/me");
    expect(put?.body).toMatchObject({
      preferenceJsonb: { help: { whatsNewSeen: latestWhatsNewId() } },
    });
  });

  it("links Help insights for a user admin", async () => {
    useAuthStore.setState({ grantList: [grants.LOGON, grants.READ_USER] });
    act(() => useHelpStore.getState().openHelp(null));
    await renderAt(ROUTES.DASHBOARD, <HelpPanel />);
    expect(bodyText()).toContain("Help insights:");
  });

  it("searches, and records a search that finds nothing", async () => {
    act(() => useHelpStore.getState().openHelp(null));
    await renderAt(ROUTES.DASHBOARD, <HelpPanel />);
    await typeSearch("invite");
    expect(bodyText()).toContain("Invite link");
    expect(requestList.some((r) => r.url === "/api/helpEvent")).toBe(false);

    await typeSearch("quarterly zebra");
    expect(bodyText()).toContain("Nothing in the help matches");
    expect(requestList).toContainEqual({
      url: "/api/helpEvent",
      body: {
        event: "search_no_result",
        route: ROUTES.DASHBOARD,
        query: "quarterly zebra",
      },
    });
  });

  it("shows an entry and records the view", async () => {
    act(() => useHelpStore.getState().openHelp("mfa"));
    await renderAt(ROUTES.LOGIN, <HelpPanel />);
    expect(bodyText()).toContain("Multi-factor authentication");
    expect(requestList).toContainEqual({
      url: "/api/helpEvent",
      body: { event: "entry_view", route: ROUTES.LOGIN, entryId: "mfa" },
    });
  });

  it("offers no Ask box unless an app passes one", async () => {
    act(() => useHelpStore.getState().openHelp(null));
    await renderAt(
      ROUTES.DASHBOARD,
      <HelpPanel askSlot={(query) => <p>Ask about {query}</p>} />,
    );
    await typeSearch("invite");
    expect(bodyText()).toContain("Ask about invite");
  });

  it("renders Term as plain text and FieldHelp as nothing, help off", async () => {
    useProfileStore.setState({ profile: profileOf(false) });
    await renderAt(
      ROUTES.DASHBOARD,
      <p>
        Your <Term id="security-context">context</Term>
        <FieldHelp id="mfa" />
      </p>,
    );
    // MantineProvider adds <style> tags; the paragraph is the content.
    expect(container.querySelector("p")?.outerHTML).toBe(
      "<p>Your context</p>",
    );
  });

  it("renders Term as a focusable explained word with help on", async () => {
    await renderAt(
      ROUTES.DASHBOARD,
      <p>
        Your <Term id="security-context">context</Term>
        <FieldHelp id="mfa" />
      </p>,
    );
    expect(
      container.querySelector('[aria-label="Security context: what is this?"]'),
    ).not.toBeNull();
    expect(container.querySelector('[aria-label="Help: MFA"]')).not.toBeNull();
  });
});
