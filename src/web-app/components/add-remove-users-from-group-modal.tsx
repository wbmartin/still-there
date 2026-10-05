import { useState, useEffect } from "react";
import { Modal, Button, Group, MultiSelect, Title } from "@mantine/core";
import { useAppUserStore } from "../stores/app-user-store";
import { reportError } from "../utils/report-error";

interface AddRemoveUsersFromGroupModalProps {
  opened: boolean;
  onClose: () => void;
  groupId: number | null;
  mode: "add" | "remove";
  onSubmit: (userIdList: number[]) => Promise<void>;
}

export function AddRemoveUsersFromGroupModal({
  opened,
  onClose,
  groupId,
  mode,
  onSubmit,
}: AddRemoveUsersFromGroupModalProps) {
  const { itemList: userList, readList: readUserList } = useAppUserStore();
  const [selectedUsers, setSelectedUsers] = useState<number[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (opened) {
      // Fetch all users for the security context
      void readUserList().catch((err) => reportError(err, "group membership modal"));
    }
  }, [opened, readUserList]);
  // ============================^ HOOK LINE ^====================

  const availableUserList = userList.filter((user) => {
    if (mode === "add") {
      return !user.appUserGroupMembershipList.includes(groupId || 0);
    } else {
      return user.appUserGroupMembershipList.includes(groupId || 0);
    }
  });

  const userOptions = availableUserList.map((user) => ({
    value: user.appUserId.toString(),
    label: `${user.firstName} ${user.lastName} (${user.userName})`,
  }));

  const handleConfirm = async () => {
    if (selectedUsers.length === 0) return;
    setLoading(true);
    try {
      await onSubmit(selectedUsers);
      setSelectedUsers([]);
      onClose();
    } catch (error) {
      reportError(error, "updating users");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={mode === "add" ? "Add Users to Group" : "Remove Users from Group"}
      size="lg"
    >
      <Title order={4} mb="md">
        {mode === "add" ? "Select users to add:" : "Select users to remove:"}
      </Title>
      <MultiSelect
        data={userOptions}
        value={selectedUsers.map(String)}
        onChange={(values) => setSelectedUsers(values.map(Number))}
        placeholder="Choose users..."
        searchable
        mb="md"
      />
      <Group justify="flex-end">
        <Button variant="outline" onClick={onClose}>
          Cancel
        </Button>
        <Button
          onClick={() => void handleConfirm()}
          loading={loading}
          disabled={selectedUsers.length === 0}
        >
          {mode === "add" ? "Add Users" : "Remove Users"}
        </Button>
      </Group>
    </Modal>
  );
}
