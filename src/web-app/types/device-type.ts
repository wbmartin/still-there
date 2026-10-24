// Mirrors DeviceModel, BeaconModel and DeviceTokenIssuedModel in
// src/api-srv/internal/model/device_model.go. The runtime fields (last
// check-in, token, a band's presence) are written by the Pi's check-ins and
// are read-only here.
export interface Device {
  deviceId: number;
  deviceName: string;
  checkinIntervalSec: number;
  missedCheckinBeforeAlarm: number;
  absentAfterSec: number;
  configVersion: number;
  tokenIssuedTs?: number;
  lastCheckinTs?: number;
  lastCheckinJson?: unknown;
  lastAgentVersion?: string;
  securityContext: number;
  createTs: number;
  createUserId: number;
  updateTs: number;
  updateUserId: number;
  deleteTs?: number;
  deleteUserId?: number;
}

// POST /deviceToken: the token is in this answer and nowhere else.
export interface DeviceTokenIssued {
  deviceId: number;
  deviceToken: string;
  tokenIssuedTs: number;
  updateTs: number;
}

export interface Beacon {
  beaconId: number;
  deviceId: number;
  // Upper-case MAC, AA:BB:CC:DD:EE:FF.
  address: string;
  label: string;
  // The Pi's last report; absent before any.
  present?: boolean;
  lastSeenTs?: number;
  lastRssi?: number;
  lastReportTs?: number;
  securityContext: number;
  createTs: number;
  createUserId: number;
  updateTs: number;
  updateUserId: number;
  deleteTs?: number;
  deleteUserId?: number;
}
