import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MantineProvider } from "@mantine/core";
import { MfaExemptField } from "../components/mfa-exempt-field";

// B097 follow-up: the e2e API runs with ALLOW_MFA_EXEMPT_VIA_API=true, so
// the refused cases (the checkbox hidden, or kept only to clear a stored
// exemption) are pinned here.

describe("MfaExemptField", () => {
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

  const renderField = (
    allowSetting: boolean,
    storedExempt: boolean,
    checked: boolean,
    onChange: (checked: boolean) => void = () => {},
  ) =>
    act(() =>
      root.render(
        <MantineProvider>
          <MfaExemptField
            allowSetting={allowSetting}
            storedExempt={storedExempt}
            checked={checked}
            onChange={onChange}
          />
        </MantineProvider>,
      ),
    );

  const checkbox = () =>
    container.querySelector<HTMLInputElement>('input[type="checkbox"]');
  const helpIcon = () => container.querySelector('[role="img"]');
  // MantineProvider adds <style> tags; the label is what a user reads.
  const labelText = () => container.querySelector("label")?.textContent;

  it("is the plain checkbox when the server allows setting it", () => {
    const onChange = vi.fn();
    renderField(true, false, false, onChange);
    expect(checkbox()).not.toBeNull();
    expect(checkbox()!.checked).toBe(false);
    expect(labelText()).toBe("MFA Exempt");
    expect(helpIcon()).toBeNull();
    act(() => checkbox()!.click());
    expect(onChange).toHaveBeenCalledWith(true);
  });

  it("is hidden when refused and the login is not exempt", () => {
    renderField(false, false, false);
    expect(checkbox()).toBeNull();
    expect(container.querySelector("label")).toBeNull();
  });

  it("shows a stored exemption when refused, and lets it be cleared", () => {
    const onChange = vi.fn();
    renderField(false, true, true, onChange);
    expect(checkbox()!.checked).toBe(true);
    expect(labelText()).toContain("MFA Exempt");
    expect(helpIcon()?.getAttribute("aria-label")).toBe("About MFA Exempt");
    act(() => checkbox()!.click());
    expect(onChange).toHaveBeenCalledWith(false);
  });

  it("shows a stored exemption as checked when allowed", () => {
    renderField(true, true, true);
    expect(checkbox()!.checked).toBe(true);
  });
});
