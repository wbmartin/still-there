import { Text } from "@mantine/core";
import { modals } from "@mantine/modals";

export const confirmDelete = (
  entityType: string,
  onConfirm: () => void,
): void => {
  modals.openConfirmModal({
    title: "Confirm Delete",
    centered: true,
    children: <Text>Are you sure you want to delete this {entityType}?</Text>,
    labels: { confirm: "Delete", cancel: "Cancel" },
    confirmProps: { color: "red" },
    onConfirm,
  });
};
