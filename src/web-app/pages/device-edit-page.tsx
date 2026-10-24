import { useState, useEffect, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useForm } from "@mantine/form";
import { zod4Resolver } from "mantine-form-zod-resolver";
import {
  Alert,
  Badge,
  Button,
  Container,
  Group,
  Loader,
  NumberInput,
  Table,
  Text,
  TextInput,
  Title,
} from "@mantine/core";
import { modals } from "@mantine/modals";
import { ROUTES } from "../constants/routes";
import { grants } from "../constants/grants";
import { useDeviceStore } from "../stores/device-store";
import { useBeaconStore } from "../stores/beacon-store";
import { useContactStore } from "../stores/contact-store";
import { useAuthStore } from "../stores/auth-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { confirmDelete } from "../utils/confirm-delete";
import { notifyFailure, notifySuccess } from "../utils/notify";
import { formatAgo, formatTs } from "../utils/relative-time";
import { reportError } from "../utils/report-error";
import { deviceSchema, type DeviceFormData } from "../schemas/form-schemas";
import type { Device, DeviceTokenIssued } from "../types/device-type";
import { DeviceTokenModal } from "../components/device-token-modal";
import { ContactStatusBadge } from "../components/contact-status-badge";
import { LabelWithHelp } from "../components/help-tip";

// What a new device starts with: a check-in a minute, the alarm after three
// missed ones, a band absent after five minutes unseen.
const DEFAULT_DEVICE_FORM: DeviceFormData = {
  deviceName: "",
  checkinIntervalSec: 60,
  missedCheckinBeforeAlarm: 3,
  absentAfterSec: 300,
};

