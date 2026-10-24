import { z } from "zod";
import { isE164, normalisePhone } from "../utils/phone";

// Form schemas, used through mantine-form-zod-resolver's zod4Resolver
// (zodResolver expects zod 3 and throws on a zod 4 failure). The template's
// entity edit pages validate inline; the monitoring pages (device, band,
// contact, tenant setting) use these. Each mirrors its Go *Problem check in
// src/api-srv/internal/model/, so a form refuses what the API would.
export const loginSchema = z.object({
  username: z.string().min(1, 'Username is required'),
  password: z.string().min(1, 'Password is required'),
});

export type LoginFormData = z.infer<typeof loginSchema>;

// A whole number of at least `min`; Mantine's NumberInput holds "" when
// empty, which fails as "Required".
const wholeNumberAtLeast = (min: number, message: string) =>
  z.number({ error: "Required" }).int("Whole numbers only").min(min, message);

// DeviceProblem (device_model.go).
export const deviceSchema = z.object({
  deviceName: z.string().trim().min(1, "Name is required"),
  checkinIntervalSec: wholeNumberAtLeast(10, "At least 10 seconds"),
  missedCheckinBeforeAlarm: wholeNumberAtLeast(1, "At least 1"),
  absentAfterSec: wholeNumberAtLeast(10, "At least 10 seconds"),
});

export type DeviceFormData = z.infer<typeof deviceSchema>;

const BEACON_ADDRESS_PATTERN = /^[0-9A-F]{2}(:[0-9A-F]{2}){5}$/;

// NormaliseBeaconAddress (device_model.go): trimmed and upper-cased.
export const normaliseBeaconAddress = (address: string) =>
  address.trim().toUpperCase();

// BeaconProblem (device_model.go).
export const beaconSchema = z.object({
  address: z
    .string()
    .refine(
      (address) => BEACON_ADDRESS_PATTERN.test(normaliseBeaconAddress(address)),
      "A MAC address, AA:BB:CC:DD:EE:FF",
    ),
  label: z.string().trim().min(1, "Label is required"),
});

export type BeaconFormData = z.infer<typeof beaconSchema>;

// A typed number that normalises to E.164 (utils/phone.ts).
const phoneField = z
  .string()
  .refine(
    (phone) => isE164(normalisePhone(phone)),
    "A phone number: 10 digits for the US, or + and the country code",
  );

// ContactProblem (contact_model.go). An empty event list is allowed: the
// contact is texted nothing but the invite.
export const contactSchema = z.object({
  contactName: z.string().trim().min(1, "Name is required"),
  phone: phoneField,
  notifyEventList: z.array(z.string()),
});

export type ContactFormData = z.infer<typeof contactSchema>;

// TenantSettingProblem (tenant_setting_model.go); the zone is picked from
// the browser's list, so only its presence is checked here.
export const tenantSettingSchema = z.object({
  timeZone: z.string().min(1, "Time zone is required"),
  remindIntervalMin: z
    .number({ error: "Required" })
    .int("Whole minutes only")
    .min(1, "From 1 to 1440 minutes")
    .max(1440, "From 1 to 1440 minutes"),
  dailyReminderTime: z
    .string()
    .regex(/^([01][0-9]|2[0-3]):[0-5][0-9]$/, "A time, HH:MM"),
});

export type TenantSettingFormData = z.infer<typeof tenantSettingSchema>;

// POST /smsSimulateInbound {fromPhone, body}.
export const simulateTextSchema = z.object({
  fromPhone: phoneField,
  body: z.string().trim().min(1, "Type the text"),
});

export type SimulateTextFormData = z.infer<typeof simulateTextSchema>;
