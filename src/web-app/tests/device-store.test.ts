import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useDeviceStore } from "../stores/device-store";
import { useBeaconStore } from "../stores/beacon-store";
import { useContactStore } from "../stores/contact-store";
import type { Beacon, Device } from "../types/device-type";
import type { Contact } from "../types/contact-type";

// The device, band and contact stores (README "Entity Stores"): each
// request's URL and body against the json tags in
// src/api-srv/internal/model/{device,contact}_model.go, which Go's decoder
// would otherwise drop silently (B013).

const stamps = {
  securityContext: 2,
  createTs: 100,
  createUserId: 1,
  updateTs: 200,
  updateUserId: 1,
};

const device: Device = {
  deviceId: 5,
  deviceName: "Kitchen",
  checkinIntervalSec: 60,
  missedCheckinBeforeAlarm: 3,
  absentAfterSec: 300,
  configVersion: 1,
  ...stamps,
};

const beacon: Beacon = {
  beaconId: 8,
  deviceId: 5,
  address: "AA:BB:CC:DD:EE:FF",
  label: "Grandma",
  ...stamps,
};

const contact: Contact = {
  contactId: 9,
  deviceId: 5,
  contactName: "Daughter",
  phoneE164: "+17048609088",
  contactStatus: "invited",
  notifyEventList: ["raised"],
  ...stamps,
};

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

const sentBody = (init?: RequestInit): unknown =>
  JSON.parse(init?.body as string);

describe("device store requests", () => {
  beforeEach(() => {
    useDeviceStore.getState().clear();
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("readDeviceById reads the list filtered to one id", async () => {
    const fetchMock = recordFetch([device]);

    const loaded = await useDeviceStore.getState().readDeviceById(5);

    expect(fetchMock.mock.calls[0][0]).toBe("/api/deviceList?deviceIdList=5");
    expect(loaded).toEqual(device);
  });

  it("readDeviceById answers null for an empty list", async () => {
    recordFetch([]);
    expect(await useDeviceStore.getState().readDeviceById(5)).toBeNull();
  });

  it("createItem posts the writable fields by their json tags", async () => {
    const fetchMock = recordFetch(device);
    const data = {
      deviceName: "Kitchen",
      checkinIntervalSec: 60,
      missedCheckinBeforeAlarm: 3,
      absentAfterSec: 300,
    };

    await useDeviceStore.getState().createItem(data);

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/device");
    expect(init?.method).toBe("POST");
    expect(sentBody(init)).toEqual(data);
  });

  it("issueDeviceToken posts deviceId and updateTs and moves the lock",
    async () => {
      useDeviceStore.setState({ itemList: [device] });
      const issued = {
        deviceId: 5,
        deviceToken: "ab".repeat(32),
        tokenIssuedTs: 300,
        updateTs: 300,
      };
      const fetchMock = recordFetch(issued);

      const answer = await useDeviceStore
        .getState()
        .issueDeviceToken(5, 200);

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/deviceToken");
      expect(init?.method).toBe("POST");
      expect(sentBody(init)).toEqual({ deviceId: 5, updateTs: 200 });
      expect(answer).toEqual(issued);
      expect(useDeviceStore.getState().itemList[0]).toMatchObject({
        updateTs: 300,
        tokenIssuedTs: 300,
      });
    });

  it("deleteItem sends deviceId and updateTs as query parameters",
    async () => {
      const fetchMock = recordFetch({ deviceId: 5 });

      await useDeviceStore.getState().deleteItem(5, 200);

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/device?deviceId=5&updateTs=200");
      expect(init?.method).toBe("DELETE");
    });
});

describe("band store requests", () => {
  beforeEach(() => {
    useBeaconStore.getState().clear();
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("readList for one device sends ?deviceId=", async () => {
    const fetchMock = recordFetch([beacon]);

    await useBeaconStore.getState().readList({ queryParams: { deviceId: 5 } });

    expect(fetchMock.mock.calls[0][0]).toBe("/api/beaconList?deviceId=5");
    expect(useBeaconStore.getState().itemList).toEqual([beacon]);
  });

  it("readBeaconById reads the list filtered to one id", async () => {
    const fetchMock = recordFetch([beacon]);

    expect(await useBeaconStore.getState().readBeaconById(8)).toEqual(beacon);
    expect(fetchMock.mock.calls[0][0]).toBe("/api/beaconList?beaconIdList=8");
  });

  it("updateItem puts beaconId, address, label and updateTs", async () => {
    const fetchMock = recordFetch(beacon);
    const data = {
      beaconId: 8,
      address: "AA:BB:CC:DD:EE:FF",
      label: "Grandma",
      updateTs: 200,
    };

    await useBeaconStore.getState().updateItem(data);

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/beacon");
    expect(init?.method).toBe("PUT");
    expect(sentBody(init)).toEqual(data);
  });
});

describe("contact store requests", () => {
  beforeEach(() => {
    useContactStore.getState().clear();
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("readContactById reads the list filtered to one id", async () => {
    const fetchMock = recordFetch([contact]);

    expect(await useContactStore.getState().readContactById(9)).toEqual(
      contact,
    );
    expect(fetchMock.mock.calls[0][0]).toBe(
      "/api/contactList?contactIdList=9",
    );
  });

  it("createItem posts deviceId, contactName, phoneE164, notifyEventList",
    async () => {
      const fetchMock = recordFetch(contact);
      const data = {
        deviceId: 5,
        contactName: "Daughter",
        phoneE164: "+17048609088",
        notifyEventList: ["raised", "reminder"],
      };

      await useContactStore.getState().createItem(data);

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/contact");
      expect(init?.method).toBe("POST");
      expect(sentBody(init)).toEqual(data);
    });
});
