import {
  Alert,
  Button,
  CopyButton,
  Group,
  Modal,
  Text,
  TextInput,
} from "@mantine/core";
import type { DeviceTokenIssued } from "../types/device-type";

// Shows a freshly issued device token once, as invite-link-modal.tsx shows
// an invite link. The API keeps only the token's hash, so this is the one
// time it can be copied; closing the modal loses it, and a new token is the
// remedy.

interface DeviceTokenModalProps {
  issued: DeviceTokenIssued | null;
  deviceName: string;
  onClose: () => void;
}

export function DeviceTokenModal({
  issued,
  deviceName,
  onClose,
}: DeviceTokenModalProps) {
  const token = issued?.deviceToken ?? "";
  return (
    <Modal
      opened={issued !== null}
      onClose={onClose}
      title="Device token"
      size="lg"
    >
      {issued && (
        <>
          <Text size="sm" mb="sm">
            Put this token in the config file of <b>{deviceName}</b>&apos;s
            base station.
          </Text>
          <Group gap="xs" wrap="nowrap" mb="md" align="flex-end">
            <TextInput
              value={token}
              readOnly
              aria-label="Device token"
              ff="monospace"
              style={{ flex: 1 }}
              onFocus={(event) => event.currentTarget.select()}
            />
            <CopyButton value={token}>
              {({ copied, copy }) => (
                <Button color={copied ? "teal" : undefined} onClick={copy}>
                  {copied ? "Copied" : "Copy token"}
                </Button>
              )}
            </CopyButton>
          </Group>
          <Alert color="yellow" variant="light" mb="md">
            This is the only time the token is shown. Any token issued
            before it has stopped working: a base station still using the
            old one can no longer check in. If this one is lost, issue a new
            one.
          </Alert>
          <Group justify="flex-end">
            <Button variant="default" onClick={onClose}>
              Done
            </Button>
          </Group>
        </>
      )}
    </Modal>
  );
}
