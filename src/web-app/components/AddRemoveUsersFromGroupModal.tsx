import { useState, useEffect } from "react";
import { Modal, Button, Group, MultiSelect, Title } from "@mantine/core";
import { useAppUserStore } from "../stores/app-user-store";

interface AddRemoveUsersFromGroupModalProps {
  isOpen: boolean;
  onClose: () => void;
  groupId: number | null;
  mode: "add" | "remove";
  onConfirm: (userIds: number[]) => Promise<void>;
}

function AddRemoveUsersFromGroupModal({
  isOpen,
  onClose,
  groupId,
  mode,
  onConfirm,
}: AddRemoveUsersFromGroupModalProps) {
  const { users, readUserList } = useAppUserStore();
  const [selectedUsers, setSelectedUsers] = useState<number[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (isOpen) {
      readUserList(); // Fetch all users for the security context
    }
  }, [isOpen, readUserList]);
  // ============================^HOOOK LINE ^====================

  const availableUsers = users.filter((user) => {
    if (mode === "add") {
      return !user.userGroupMembershipList.includes(groupId || 0);
    } else {
      return user.userGroupMembershipList.includes(groupId || 0);
    }
  });

  const userOptions = availableUsers.map((user) => ({
    value: user.appUserId.toString(),
    label: `${user.firstName} ${user.lastName} (${user.userName})`,
  }));

  const handleConfirm = async () => {
    if (selectedUsers.length === 0) return;
    setLoading(true);
    try {
      await onConfirm(selectedUsers);
      setSelectedUsers([]);
      onClose();
    } catch (error) {
      console.error("Error updating users:", error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      opened={isOpen}
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
          onClick={handleConfirm}
          loading={loading}
          disabled={selectedUsers.length === 0}
        >
          {mode === "add" ? "Add Users" : "Remove Users"}
        </Button>
      </Group>
    </Modal>
  );
}

export default AddRemoveUsersFromGroupModal;
