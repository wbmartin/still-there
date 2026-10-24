import type { Beacon } from "../types/device-type";
import type { ApiResponse } from "../types/api";
import { apiJson } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";
import { createEntityStore } from "./create-entity-store";

// A device's bands. readList takes { queryParams: { deviceId } } for one
// device's; a page showing one device filters itemList by deviceId too,
// since the cache may hold another read's rows.
interface BeaconExtras {
  // There is no GET /beacon; read through the list filtered to one id.
  // Returns null when it does not exist.
  readBeaconById: (id: number) => Promise<Beacon | null>;
}

export const useBeaconStore = createEntityStore<Beacon, BeaconExtras>(
  {
    idParamName: "beaconId",
    idField: "beaconId",
    listEndpoint: API_ENDPOINTS.BEACON_LIST,
    itemEndpoint: API_ENDPOINTS.BEACON,
  },
  (set, get) => ({
    readBeaconById: async (id: number) => {
      const searchParams = new URLSearchParams({ beaconIdList: String(id) });
      const body = await apiJson<ApiResponse<Beacon[]> | null | undefined>(
        `${API_ENDPOINTS.BEACON_LIST}?${searchParams.toString()}`,
      );
      const beacon = body?.data?.[0];
      if (!beacon) return null;
      set({
        itemList: get().itemList.map((b) =>
          b.beaconId === beacon.beaconId ? beacon : b,
        ),
      });
      return beacon;
    },
  }),
);
