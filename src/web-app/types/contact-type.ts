// Mirrors ContactModel in src/api-srv/internal/model/contact_model.go.
// contactStatus and the invite / opt timestamps are the watchdog's: the API
// never takes them from a client.
export type ContactStatus = "invited" | "active" | "opted_out";

export interface Contact {
  contactId: number;
  deviceId: number;
  contactName: string;
  phoneE164: string;
  contactStatus: ContactStatus;
  notifyEventList: string[];
  inviteSentTs?: number;
  optedInTs?: number;
  optedOutTs?: number;
  securityContext: number;
  createTs: number;
  createUserId: number;
  updateTs: number;
  updateUserId: number;
  deleteTs?: number;
  deleteUserId?: number;
}

// ContactNotifyEventList in the Go model, in its order, with the label the
// edit page shows.
export const CONTACT_NOTIFY_EVENT_LIST = [
  { value: "raised", label: "Alarm raised" },
  { value: "reminder", label: "Reminder while unacknowledged" },
  { value: "acknowledged", label: "Someone acknowledged" },
  { value: "turned_off", label: "Someone turned the alarm off" },
  { value: "reactivated", label: "Alerting re-activated (band seen again)" },
  { value: "daily_reminder", label: "Daily reminder while still open" },
];
