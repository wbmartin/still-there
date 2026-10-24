import { describe, expect, it } from "vitest";
import { zod4Resolver } from "mantine-form-zod-resolver";
import {
  beaconSchema,
  contactSchema,
  deviceSchema,
  normaliseBeaconAddress,
  simulateTextSchema,
  tenantSettingSchema,
} from "../schemas/form-schemas";
import { isE164, normalisePhone } from "../utils/phone";
import { formatAgo } from "../utils/relative-time";

// The monitoring forms refuse what the API's *Problem checks refuse
// (src/api-srv/internal/model/), and phone numbers typed the US way reach
// the API in E.164.

const firstMessage = (result: { success: boolean; error?: unknown }) =>
  result.success
    ? null
    : (result.error as { issues: { message: string }[] }).issues[0].message;

describe("normalisePhone", () => {
  it.each([
    ["704.860.9088", "+17048609088"],
    ["(704) 860-9088", "+17048609088"],
    ["1 704 860 9088", "+17048609088"],
    ["+1 704 860 9088", "+17048609088"],
    ["+44 7700 900123", "+447700900123"],
    [" 7048609088 ", "+17048609088"],
  ])("%s is %s", (typed, expected) => {
    expect(normalisePhone(typed)).toBe(expected);
    expect(isE164(normalisePhone(typed))).toBe(true);
  });

  it.each(["860-9088", "", "abc", "+0 123 456 789", "2 704 860 9088"])(
    "%s does not become E.164",
    (typed) => {
      expect(isE164(normalisePhone(typed))).toBe(false);
    },
  );
});

describe("deviceSchema", () => {
  const valid = {
    deviceName: "Kitchen",
    checkinIntervalSec: 10,
    missedCheckinBeforeAlarm: 1,
    absentAfterSec: 10,
  };

  it("takes the lowest values the API takes", () => {
    expect(deviceSchema.safeParse(valid).success).toBe(true);
  });

  // The pages validate through zod4Resolver: zodResolver expects zod 3's
  // error shape and throws on a zod 4 failure instead of answering it.
  it("reaches the form as field errors through zod4Resolver", () => {
    expect(
      zod4Resolver(deviceSchema)({ ...valid, checkinIntervalSec: 9 }),
    ).toEqual({ checkinIntervalSec: "At least 10 seconds" });
  });

  it.each([
    [{ checkinIntervalSec: 9 }, "At least 10 seconds"],
    [{ missedCheckinBeforeAlarm: 0 }, "At least 1"],
    [{ absentAfterSec: 9 }, "At least 10 seconds"],
    [{ absentAfterSec: "" }, "Required"],
    [{ checkinIntervalSec: 10.5 }, "Whole numbers only"],
    [{ deviceName: "  " }, "Name is required"],
  ])("refuses %o", (change, message) => {
    expect(firstMessage(deviceSchema.safeParse({ ...valid, ...change })))
      .toBe(message);
  });
});

describe("beaconSchema", () => {
  it("takes a lower-case MAC and normalises it to upper case", () => {
    const typed = " aa:bb:cc:dd:ee:0f ";
    expect(beaconSchema.safeParse({ address: typed, label: "G" }).success)
      .toBe(true);
    expect(normaliseBeaconAddress(typed)).toBe("AA:BB:CC:DD:EE:0F");
  });

  it.each(["AA:BB:CC:DD:EE", "AA-BB-CC-DD-EE-FF", "GG:BB:CC:DD:EE:FF", ""])(
    "refuses %s",
    (address) => {
      expect(beaconSchema.safeParse({ address, label: "G" }).success)
        .toBe(false);
    },
  );

  it("needs a label", () => {
    expect(
      firstMessage(
        beaconSchema.safeParse({ address: "AA:BB:CC:DD:EE:FF", label: "" }),
      ),
    ).toBe("Label is required");
  });
});

describe("contactSchema", () => {
  it("takes a US number typed with dots and an empty event list", () => {
    expect(
      contactSchema.safeParse({
        contactName: "Daughter",
        phone: "704.860.9088",
        notifyEventList: [],
      }).success,
    ).toBe(true);
  });

  it("refuses a short number", () => {
    expect(
      contactSchema.safeParse({
        contactName: "Daughter",
        phone: "860.9088",
        notifyEventList: ["raised"],
      }).success,
    ).toBe(false);
  });
});

describe("tenantSettingSchema", () => {
  const valid = {
    timeZone: "America/New_York",
    remindIntervalMin: 15,
    dailyReminderTime: "09:00",
  };

  it("takes the defaults and the bounds", () => {
    expect(tenantSettingSchema.safeParse(valid).success).toBe(true);
    for (const remindIntervalMin of [1, 1440]) {
      expect(
        tenantSettingSchema.safeParse({ ...valid, remindIntervalMin }).success,
      ).toBe(true);
    }
  });

  it.each([
    { remindIntervalMin: 0 },
    { remindIntervalMin: 1441 },
    { dailyReminderTime: "24:00" },
    { dailyReminderTime: "9:00" },
    { timeZone: "" },
  ])("refuses %o", (change) => {
    expect(tenantSettingSchema.safeParse({ ...valid, ...change }).success)
      .toBe(false);
  });
});

describe("simulateTextSchema", () => {
  it("needs a phone and a text", () => {
    expect(
      simulateTextSchema.safeParse({ fromPhone: "7048609088", body: "OK" })
        .success,
    ).toBe(true);
    expect(
      simulateTextSchema.safeParse({ fromPhone: "7048609088", body: " " })
        .success,
    ).toBe(false);
  });
});

describe("formatAgo", () => {
  const nowMs = 1_000_000 * 1000;

  it.each([
    [undefined, "never"],
    [1_000_000 - 30, "just now"],
    [1_000_000 - 5 * 60, "5 min ago"],
    [1_000_000 - 3 * 3600, "3 h ago"],
    [1_000_000 - 2 * 86400, "2 d ago"],
    // A clock a little ahead of the browser's is not "in the future".
    [1_000_000 + 10, "just now"],
  ])("%s is %s", (ts, expected) => {
    expect(formatAgo(ts, nowMs)).toBe(expected);
  });
});