function DeviceEditPage() {
  const [searchParams] = useSearchParams();
  const id = searchParams.get("deviceId") || "";
  const isNew = id === "new";
  const requiredGrant = isNew ? grants.CREATE_DEVICE : grants.UPDATE_DEVICE;
  const authState = useAuthGuard([requiredGrant]);
  const grantList = useAuthStore((state) => state.grantList);
  const { readDeviceById, createItem, updateItem, issueDeviceToken } =
    useDeviceStore();
  const beaconStore = useBeaconStore();
  const contactStore = useContactStore();
  const navigate = useNavigate();

  const numericId = isNaN(Number(id)) ? null : Number(id);
  const canReadBeacon = grantList.includes(grants.READ_BEACON);
  const canReadContact = grantList.includes(grants.READ_CONTACT);

  const form = useForm<DeviceFormData>({
    initialValues: DEFAULT_DEVICE_FORM,
    validate: zod4Resolver(deviceSchema),
  });
  const [issuedToken, setIssuedToken] = useState<DeviceTokenIssued | null>(
    null,
  );
  // The route id the form was last loaded for, and what that load found
  // (example-edit-page.tsx): the row, for its read-only fields and the
  // updateTs a save locks on, or null.
  const requestedId = useRef<string | null>(null);
  const [loadResult, setLoadResult] = useState<{
    id: string;
    device: Device | null;
  } | null>(null);

  useEffect(() => {
    if (authState !== "YES" || requestedId.current === id) return;
    requestedId.current = id;
    if (isNew || !numericId) {
      form.reset();
      return;
    }
    void readDeviceById(numericId)
      .then((existingDevice) => {
        if (requestedId.current !== id) return; // a later id took over
        if (existingDevice) {
          form.setValues({
            deviceName: existingDevice.deviceName,
            checkinIntervalSec: existingDevice.checkinIntervalSec,
            missedCheckinBeforeAlarm: existingDevice.missedCheckinBeforeAlarm,
            absentAfterSec: existingDevice.absentAfterSec,
          });
        }
        setLoadResult({ id, device: existingDevice });
      })
      .catch((error: unknown) => {
        if (requestedId.current !== id) return;
        notifyFailure(error, "loading device", "Failed to load device");
        setLoadResult({ id, device: null });
      });
    // The device's bands and contacts, each behind its own read grant.
    if (canReadBeacon) {
      void beaconStore
        .readList({ queryParams: { deviceId: numericId } })
        .catch((err) => reportError(err, "band list"));
    }
    if (canReadContact) {
      void contactStore
        .readList({ queryParams: { deviceId: numericId } })
        .catch((err) => reportError(err, "contact list"));
    }
    // `form` is a new object every render and the stores' actions are
    // stable; requestedId makes this run once per route id.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [authState, id, isNew, numericId, readDeviceById]);

  //============================^ HOOK LINE ^====================
  const device =
    !isNew && loadResult?.id === id ? loadResult.device : null;
  if (authState === "CHECKING") {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }
  if (authState !== "YES") {
    return <Container size={1280} px={32} py={32} mx="auto"><Alert color="red">Authentication required.</Alert></Container>;
  }

  if (!isNew && numericId && loadResult?.id !== id) {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }
  // Never render the form for a record that was not loaded.
  if (!isNew && (!numericId || !device)) {
    return (
      <Container size={1280} px={32} py={32} mx="auto">
        <Title order={1} mb="lg">Edit Device</Title>
        <Alert color="red">This device could not be loaded.</Alert>
      </Container>
    );
  }

  const handleSubmit = async (values: DeviceFormData) => {
    const data = {
      deviceName: values.deviceName.trim(),
      checkinIntervalSec: values.checkinIntervalSec,
      missedCheckinBeforeAlarm: values.missedCheckinBeforeAlarm,
      absentAfterSec: values.absentAfterSec,
    };
    try {
      if (isNew || !device) {
        const created = await createItem(data);
        notifySuccess("Device created successfully");
        // Stay on the device: its bands, contacts and token come next.
        void navigate(`${ROUTES.DEVICE_EDIT}?deviceId=${created.deviceId}`);
      } else {
        await updateItem({
          deviceId: device.deviceId,
          ...data,
          updateTs: device.updateTs,
        });
        notifySuccess("Device updated successfully");
        void navigate(ROUTES.DEVICE_LIST);
      }
    } catch (error) {
      notifyFailure(error, "saving device", "Failed to save device");
    }
  };

  const issueToken = async (current: Device) => {
    try {
      const issued = await issueDeviceToken(
        current.deviceId,
        current.updateTs,
      );
      setLoadResult({
        id,
        device: {
          ...current,
          updateTs: issued.updateTs,
          tokenIssuedTs: issued.tokenIssuedTs,
        },
      });
      setIssuedToken(issued);
      notifySuccess("Device token issued");
    } catch (error) {
      notifyFailure(error, "issuing device token", "Failed to issue a token");
    }
  };

  // A device that already has a token is asked about first: the new one
  // replaces it, and the base station using it stops checking in.
  const handleIssueTokenClick = (current: Device) => {
    if (current.tokenIssuedTs === undefined) {
      void issueToken(current);
      return;
    }
    modals.openConfirmModal({
      title: "Issue a new token?",
      centered: true,
      children: (
        <Text>
          The token issued {formatTs(current.tokenIssuedTs)} stops working at
          once. The base station needs the new one in its config file.
        </Text>
      ),
      labels: { confirm: "Issue new token", cancel: "Cancel" },
      confirmProps: { color: "red" },
      onConfirm: () => void issueToken(current),
    });
  };

  const handleDeleteBeacon = (beaconId: number, updateTs: number) => {
    confirmDelete("band", () => {
      beaconStore
        .deleteItem(beaconId, updateTs)
        .then(() => notifySuccess("Band deleted successfully"))
        .catch((error: unknown) =>
          notifyFailure(error, "deleting band", "Failed to delete band"),
        );
    });
  };

  const handleDeleteContact = (contactId: number, updateTs: number) => {
    confirmDelete("contact", () => {
      contactStore
        .deleteItem(contactId, updateTs)
        .then(() => notifySuccess("Contact deleted successfully"))
        .catch((error: unknown) =>
          notifyFailure(error, "deleting contact", "Failed to delete contact"),
        );
    });
  };

  const deviceBeaconList = device
    ? beaconStore.itemList.filter((b) => b.deviceId === device.deviceId)
    : [];
  const deviceContactList = device
    ? contactStore.itemList.filter((c) => c.deviceId === device.deviceId)
    : [];

  return (
    <Container size={1280} px={32} py={32} mx="auto">
      <Group justify="space-between" mb="lg">
        <Title order={1}>{isNew ? "Add Device" : "Edit Device"}</Title>
        {device && grantList.includes(grants.ISSUE_DEVICE_TOKEN) && (
          <Button
            variant="light"
            onClick={() => handleIssueTokenClick(device)}
          >
            Issue device token
          </Button>
        )}
      </Group>
      <form onSubmit={form.onSubmit((values) => void handleSubmit(values))}>
        <TextInput
          label="Name"
          {...form.getInputProps("deviceName")}
          mb="md"
        />
        <NumberInput
          label={
            <LabelWithHelp
              label="Check-in interval (seconds)"
              help="How often the base station reports; at least 10"
            />
          }
          allowDecimal={false}
          {...form.getInputProps("checkinIntervalSec")}
          mb="md"
        />
        <NumberInput
          label={
            <LabelWithHelp
              label="Missed check-ins before alarm"
              help="How many check-ins can be missed before the base station counts as offline; at least 1"
            />
          }
          allowDecimal={false}
          {...form.getInputProps("missedCheckinBeforeAlarm")}
          mb="md"
        />
        <NumberInput
          label={
            <LabelWithHelp
              label="Absent after (seconds)"
              help="How long a band can go unseen before it counts as absent; at least 10"
            />
          }
          allowDecimal={false}
          {...form.getInputProps("absentAfterSec")}
          mb="md"
        />
        {device && (
          <Group mb="md" grow>
            <TextInput
              label="Config version"
              value={String(device.configVersion)}
              readOnly
            />
            <TextInput
              label="Last check-in"
              value={
                device.lastCheckinTs === undefined
                  ? "never"
                  : `${formatTs(device.lastCheckinTs)} (${formatAgo(device.lastCheckinTs)})`
              }
              readOnly
            />
            <TextInput
              label="Token issued"
              value={formatTs(device.tokenIssuedTs)}
              readOnly
            />
          </Group>
        )}
        <Group>
          <Button type="submit">Save</Button>
          <Button
            variant="outline"
            onClick={() => void navigate(ROUTES.DEVICE_LIST)}
          >
            Cancel
          </Button>
        </Group>
      </form>

      {device && canReadBeacon && (
        <>
          <Group justify="space-between" mt="xl" mb="sm">
            <Title order={2} size="h3">Bands</Title>
            {grantList.includes(grants.CREATE_BEACON) && (
              <Button
                size="xs"
                onClick={() =>
                  void navigate(
                    `${ROUTES.BEACON_EDIT}?deviceId=${device.deviceId}`,
                  )
                }
              >
                Add Band
              </Button>
            )}
          </Group>
          {deviceBeaconList.length === 0 ? (
            <Text c="dimmed">No bands yet.</Text>
          ) : (
            <Table>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Label</Table.Th>
                  <Table.Th>Address</Table.Th>
                  <Table.Th>Present</Table.Th>
                  <Table.Th>Last seen</Table.Th>
                  <Table.Th>Actions</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {deviceBeaconList.map((beacon) => (
                  <Table.Tr key={beacon.beaconId}>
                    <Table.Td>{beacon.label}</Table.Td>
                    <Table.Td ff="monospace">{beacon.address}</Table.Td>
                    <Table.Td>
                      {beacon.present === undefined ? (
                        <Badge color="gray" variant="light">unknown</Badge>
                      ) : beacon.present ? (
                        <Badge color="green" variant="light">present</Badge>
                      ) : (
                        <Badge color="red" variant="light">absent</Badge>
                      )}
                    </Table.Td>
                    <Table.Td>{formatAgo(beacon.lastSeenTs)}</Table.Td>
                    <Table.Td>
                      {grantList.includes(grants.UPDATE_BEACON) && (
                        <Button
                          size="xs"
                          onClick={() =>
                            void navigate(
                              `${ROUTES.BEACON_EDIT}?beaconId=${beacon.beaconId}`,
                            )
                          }
                        >
                          Edit
                        </Button>
                      )}
                      {grantList.includes(grants.DELETE_BEACON) && (
                        <Button
                          size="xs"
                          color="red"
                          ml="xs"
                          onClick={() =>
                            handleDeleteBeacon(beacon.beaconId, beacon.updateTs)
                          }
                        >
                          Delete
                        </Button>
                      )}
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          )}
        </>
      )}

      {device && canReadContact && (
        <>
          <Group justify="space-between" mt="xl" mb="sm">
            <Title order={2} size="h3">Contacts</Title>
            {grantList.includes(grants.CREATE_CONTACT) && (
              <Button
                size="xs"
                onClick={() =>
                  void navigate(
                    `${ROUTES.CONTACT_EDIT}?deviceId=${device.deviceId}`,
                  )
                }
              >
                Add Contact
              </Button>
            )}
          </Group>
          {deviceContactList.length === 0 ? (
            <Text c="dimmed">No contacts yet.</Text>
          ) : (
            <Table>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Name</Table.Th>
                  <Table.Th>Phone</Table.Th>
                  <Table.Th>Status</Table.Th>
                  <Table.Th>Actions</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {deviceContactList.map((contact) => (
                  <Table.Tr key={contact.contactId}>
                    <Table.Td>{contact.contactName}</Table.Td>
                    <Table.Td ff="monospace">{contact.phoneE164}</Table.Td>
                    <Table.Td>
                      <ContactStatusBadge status={contact.contactStatus} />
                    </Table.Td>
                    <Table.Td>
                      {grantList.includes(grants.UPDATE_CONTACT) && (
                        <Button
                          size="xs"
                          onClick={() =>
                            void navigate(
                              `${ROUTES.CONTACT_EDIT}?contactId=${contact.contactId}`,
                            )
                          }
                        >
                          Edit
                        </Button>
                      )}
                      {grantList.includes(grants.DELETE_CONTACT) && (
                        <Button
                          size="xs"
                          color="red"
                          ml="xs"
                          onClick={() =>
                            handleDeleteContact(
                              contact.contactId,
                              contact.updateTs,
                            )
                          }
                        >
                          Delete
                        </Button>
                      )}
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          )}
        </>
      )}

      <DeviceTokenModal
        issued={issuedToken}
        deviceName={device?.deviceName ?? ""}
        onClose={() => setIssuedToken(null)}
      />
    </Container>
  );
}

export default DeviceEditPage;
