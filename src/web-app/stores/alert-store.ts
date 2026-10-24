import { create } from "zustand";
import type { ApiResponse } from "../types/api";
import type { Alert } from "../types/alert-type";
import { apiJson, isApiStatus, jsonBody } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";
import { registerCacheClearer } from "../utils/cache-utils";

// Alarms. Not an entity store: GET /alertList has no afterUpdateTs (it
// answers the open alarms and the last week's resolved ones, in that
// order), and alarms are never created or deleted through the API. The
// list page re-reads it on a timer.
interface AlertStore {
  alertList: Alert[];
  // A read has answered; a refresh does not drop back to a loader.
  loaded: boolean;
  error: string | null;
  readAlertList: (deviceId?: number) => Promise<void>;
  // POST /alertAcknowledge and /alertTurnOff {alertId, updateTs}; the
  // changed row replaces the cached one. 409 when it is no longer in a
  // state the action takes or someone changed it first.
  acknowledgeAlert: (alert: Alert) => Promise<Alert>;
  turnOffAlert: (alert: Alert) => Promise<Alert>;
  // POST /smsSimulateInbound {fromPhone, body}: the reply that would have
  // been texted back. The endpoint exists only under SMS_PROVIDER=log.
  simulateInboundText: (fromPhone: string, body: string) => Promise<string>;
  // Whether the simulator is on: an empty body is refused 400 by a live
  // endpoint before it acts, and answered 404 by one that is off.
  probeSimulateInbound: () => Promise<boolean>;
  clear: () => void;
}

export const useAlertStore = create<AlertStore>((set, get) => {
  const change = async (endpoint: string, alert: Alert) => {
    const { data: changed } = await apiJson<ApiResponse<Alert>>(endpoint, {
      method: "POST",
      ...jsonBody({ alertId: alert.alertId, updateTs: alert.updateTs }),
    });
    set({
      alertList: get().alertList.map((a) =>
        a.alertId === changed.alertId ? changed : a,
      ),
    });
    return changed;
  };

  return {
    alertList: [],
    loaded: false,
    error: null,

    readAlertList: async (deviceId?: number) => {
      const query = deviceId ? `?deviceId=${deviceId}` : "";
      try {
        const body = await apiJson<ApiResponse<Alert[]> | undefined>(
          `${API_ENDPOINTS.ALERT_LIST}${query}`,
        );
        set({ alertList: body?.data ?? [], loaded: true, error: null });
      } catch (err) {
        set({
          loaded: true,
          error: err instanceof Error ? err.message : "Failed to load",
        });
        throw err;
      }
    },

    acknowledgeAlert: (alert: Alert) =>
      change(API_ENDPOINTS.ALERT_ACKNOWLEDGE, alert),
    turnOffAlert: (alert: Alert) => change(API_ENDPOINTS.ALERT_TURN_OFF, alert),

    simulateInboundText: async (fromPhone: string, body: string) => {
      const { data } = await apiJson<ApiResponse<{ reply: string }>>(
        API_ENDPOINTS.SMS_SIMULATE_INBOUND,
        { method: "POST", ...jsonBody({ fromPhone, body }) },
      );
      return data.reply;
    },

    probeSimulateInbound: async () => {
      try {
        await apiJson<unknown>(API_ENDPOINTS.SMS_SIMULATE_INBOUND, {
          method: "POST",
          ...jsonBody({ fromPhone: "", body: "" }),
        });
        return true;
      } catch (err) {
        if (isApiStatus(err, 400)) return true;
        if (isApiStatus(err, 404)) return false;
        throw err;
      }
    },

    clear: () => set({ alertList: [], loaded: false, error: null }),
  };
});

registerCacheClearer(() => useAlertStore.getState().clear());
