import { useEffect, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useForm } from "@mantine/form";
import { zod4Resolver } from "mantine-form-zod-resolver";
import {
  Alert as MantineAlert,
  Badge,
  Button,
  Container,
  Group,
  Loader,
  Paper,
  Table,
  Text,
  TextInput,
  Title,
} from "@mantine/core";
import { modals } from "@mantine/modals";
import { useAlertStore } from "../stores/alert-store";
import { useDeviceStore } from "../stores/device-store";
import { useBeaconStore } from "../stores/beacon-store";
import { useAuthStore } from "../stores/auth-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { notifyFailure, notifySuccess } from "../utils/notify";
import { normalisePhone } from "../utils/phone";
import { formatTs } from "../utils/relative-time";
import { reportError } from "../utils/report-error";
import { isApiStatus } from "../utils/api";
import {
  simulateTextSchema,
  type SimulateTextFormData,
} from "../schemas/form-schemas";
import {
  ALERT_KIND_LABEL,
  ALERT_STATE_LABEL,
  type Alert,
  type AlertState,
} from "../types/alert-type";
import { ROUTES } from "../constants/routes";
import { grants } from "../constants/grants";
import FieldHelp from "../help/field-help";

// The list re-reads itself this often while the page is open, so an alarm
// raised or acknowledged by text shows up without a reload.
const REFRESH_INTERVAL_MS = 30_000;

const STATE_COLOR: Record<AlertState, string> = {
  raised: "red",
  acknowledged: "orange",
  turned_off: "gray",
  resolved: "green",
};

