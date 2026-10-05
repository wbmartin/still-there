// A sortable list-page column header. The label is a real button, so a
// keyboard user can reach it with Tab and sort with Enter or Space; the
// <th> carries aria-sort for the column the list is ordered by. The page
// keeps its own sort state and order (in the URL; see README "Scaffolding")
// and hands this only the current column, direction and handler.
import { Group, Table, Text, UnstyledButton } from "@mantine/core";
import {
  IconChevronDown,
  IconChevronUp,
  IconSelector,
} from "@tabler/icons-react";

export type SortDirection = "asc" | "desc";

export function SortTh({
  label,
  column,
  sortColumn,
  sortDirection,
  onSort,
}: {
  label: string;
  column: string;
  sortColumn: string;
  sortDirection: SortDirection;
  onSort: (column: string) => void;
}) {
  const active = sortColumn === column;
  const Icon = active
    ? sortDirection === "asc"
      ? IconChevronUp
      : IconChevronDown
    : IconSelector;
  return (
    <Table.Th
      aria-sort={
        active
          ? sortDirection === "asc"
            ? "ascending"
            : "descending"
          : undefined
      }
    >
      <UnstyledButton onClick={() => onSort(column)} w="100%">
        <Group gap={4} justify="space-between" wrap="nowrap">
          <Text fw={700} size="sm">
            {label}
          </Text>
          <Icon
            size={14}
            aria-hidden
            style={{ opacity: active ? 1 : 0.4 }}
          />
        </Group>
      </UnstyledButton>
    </Table.Th>
  );
}
