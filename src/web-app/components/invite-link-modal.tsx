import {
  Alert,
  Button,
  CopyButton,
  Group,
  Modal,
  Text,
  TextInput,
} from "@mantine/core";
import { buildInviteLink, formatInviteExpiry } from "../utils/invite-link";
import type { IssuedInvite } from "../types/user-type";

// Shows a freshly issued invite link once (B035). The API keeps only the
// token's hash, so this is the one time the link can be copied; closing the
// modal loses it, and a new link is the remedy. No e-mail is sent: the
// admin sends the link themselves.

interface InviteLinkModalProps {
  invite: IssuedInvite | null;
  onClose: () => void;
}

export function InviteLinkModal({ invite, onClose }: InviteLinkModalProps) {
  const link = invite
    ? buildInviteLink(window.location.origin, invite.inviteToken)
    : "";
  return (
    <Modal
      opened={invite !== null}
      onClose={onClose}
      title="Invite link"
      size="lg"
    >
      {invite && (
        <>
          <Text size="sm" mb="sm">
            Send this link to <b>{invite.userName}</b>. It works once and
            expires {formatInviteExpiry(invite.expiresTs)}.
          </Text>
          <Group gap="xs" wrap="nowrap" mb="md" align="flex-end">
            <TextInput
              value={link}
              readOnly
              aria-label="Invite link"
              style={{ flex: 1 }}
              onFocus={(event) => event.currentTarget.select()}
            />
            <CopyButton value={link}>
              {({ copied, copy }) => (
                <Button color={copied ? "teal" : undefined} onClick={copy}>
                  {copied ? "Copied" : "Copy link"}
                </Button>
              )}
            </CopyButton>
          </Group>
          <Alert color="yellow" variant="light" mb="md">
            This is the only time the link is shown. If it is lost, issue a
            new one from the user&apos;s page; the old link then stops
            working.
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