// Alarms: the open ones first, then the last week's resolved ones (the
// API's order). `?deviceId=N` shows one device's.
function AlertListPage() {
  const authState = useAuthGuard([grants.READ_ALERT]);
  const grantList = useAuthStore((state) => state.grantList);
  const {
    alertList,
    loaded,
    error,
    readAlertList,
    acknowledgeAlert,
    turnOffAlert,
    simulateInboundText,
    probeSimulateInbound,
  } = useAlertStore();
  const deviceList = useDeviceStore((state) => state.itemList);
  const readDeviceList = useDeviceStore((state) => state.readList);
  const beaconList = useBeaconStore((state) => state.itemList);
  const readBeaconList = useBeaconStore((state) => state.readList);
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const deviceId = Number(searchParams.get("deviceId")) || undefined;
  const canUpdate = grantList.includes(grants.UPDATE_ALERT);
  const canReadDevice = grantList.includes(grants.READ_DEVICE);
  const canReadBeacon = grantList.includes(grants.READ_BEACON);
  const hasFetched = useRef(false);
  // Whether POST /smsSimulateInbound is on (SMS_PROVIDER=log); null until
  // the probe answers, and the panel shows only on true.
  const [simulateOn, setSimulateOn] = useState<boolean | null>(null);
  const [simulateReply, setSimulateReply] = useState<string | null>(null);
  const simulateForm = useForm<SimulateTextFormData>({
    initialValues: { fromPhone: "", body: "OK" },
    validate: zod4Resolver(simulateTextSchema),
  });

  // The names the rows show, each behind its own read grant; without one
  // the row shows the id. Loaded once.
  useEffect(() => {
    if (hasFetched.current || authState !== "YES") return;
    hasFetched.current = true;
    if (canReadDevice) {
      void readDeviceList({ incremental: true }).catch((err) =>
        reportError(err, "device list"),
      );
    }
    if (canReadBeacon) {
      void readBeaconList().catch((err) => reportError(err, "band list"));
    }
    if (canUpdate) {
      void probeSimulateInbound()
        .then(setSimulateOn)
        .catch((err) => reportError(err, "simulate probe"));
    }
  }, [
    authState,
    canReadDevice,
    canReadBeacon,
    canUpdate,
    readDeviceList,
    readBeaconList,
    probeSimulateInbound,
  ]);

  // The alarms: now, and every REFRESH_INTERVAL_MS while the page is open.
  useEffect(() => {
    if (authState !== "YES") return;
    const refresh = () =>
      void readAlertList(deviceId).catch((err) =>
        reportError(err, "alert list"),
      );
    refresh();
    const timer = window.setInterval(refresh, REFRESH_INTERVAL_MS);
    return () => window.clearInterval(timer);
  }, [authState, deviceId, readAlertList]);

  //============================^ HOOK LINE ^====================

  if (authState === "CHECKING") {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }
  if (authState !== "YES") {
    return <Container size={1280} px={32} py={32} mx="auto"><MantineAlert color="red">Authentication required.</MantineAlert></Container>;
  }

  const deviceName = (id: number) =>
    deviceList.find((device) => device.deviceId === id)?.deviceName ??
    `Device ${id}`;
  const bandLabel = (alert: Alert) =>
    alert.beaconId === undefined
      ? "base station"
      : (beaconList.find((beacon) => beacon.beaconId === alert.beaconId)
          ?.label ?? `Band ${alert.beaconId}`);

  const runAction = async (
    action: (alert: Alert) => Promise<Alert>,
    alert: Alert,
    doneMessage: string,
  ) => {
    try {
      await action(alert);
      notifySuccess(doneMessage);
    } catch (error) {
      // 409: someone (or a text) changed it first; the API says how.
      notifyFailure(error, "changing alert", "Failed to change the alarm");
      if (isApiStatus(error, 409)) {
        void readAlertList(deviceId).catch((err) =>
          reportError(err, "alert list"),
        );
      }
    }
  };

  const confirmAction = (
    alert: Alert,
    title: string,
    body: string,
    confirmLabel: string,
    action: (alert: Alert) => Promise<Alert>,
    doneMessage: string,
  ) => {
    modals.openConfirmModal({
      title,
      centered: true,
      children: (
        <Text>
          {body} ({deviceName(alert.deviceId)}, {bandLabel(alert)})
        </Text>
      ),
      labels: { confirm: confirmLabel, cancel: "Cancel" },
      confirmProps: { color: "red" },
      onConfirm: () => void runAction(action, alert, doneMessage),
    });
  };

  const handleSimulate = async (values: SimulateTextFormData) => {
    try {
      const reply = await simulateInboundText(
        normalisePhone(values.fromPhone),
        values.body.trim(),
      );
      setSimulateReply(reply);
      void readAlertList(deviceId).catch((err) =>
        reportError(err, "alert list"),
      );
    } catch (error) {
      if (isApiStatus(error, 404)) {
        setSimulateOn(false);
        return;
      }
      notifyFailure(error, "simulating text", "Failed to simulate the text");
    }
  };

  return (
    <Container size={1280} px={32} py={32} mx="auto">
      <Group justify="space-between" mb="lg">
        <Group gap={4}>
          <Title order={1}>Alerts</Title>
          <FieldHelp id="alarm-life" />
        </Group>
        {deviceId !== undefined && (
          <Button
            variant="outline"
            onClick={() => void navigate(ROUTES.ALERT_LIST)}
          >
            All devices
          </Button>
        )}
      </Group>
      {deviceId !== undefined && (
        <Text mb="md">Showing {deviceName(deviceId)} only.</Text>
      )}
      {!loaded ? (
        <Container ta="center"><Loader /></Container>
      ) : error ? (
        <MantineAlert color="red">Error loading alerts: {error}</MantineAlert>
      ) : alertList.length === 0 ? (
        <MantineAlert color="green">
          No open alarms, and none resolved in the last week.
        </MantineAlert>
      ) : (
        <Table>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Device</Table.Th>
              <Table.Th>Band</Table.Th>
              <Table.Th>Kind</Table.Th>
              <Table.Th>State</Table.Th>
              <Table.Th>Raised</Table.Th>
              <Table.Th>Acknowledged / turned off by</Table.Th>
              {canUpdate && <Table.Th>Actions</Table.Th>}
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {alertList.map((alert) => (
              <Table.Tr key={alert.alertId}>
                <Table.Td>{deviceName(alert.deviceId)}</Table.Td>
                <Table.Td>{bandLabel(alert)}</Table.Td>
                <Table.Td>
                  {ALERT_KIND_LABEL[alert.alertKind] ?? alert.alertKind}
                </Table.Td>
                <Table.Td>
                  <Badge color={STATE_COLOR[alert.alertState] ?? "gray"}>
                    {ALERT_STATE_LABEL[alert.alertState] ?? alert.alertState}
                  </Badge>
                </Table.Td>
                <Table.Td>{formatTs(alert.raisedTs)}</Table.Td>
                <Table.Td>
                  {[
                    alert.ackBy && `acknowledged by ${alert.ackBy}`,
                    alert.offBy && `turned off by ${alert.offBy}`,
                  ]
                    .filter(Boolean)
                    .join("; ")}
                </Table.Td>
                {canUpdate && (
                  <Table.Td>
                    {alert.alertState === "raised" && (
                      <Button
                        size="xs"
                        onClick={() =>
                          confirmAction(
                            alert,
                            "Acknowledge this alarm?",
                            "Everyone on the device's contact list is " +
                              "texted that you acknowledged it; reminders " +
                              "drop to one a day.",
                            "Acknowledge",
                            acknowledgeAlert,
                            "Alarm acknowledged",
                          )
                        }
                      >
                        Acknowledge
                      </Button>
                    )}
                    {(alert.alertState === "raised" ||
                      alert.alertState === "acknowledged") && (
                      <Button
                        size="xs"
                        color="red"
                        ml="xs"
                        onClick={() =>
                          confirmAction(
                            alert,
                            "Turn this alarm off?",
                            "Everyone on the device's contact list is " +
                              "texted that you turned it off; no more " +
                              "reminders are sent for it.",
                            "Turn off",
                            turnOffAlert,
                            "Alarm turned off",
                          )
                        }
                      >
                        Turn off
                      </Button>
                    )}
                  </Table.Td>
                )}
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      )}

      {canUpdate && simulateOn === true && (
        <Paper withBorder p="md" mt="xl" maw={560}>
          <Title order={2} size="h4" mb="xs">
            Simulate a text reply
          </Title>
          <Text size="sm" c="dimmed" mb="sm">
            This server logs texts instead of sending them. Type what a
            contact would text back (OK, Y, STOP, START, HELP) as if from
            their number.
          </Text>
          <form
            onSubmit={simulateForm.onSubmit(
              (values) => void handleSimulate(values),
            )}
          >
            <Group align="flex-start" grow mb="sm">
              <TextInput
                label="From phone"
                {...simulateForm.getInputProps("fromPhone")}
              />
              <TextInput
                label="Text"
                {...simulateForm.getInputProps("body")}
              />
            </Group>
            <Button type="submit">Send</Button>
          </form>
          {simulateReply !== null && (
            <MantineAlert color="blue" variant="light" mt="sm" title="Reply">
              {simulateReply || "(no reply)"}
            </MantineAlert>
          )}
        </Paper>
      )}
    </Container>
  );
}

export default AlertListPage;
