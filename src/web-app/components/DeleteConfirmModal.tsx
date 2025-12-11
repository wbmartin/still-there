import { Modal, Text, Button, Group } from "@mantine/core";

interface DeleteConfirmModalProps {
  isOpen: boolean;
  onConfirm: () => void;
  onCancel: () => void;
  entityType: string;
}

export function DeleteConfirmModal({
  isOpen,
  onConfirm,
  onCancel,
  entityType,
}: DeleteConfirmModalProps) {
  return (
    <Modal opened={isOpen} onClose={onCancel} title="Confirm Delete" centered size="sm">
      <Text>Are you sure you want to delete this {entityType}?</Text>
      <Group justify="center" mt="md" grow>
        <Button variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button color="red" onClick={onConfirm}>
          Delete
        </Button>
      </Group>
    </Modal>
  );
}