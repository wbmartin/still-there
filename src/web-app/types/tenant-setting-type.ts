// Mirrors TenantSettingModel in
// src/api-srv/internal/model/tenant_setting_model.go: one row per tenant,
// absent (GET answers 404) until the first save.
export interface TenantSetting {
  tenantSettingId: number;
  // IANA name.
  timeZone: string;
  remindIntervalMin: number;
  // HH:MM in timeZone.
  dailyReminderTime: string;
  securityContext: number;
  createTs: number;
  createUserId: number;
  updateTs: number;
  updateUserId: number;
}
