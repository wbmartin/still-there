import { useState, useEffect, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useForm } from "@mantine/form";
import { zod4Resolver } from "mantine-form-zod-resolver";
import {
  Alert,
  Button,
  Container,
  Group,
  Loader,
  TextInput,
  Title,
} from "@mantine/core";
import { ROUTES } from "../constants/routes";
import { grants } from "../constants/grants";
import { useBeaconStore } from "../stores/beacon-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { notifyFailure, notifySuccess } from "../utils/notify";
import { formatAgo, formatTs } from "../utils/relative-time";
import {
  beaconSchema,
  normaliseBeaconAddress,
  type BeaconFormData,
} from "../schemas/form-schemas";
import type { Beacon } from "../types/device-type";
import { LabelWithHelp } from "../components/help-tip";

const DEFAULT_BEACON_FORM: BeaconFormData = { address: "", label: "" };

// A band: `?beaconId=N` edits one, `?deviceId=N` adds one to that device.
// Either way the page returns to the device.
function BeaconEditPage() {
  const [searchParams] = useSearchParams();
  const id = searchParams.get("beaconId") || "";
  const isNew = id === "";
  const newDeviceId = Number(searchParams.get("deviceId")) || null;
  const requiredGrant = isNew ? grants.CREATE_BEACON : grants.UPDATE_BEACON;
  const authState = useAuthGuard([requiredGrant]);
  const { readBeaconById, createItem, updateItem } = useBeaconStore();
  const navigate = useNavigate();

  const numericId = isNaN(Number(id)) ? null : Number(id);

  const form = useForm<BeaconFormData>({
    initialValues: DEFAULT_BEACON_FORM,
    validate: zod4Resolver(beaconSchema),
  });
  // The route id the form was last loaded for, and what that load found
  // (example-edit-page.tsx): the row, or null.
  const requestedId = useRef<string | null>(null);
  const [loadResult, setLoadResult] = useState<{
    id: string;
    beacon: Beacon | null;
  } | null>(null);

  useEffect(() => {
    if (authState !== "YES" || requestedId.current === id) return;
    requestedId.current = id;
    if (isNew || !numericId) {
      form.reset();
      return;
    }
    void readBeaconById(numericId)
      .then((existingBeacon) => {
        if (requestedId.current !== id) return; // a later id took over
        if (existingBeacon) {
          form.setValues({
            address: existingBeacon.address,
            label: existingBeacon.label,
          });
        }
        setLoadResult({ id, beacon: existingBeacon });
      })
      .catch((error: unknown) => {
        if (requestedId.current !== id) return;
        notifyFailure(error, "loading band", "Failed to load band");
        setLoadResult({ id, beacon: null });
      });
    // `form` is a new object every render; requestedId makes this run once
    // per route id.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [authState, id, isNew, numericId, readBeaconById]);

  //============================^ HOOK LINE ^====================
  const beacon =
    !isNew && loadResult?.id === id ? loadResult.beacon : null;
  if (authState === "CHECKING") {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }
  if (authState !== "YES") {
    return <Container size={1280} px={32} py={32} mx="auto"><Alert color="red">Authentication required.</Alert></Container>;
  }

  if (!isNew && numericId && loadResult?.id !== id) {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }
  // Never render the form for a record that was not loaded, nor a new band
  // with no device to add it to.
  const deviceId = beacon?.deviceId ?? newDeviceId;
  if (
    (!isNew && (!numericId || !beacon)) ||
    deviceId === null
  ) {
    return (
      <Container size={1280} px={32} py={32} mx="auto">
        <Title order={1} mb="lg">{isNew ? "Add Band" : "Edit Band"}</Title>
        <Alert color="red">This band could not be loaded.</Alert>
      </Container>
    );
  }
  const devicePath = `${ROUTES.DEVICE_EDIT}?deviceId=${deviceId}`;

  const handleSubmit = async (values: BeaconFormData) => {
    const data = {
      address: normaliseBeaconAddress(values.address),
      label: values.label.trim(),
    };
    try {
      if (beacon) {
        await updateItem({
          beaconId: beacon.beaconId,
          ...data,
          updateTs: beacon.updateTs,
        });
        notifySuccess("Band updated successfully");
      } else {
        await createItem({ deviceId, ...data });
        notifySuccess("Band created successfully");
      }
      void navigate(devicePath);
    } catch (error) {
      notifyFailure(error, "saving band", "Failed to save band");
    }
  };

  const presentText =
    beacon?.present === undefined
      ? "not reported yet"
      : beacon.present
        ? "present"
        : "absent";

  return (
    <Container size={1280} px={32} py={32} mx="auto">
      <Title order={1} mb="lg">
        {isNew ? "Add Band" : "Edit Band"}
      </Title>
      <form onSubmit={form.onSubmit((values) => void handleSubmit(values))}>
        <TextInput
          label={
            <LabelWithHelp
              label="Address"
              help="The band's Bluetooth MAC address, AA:BB:CC:DD:EE:FF"
            />
          }
          ff="monospace"
          {...form.getInputProps("address")}
          onBlur={() =>
            form.setFieldValue(
              "address",
              normaliseBeaconAddress(form.values.address),
            )
          }
          mb="md"
        />
        <TextInput
          label="Label"
          placeholder="Whose band, e.g. Grandma"
          {...form.getInputProps("label")}
          mb="md"
        />
        {beacon && (
          <Group mb="md" grow>
            <TextInput label="Present" value={presentText} readOnly />
            <TextInput
              label="Last seen"
              value={
                beacon.lastSeenTs === undefined
                  ? "never"
                  : `${formatTs(beacon.lastSeenTs)} (${formatAgo(beacon.lastSeenTs)})`
              }
              readOnly
            />
            <TextInput
              label="RSSI"
              value={
                beacon.lastRssi === undefined ? "" : `${beacon.lastRssi} dBm`
              }
              readOnly
            />
          </Group>
        )}
        <Group>
          <Button type="submit">Save</Button>
          <Button variant="outline" onClick={() => void navigate(devicePath)}>
            Cancel
          </Button>
        </Group>
      </form>
    </Container>
  );
}

export default BeaconEditPage;
