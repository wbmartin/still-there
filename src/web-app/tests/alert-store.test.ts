import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useAlertStore } from "../stores/alert-store";
import { useTenantSettingStore } from "../stores/tenant-setting-store";
import type { Alert } from "../types/alert-type";
import type { TenantSetting } from "../types/tenant-setting-type";

// The alert and tenant-setting stores: URLs and bodies against the json
// tags in src/api-srv/internal/model/{alert,tenant_setting}_model.go and
// sms_handler.go, and the statuses each one turns into an answer.

const alert: Alert = {
  alertId: 3,
  deviceId: 5,
  beaconId: 8,
  alertKind: "beacon_absent",
  alertState: "raised",
  raisedTs: 100,
  notifyCount: 1,
  securityContext: 2,
  createTs: 100,
  createUserId: 0,
  updateTs: 150,
  updateUserId: 0,
};

const setting: TenantSetting = {
  tenantSettingId: 1,
  timeZone: "America/Chicago",
  remindIntervalMin: 30,
  dailyReminderTime: "08:15",
  securityContext: 2,
  createTs: 100,
  createUserId: 1,
  updateTs: 200,
  updateUserId: 1,
};

const answer = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });

const recordFetch = (status: number, body: unknown) => {
  const fetchMock = vi.fn<
    (input: string, init?: RequestInit) => Promise<Response>
  >(() => Promise.resolve(answer(status, body)));
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
};

const sentBody = (init?: RequestInit): unknown =>
  JSON.parse(init?.body as string);

describe("alert store requests", () => {
  beforeEach(() => {
    useAlertStore.getState().clear();
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("readAlertList sends ?deviceId= only when given", async () => {
    const fetchMock = recordFetch(200, { data: [alert] });

    await useAlertStore.getState().readAlertList();
    await useAlertStore.getState().readAlertList(5);

    expect(fetchMock.mock.calls[0][0]).toBe("/api/alertList");
    expect(fetchMock.mock.calls[1][0]).toBe("/api/alertList?deviceId=5");
    expect(useAlertStore.getState()).toMatchObject({
      alertList: [alert],
      loaded: true,
      error: null,
    });
  });

  it("acknowledgeAlert posts alertId and updateTs and stores the answer",
    async () => {
      useAlertStore.setState({ alertList: [alert] });
      const acknowledged = { ...alert, alertState: "acknowledged" };
      const fetchMock = recordFetch(200, { data: acknowledged });

      await useAlertStore.getState().acknowledgeAlert(alert);

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/alertAcknowledge");
      expect(init?.method).toBe("POST");
      expect(sentBody(init)).toEqual({ alertId: 3, updateTs: 150 });
      expect(useAlertStore.getState().alertList).toEqual([acknowledged]);
    });

  it("turnOffAlert throws the API's 409 and keeps the row", async () => {
    useAlertStore.setState({ alertList: [alert] });
    const fetchMock = recordFetch(409, {
      error: "This alert is resolved and cannot be turned off",
    });

    await expect(
      useAlertStore.getState().turnOffAlert(alert),
    ).rejects.toMatchObject({
      status: 409,
      message: "This alert is resolved and cannot be turned off",
    });
    expect(fetchMock.mock.calls[0][0]).toBe("/api/alertTurnOff");
    expect(useAlertStore.getState().alertList).toEqual([alert]);
  });

  it("simulateInboundText posts fromPhone and body, answers the reply",
    async () => {
      const fetchMock = recordFetch(200, { data: { reply: "Thanks" } });

      const reply = await useAlertStore
        .getState()
        .simulateInboundText("+17048609088", "OK");

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/smsSimulateInbound");
      expect(sentBody(init)).toEqual({ fromPhone: "+17048609088", body: "OK" });
      expect(reply).toBe("Thanks");
    });

  it("probeSimulateInbound: 400 is on, 404 is off, 403 throws", async () => {
    recordFetch(400, { error: "fromPhone (E.164) and body are required" });
    expect(await useAlertStore.getState().probeSimulateInbound()).toBe(true);

    recordFetch(404, { error: "Not found" });
    expect(await useAlertStore.getState().probeSimulateInbound()).toBe(false);

    recordFetch(403, { error: "Forbidden" });
    await expect(
      useAlertStore.getState().probeSimulateInbound(),
    ).rejects.toMatchObject({ status: 403 });
  });
});

describe("tenant setting store requests", () => {
  beforeEach(() => {
    useTenantSettingStore.getState().clear();
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("readTenantSetting answers null for the 404 of a tenant never saved",
    async () => {
      recordFetch(404, { error: "Not found" });
      expect(await useTenantSettingStore.getState().readTenantSetting())
        .toBeNull();
    });

  it("readTenantSetting answers and keeps the saved row", async () => {
    const fetchMock = recordFetch(200, { data: setting });

    expect(await useTenantSettingStore.getState().readTenantSetting())
      .toEqual(setting);
    expect(fetchMock.mock.calls[0][0]).toBe("/api/tenantSetting");
    expect(useTenantSettingStore.getState().tenantSetting).toEqual(setting);
  });

  it("saveTenantSetting puts the json tags", async () => {
    const fetchMock = recordFetch(200, { data: setting });
    const update = {
      timeZone: "America/Chicago",
      remindIntervalMin: 30,
      dailyReminderTime: "08:15",
      updateTs: 0,
    };

    await useTenantSettingStore.getState().saveTenantSetting(update);

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/tenantSetting");
    expect(init?.method).toBe("PUT");
    expect(sentBody(init)).toEqual(update);
  });
});
