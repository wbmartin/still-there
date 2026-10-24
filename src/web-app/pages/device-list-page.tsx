import { useEffect, useMemo, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import {
  Alert,
  Badge,
  Button,
  Container,
  Group,
  Loader,
  Table,
  Title,
} from "@mantine/core";
import { useDeviceStore } from "../stores/device-store";
import { useBeaconStore } from "../stores/beacon-store";
import { useAlertStore } from "../stores/alert-store";
import { useAuthStore } from "../stores/auth-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { confirmDelete } from "../utils/confirm-delete";
import { notifyFailure, notifySuccess } from "../utils/notify";
import { formatAgo, formatTs } from "../utils/relative-time";
import { reportError } from "../utils/report-error";
import { isOpenAlert } from "../types/alert-type";
import { ROUTES } from "../constants/routes";
import { grants } from "../constants/grants";
import { SortTh } from "../components/sort-th";
import FieldHelp from "../help/field-help";

// The tenant's base stations. A household has one or two, so the list is
// not paged; sort lives in the URL as on every list page.
function DeviceListPage() {
  const authState = useAuthGuard([grants.READ_DEVICE]);
  const grantList = useAuthStore((state) => state.grantList);
  const {
    itemList: deviceList,
    loading,
    error,
    readList,
    deleteItem,
  } = useDeviceStore();
  const beaconList = useBeaconStore((state) => state.itemList);
  const readBeaconList = useBeaconStore((state) => state.readList);
  const alertList = useAlertStore((state) => state.alertList);
  const readAlertList = useAlertStore((state) => state.readAlertList);
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const setListParams = (update: Record<string, string>) =>
    setSearchParams((previous) => {
      const next = new URLSearchParams(previous);
      for (const [key, value] of Object.entries(update)) {
        if (value) next.set(key, value);
        else next.delete(key);
      }
      return next;
    });
  const hasFetched = useRef(false);
  const sortColumn = searchParams.get("sort") || "deviceName";
  const sortDirection = searchParams.get("dir") === "desc" ? "desc" : "asc";
  const canReadBeacon = grantList.includes(grants.READ_BEACON);
  const canReadAlert = grantList.includes(grants.READ_ALERT);

  useEffect(() => {
    if (hasFetched.current || authState !== "YES") return;
    hasFetched.current = true;
    void readList({ incremental: true }).catch((err) =>
      reportError(err, "device list"),
    );
    // The band and alarm counts are extras: each needs its own grant, and
    // a failure leaves its column blank rather than the page.
    if (canReadBeacon) {
      void readBeaconList().catch((err) => reportError(err, "band list"));
    }
    if (canReadAlert) {
      void readAlertList().catch((err) => reportError(err, "alert list"));
    }
  }, [
    authState,
    readList,
    readBeaconList,
    readAlertList,
    canReadBeacon,
    canReadAlert,
  ]);

  const sortedDeviceList = useMemo(
    () =>
      [...deviceList].sort((a, b) => {
        const order =
          sortColumn === "lastCheckinTs"
            ? (a.lastCheckinTs ?? 0) - (b.lastCheckinTs ?? 0)
            : a.deviceName.localeCompare(b.deviceName);
        return sortDirection === "asc" ? order : -order;
      }),
    [deviceList, sortColumn, sortDirection],
  );

  //============================^ HOOK LINE ^====================

  if (authState === "CHECKING") {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }
  if (authState !== "YES") {
    return <Container size={1280} px={32} py={32} mx="auto"><Alert color="red">Authentication required.</Alert></Container>;
  }
  if (loading) return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  if (error) return <Container size={1280} px={32} py={32} mx="auto"><Alert color="red">Error loading devices: {error}</Alert></Container>;

  const bandCount = (deviceId: number) =>
    beaconList.filter((beacon) => beacon.deviceId === deviceId).length;
  const openAlertCount = (deviceId: number) =>
    alertList.filter(
      (alert) => alert.deviceId === deviceId && isOpenAlert(alert),
    ).length;

  const handleDeleteClick = (deviceId: number) => {
    confirmDelete("device", () => void handleDeleteConfirm(deviceId));
  };

  const handleDeleteConfirm = async (deviceId: number) => {
    const device = deviceList.find((d) => d.deviceId === deviceId);
    if (!device) return;
    try {
      await deleteItem(device.deviceId, device.updateTs);
      notifySuccess("Device deleted successfully");
    } catch (error) {
      notifyFailure(error, "deleting device", "Failed to delete device");
    }
  };

  const handleSort = (column: string) => {
    const newDirection =
      sortColumn === column && sortDirection === "asc" ? "desc" : "asc";
    setListParams({ sort: column, dir: newDirection });
  };

  return (
    <Container size={1280} px={32} py={32} mx="auto">
      <Group justify="space-between" mb="lg">
        <Group gap={4}>
          <Title order={1}>Devices</Title>
          <FieldHelp id="setting-up-a-device" />
        </Group>
        {grantList.includes(grants.CREATE_DEVICE) && (
          <Button
            onClick={() => void navigate(`${ROUTES.DEVICE_EDIT}?deviceId=new`)}
          >
            Add Device
          </Button>
        )}
      </Group>
      {sortedDeviceList.length === 0 ? (
        <Alert color="gray">No devices yet.</Alert>
      ) : (
        <Table>
          <Table.Thead>
            <Table.Tr>
              <SortTh
                label="Name"
                column="deviceName"
                sortColumn={sortColumn}
                sortDirection={sortDirection}
                onSort={handleSort}
              />
              <SortTh
                label="Last check-in"
                column="lastCheckinTs"
                sortColumn={sortColumn}
                sortDirection={sortDirection}
                onSort={handleSort}
              />
              <Table.Th>Agent version</Table.Th>
              {canReadBeacon && <Table.Th>Bands</Table.Th>}
              {canReadAlert && <Table.Th>Open alarms</Table.Th>}
              <Table.Th>Actions</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {sortedDeviceList.map((device) => {
              const openCount = openAlertCount(device.deviceId);
              return (
                <Table.Tr key={device.deviceId}>
                  <Table.Td>{device.deviceName}</Table.Td>
                  <Table.Td title={formatTs(device.lastCheckinTs)}>
                    {formatAgo(device.lastCheckinTs)}
                  </Table.Td>
                  <Table.Td>{device.lastAgentVersion ?? ""}</Table.Td>
                  {canReadBeacon && (
                    <Table.Td>{bandCount(device.deviceId)}</Table.Td>
                  )}
                  {canReadAlert && (
                    <Table.Td>
                      {openCount > 0 ? (
                        <Badge
                          color="red"
                          component="a"
                          href={`${ROUTES.ALERT_LIST}?deviceId=${device.deviceId}`}
                          onClick={(event) => {
                            event.preventDefault();
                            void navigate(
                              `${ROUTES.ALERT_LIST}?deviceId=${device.deviceId}`,
                            );
                          }}
                          style={{ cursor: "pointer" }}
                        >
                          {openCount} open
                        </Badge>
                      ) : (
                        <Badge color="gray" variant="light">
                          none
                        </Badge>
                      )}
                    </Table.Td>
                  )}
                  <Table.Td>
                    <Button
                      size="xs"
                      onClick={() =>
                        void navigate(
                          `${ROUTES.DEVICE_EDIT}?deviceId=${device.deviceId}`,
                        )
                      }
                    >
                      Edit
                    </Button>
                    {grantList.includes(grants.DELETE_DEVICE) && (
                      <Button
                        size="xs"
                        color="red"
                        ml="xs"
                        onClick={() => handleDeleteClick(device.deviceId)}
                      >
                        Delete
                      </Button>
                    )}
                  </Table.Td>
                </Table.Tr>
              );
            })}
          </Table.Tbody>
        </Table>
      )}
    </Container>
  );
}

export default DeviceListPage;
