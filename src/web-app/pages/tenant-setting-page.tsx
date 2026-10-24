import { useEffect, useMemo, useRef, useState } from "react";
import { useForm } from "@mantine/form";
import { zod4Resolver } from "mantine-form-zod-resolver";
import {
  Alert,
  Button,
  Container,
  Group,
  Loader,
  NumberInput,
  Select,
  TextInput,
  Title,
} from "@mantine/core";
import { useTenantSettingStore } from "../stores/tenant-setting-store";
import { useAuthStore } from "../stores/auth-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { notifyFailure, notifySuccess } from "../utils/notify";
import {
  tenantSettingSchema,
  type TenantSettingFormData,
} from "../schemas/form-schemas";
import { grants } from "../constants/grants";
import { LabelWithHelp } from "../components/help-tip";

// What the form shows before the tenant's first save (GET answers 404).
const DEFAULT_TENANT_SETTING_FORM: TenantSettingFormData = {
  timeZone: "America/New_York",
  remindIntervalMin: 15,
  dailyReminderTime: "09:00",
};

// The tenant's reminder settings: one row, created by the first save.
function TenantSettingPage() {
  const authState = useAuthGuard([grants.READ_TENANT_SETTING]);
  const canUpdate = useAuthStore((state) =>
    state.grantList.includes(grants.UPDATE_TENANT_SETTING),
  );
  const { tenantSetting, readTenantSetting, saveTenantSetting } =
    useTenantSettingStore();
  const form = useForm<TenantSettingFormData>({
    initialValues: DEFAULT_TENANT_SETTING_FORM,
    validate: zod4Resolver(tenantSettingSchema),
  });
  const hasFetched = useRef(false);
  // null while loading; "failed" shows an alert instead of a form whose
  // save would overwrite settings nobody read.
  const [loadState, setLoadState] = useState<"loaded" | "failed" | null>(
    null,
  );
  // The browser's IANA zones, plus the saved one should the browser not
  // list it.
  const timeZoneList = useMemo(() => {
    const zoneList = Intl.supportedValuesOf("timeZone");
    const saved = form.values.timeZone;
    return saved && !zoneList.includes(saved) ? [saved, ...zoneList] : zoneList;
  }, [form.values.timeZone]);

  useEffect(() => {
    if (hasFetched.current || authState !== "YES") return;
    hasFetched.current = true;
    void readTenantSetting()
      .then((setting) => {
        if (setting) {
          form.setValues({
            timeZone: setting.timeZone,
            remindIntervalMin: setting.remindIntervalMin,
            dailyReminderTime: setting.dailyReminderTime,
          });
        }
        setLoadState("loaded");
      })
      .catch((error: unknown) => {
        notifyFailure(error, "loading settings", "Failed to load settings");
        setLoadState("failed");
      });
    // `form` is a new object every render; hasFetched makes this run once.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [authState, readTenantSetting]);

  //============================^ HOOK LINE ^====================
  if (authState === "CHECKING" || (authState === "YES" && !loadState)) {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }
  if (authState !== "YES") {
    return <Container size={1280} px={32} py={32} mx="auto"><Alert color="red">Authentication required.</Alert></Container>;
  }
  if (loadState === "failed") {
    return (
      <Container size={1280} px={32} py={32} mx="auto">
        <Title order={1} mb="lg">Settings</Title>
        <Alert color="red">The settings could not be loaded.</Alert>
      </Container>
    );
  }

  const handleSubmit = async (values: TenantSettingFormData) => {
    try {
      await saveTenantSetting({
        ...values,
        // 0 before the first save, which creates the row.
        updateTs: tenantSetting?.updateTs ?? 0,
      });
      notifySuccess("Settings saved successfully");
    } catch (error) {
      notifyFailure(error, "saving settings", "Failed to save settings");
    }
  };

  return (
    <Container size={1280} px={32} py={32} mx="auto">
      <Title order={1} mb="lg">Settings</Title>
      {!tenantSetting && (
        <Alert color="blue" variant="light" mb="md" maw={480}>
          Not saved yet: these are the defaults. Reminders follow them once
          you save.
        </Alert>
      )}
      <form
        onSubmit={form.onSubmit((values) => void handleSubmit(values))}
        style={{ maxWidth: 480 }}
      >
        <Select
          label="Time zone"
          data={timeZoneList}
          searchable
          allowDeselect={false}
          disabled={!canUpdate}
          {...form.getInputProps("timeZone")}
          mb="md"
        />
        <NumberInput
          label={
            <LabelWithHelp
              label="Reminder interval (minutes)"
              help="How often an unacknowledged alarm is texted again; 1 to 1440"
            />
          }
          allowDecimal={false}
          disabled={!canUpdate}
          {...form.getInputProps("remindIntervalMin")}
          mb="md"
        />
        <TextInput
          type="time"
          label={
            <LabelWithHelp
              label="Daily reminder time"
              help="When the once-a-day reminder for an acknowledged alarm goes out, in the time zone above"
            />
          }
          disabled={!canUpdate}
          {...form.getInputProps("dailyReminderTime")}
          mb="md"
        />
        {canUpdate && (
          <Group>
            <Button type="submit">Save</Button>
          </Group>
        )}
      </form>
    </Container>
  );
}

export default TenantSettingPage;
