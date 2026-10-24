// Mirrors AlertModel in src/api-srv/internal/model/alert_model.go. beaconId
// is absent for a device_offline alarm (the base station itself).
export type AlertState = "raised" | "acknowledged" | "turned_off" | "resolved";

export interface Alert {
  alertId: number;
  deviceId: number;
  beaconId?: number;
  alertKind: string;
  alertState: AlertState;
  raisedTs: number;
  lastNotifiedTs?: number;
  notifyCount: number;
  ackTs?: number;
  ackBy?: string;
  ackVia?: string;
  offTs?: number;
  offBy?: string;
  offVia?: string;
  resolvedTs?: number;
  securityContext: number;
  createTs: number;
  createUserId: number;
  updateTs: number;
  updateUserId: number;
}

// The watchdog's kinds (src/api-srv/internal/watchdog/watchdog.go).
export const ALERT_KIND_LABEL: Record<string, string> = {
  beacon_absent: "Band absent",
  device_offline: "Base station offline",
};

export const ALERT_STATE_LABEL: Record<AlertState, string> = {
  raised: "Raised",
  acknowledged: "Acknowledged",
  turned_off: "Turned off",
  resolved: "Resolved",
};

// An alarm still texting someone: raised or acknowledged.
export const isOpenAlert = (alert: Alert) =>
  alert.alertState === "raised" || alert.alertState === "acknowledged";
