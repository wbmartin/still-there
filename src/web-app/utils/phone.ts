// Phone numbers as people type them, to the E.164 form the API takes
// (contactPhonePattern in src/api-srv/internal/model/contact_model.go).

export const E164_PATTERN = /^\+[1-9][0-9]{7,14}$/;

/**
 * The E.164 form of a typed number. Spaces, dots, dashes and brackets are
 * dropped; a US number of 10 digits (or 11 starting with 1) gains +1. A
 * number already starting with + keeps its country code. Anything else
 * comes back stripped but otherwise as typed, and fails E164_PATTERN.
 */
export const normalisePhone = (typed: string): string => {
  const trimmed = typed.trim();
  const digits = trimmed.replace(/\D/g, "");
  if (trimmed.startsWith("+")) return `+${digits}`;
  if (digits.length === 10) return `+1${digits}`;
  if (digits.length === 11 && digits.startsWith("1")) return `+${digits}`;
  return digits;
};

export const isE164 = (phone: string) => E164_PATTERN.test(phone);
