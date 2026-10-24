import type { Device, DeviceTokenIssued } from "../types/device-type";
import type { ApiResponse } from "../types/api";
import { apiJson, jsonBody } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";
import { createEntityStore } from "./create-entity-store";

interface DeviceExtras {
  // There is no GET /device; one device is read through the list endpoint
  // filtered to its id. Returns null when it does not exist.
  readDeviceById: (id: number) => Promise<Device | null>;
  // POST /deviceToken {deviceId, updateTs}. The answer holds the only copy
  // of the token; the device's previous token stops working.
  issueDeviceToken: (
    deviceId: number,
    updateTs: number,
  ) => Promise<DeviceTokenIssued>;
}

export const useDeviceStore = createEntityStore<Device, DeviceExtras>(
  {
    idParamName: "deviceId",
    idField: "deviceId",
    listEndpoint: API_ENDPOINTS.DEVICE_LIST,
    itemEndpoint: API_ENDPOINTS.DEVICE,
  },
  (set, get) => ({
    readDeviceById: async (id: number) => {
      const searchParams = new URLSearchParams({ deviceIdList: String(id) });
      const body = await apiJson<ApiResponse<Device[]> | null | undefined>(
        `${API_ENDPOINTS.DEVICE_LIST}?${searchParams.toString()}`,
      );
      const device = body?.data?.[0];
      if (!device) return null;
      // Refresh the cached row if it is present; do not add it otherwise, or
      // the list would grow without a matching maxUpdateTs.
      set({
        itemList: get().itemList.map((d) =>
          d.deviceId === device.deviceId ? device : d,
        ),
      });
      return device;
    },
    issueDeviceToken: async (deviceId: number, updateTs: number) => {
      const { data: issued } = await apiJson<ApiResponse<DeviceTokenIssued>>(
        API_ENDPOINTS.DEVICE_TOKEN,
        { method: "POST", ...jsonBody({ deviceId, updateTs }) },
      );
      // The token write moves the row's lock and issue time.
      set({
        itemList: get().itemList.map((d) =>
          d.deviceId === deviceId
            ? {
                ...d,
                updateTs: issued.updateTs,
                tokenIssuedTs: issued.tokenIssuedTs,
              }
            : d,
        ),
      });
      return issued;
    },
  }),
);
