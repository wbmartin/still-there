// The user edit page's MFA-exempt control. The server decides whether the
// API may set mfaExempt (ALLOW_MFA_EXEMPT_VIA_API, B097; the web app reads
// it from /me): when it may not, POST /user and /userInvite refuse true and
// PUT /user refuses turning it on, but keeping or clearing a stored true
// still works. So: allowed, the plain checkbox; refused and the login is
// already exempt, the checkbox stays so an admin sees the exemption and
// can clear it; refused otherwise, nothing.
import { Checkbox } from "@mantine/core";
import { LabelWithHelp } from "./help-tip";

export function MfaExemptField({
  allowSetting,
  storedExempt,
  checked,
  onChange,
}: {
  /** /me's allowMfaExemptViaApi; false until the profile has loaded. */
  allowSetting: boolean;
  /** The loaded login's mfaExempt as stored; false on create. */
  storedExempt: boolean;
  checked: boolean;
  onChange: (checked: boolean) => void;
}) {
  if (!allowSetting && !storedExempt) return null;
  return (
    <Checkbox
      label={
        <LabelWithHelp
          label="MFA Exempt"
          help={
            allowSetting
              ? undefined
              : "This server does not allow turning MFA exemption on. " +
                "Untick and save to clear it; it cannot be turned back on " +
                "here."
          }
        />
      }
      checked={checked}
      onChange={(event) => onChange(event.currentTarget.checked)}
      mb="md"
    />
  );
}
