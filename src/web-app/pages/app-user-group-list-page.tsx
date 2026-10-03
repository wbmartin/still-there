import { useEffect, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import {
  Table,
  Button,
  Title,
  Group,
  Pagination,
  Badge,
  Flex,
  Container,
  Loader,
  Alert,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useAppUserGroupStore } from "../stores/app-user-group-store";
import { useAuthStore } from "../stores/auth-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { useRowsPerPage } from "../hooks/use-preference";
import { confirmDelete } from "../utils/confirm-delete";
import { ROUTES } from "../constants/routes";
import { grants } from "../constants/grants";
import type { AppUserGroup } from "../types/app-user-group-type";
import { reportError } from "../utils/report-error";

function AppUserGroupListPage() {
  const authState = useAuthGuard([grants.READ_USER_GROUP]);
  const { grantList } = useAuthStore();
  const {
    itemList: appUserGroupList,
    loading,
    readList: readAppUserGroupList,
    deleteItem: deleteAppUserGroup,
  } = useAppUserGroupStore();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  // Page and sort live in the URL so a refresh or a shared link keeps its
  // place. setListParams merges into the query string, so a parameter a
  // derived page adds (a filter) survives a sort or a page change; an
  // empty value removes its key.
  const setListParams = (update: Record<string, string>) =>
    setSearchParams((previous) => {
      const next = new URLSearchParams(previous);
      for (const [key, value] of Object.entries(update)) {
        if (value) next.set(key, value);
        else next.delete(key);
      }
      return next;
    });
  // As asked for; clamped to the last page once the list is known.
  const requestedPage = Math.max(
    1,
    Math.floor(Number(searchParams.get("page"))) || 1,
  );
  const setPage = (nextPage: number) =>
    setListParams({ page: nextPage > 1 ? String(nextPage) : "" });
  const itemsPerPage = useRowsPerPage();
  const hasFetched = useRef(false);

  // Sort state from URL params
  const sortColumn = searchParams.get("sort") || "appUserGroupId";
  const sortDirection = searchParams.get("dir") === "desc" ? "desc" : "asc";

  useEffect(() => {
    if (!hasFetched.current && authState === "YES") {
      hasFetched.current = true;
      void readAppUserGroupList().catch((err) => reportError(err, "user group list"));
    }
  }, [authState, readAppUserGroupList]);

  //============================^ HOOK LINE ^====================

  if (authState === "CHECKING") {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }
  if (authState !== "YES") {
    return <Container size={1280} px={32} py={32} mx="auto"><Alert color="red">Authentication required.</Alert></Container>;
  }

  if (loading) return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;

  // Sort groups
  const sortedGroupList = [...appUserGroupList].sort((a, b) => {
    let aValue: string | number;
    let bValue: string | number;

    switch (sortColumn) {
      case "appUserGroupId":
        aValue = a.appUserGroupId;
        bValue = b.appUserGroupId;
        break;
      case "groupName":
        aValue = a.groupName.toLowerCase();
        bValue = b.groupName.toLowerCase();
        break;
      default:
        return 0;
    }

    if (aValue < bValue) return sortDirection === "asc" ? -1 : 1;
    if (aValue > bValue) return sortDirection === "asc" ? 1 : -1;
    return 0;
  });

  const totalPageCount = Math.ceil(sortedGroupList.length / itemsPerPage);
  // A stale link, or deleting the last row of the last page, can ask for a
  // page past the end: show the last page rather than an empty table.
  const page = Math.min(requestedPage, Math.max(1, totalPageCount));
  const paginatedGroupList = sortedGroupList.slice(
    (page - 1) * itemsPerPage,
    page * itemsPerPage,
  );

  const handleDeleteClick = (id: number) => {
    confirmDelete("user group", () => void handleDeleteConfirm(id));
  };

  const handleDeleteConfirm = async (id: number) => {
    const group = appUserGroupList.find((g) => g.appUserGroupId === id);
    if (group) {
      try {
        await deleteAppUserGroup(group.appUserGroupId, group.updateTs);
        notifications.show({
          title: "Success",
          message: "User group deleted successfully",
          color: "green",
        });
      } catch (error) {
        reportError(error, "deleting user group");
        notifications.show({
          title: "Error",
          message: "Failed to delete user group",
          color: "red",
        });
      }
    }
  };

  const handleSort = (column: string) => {
    const newDirection =
      sortColumn === column && sortDirection === "asc" ? "desc" : "asc";
    // A new order starts from the first page.
    setListParams({ sort: column, dir: newDirection, page: "" });
  };

  // Authorization checks
  const canCreateUserGroup = grantList.includes(grants.CREATE_USER_GROUP);
  const canUpdateUserGroup = grantList.includes(grants.UPDATE_USER_GROUP);
  const canDeleteUserGroup = grantList.includes(grants.DELETE_USER_GROUP);

  return (
    <Container size={1280} px={32} py={32} mx="auto">
      <Group justify="space-between" mb="lg">
        <Title order={1}>User Group Admin</Title>
        {canCreateUserGroup && (
          <Button
            onClick={() =>
              void navigate(`${ROUTES.APP_USER_GROUP_EDIT}?appUserGroupId=new`)
            }
          >
            Add User Group
          </Button>
        )}
      </Group>
      <Table>
        <Table.Thead>
          <Table.Tr>
            <Table.Th
              className="sortable-column"
              onClick={() => handleSort("appUserGroupId")}
            >
              ID{" "}
              {sortColumn === "appUserGroupId" &&
                (sortDirection === "asc" ? "↑" : "↓")}
            </Table.Th>
            <Table.Th
              className="sortable-column"
              onClick={() => handleSort("groupName")}
            >
              Group Name{" "}
              {sortColumn === "groupName" &&
                (sortDirection === "asc" ? "↑" : "↓")}
            </Table.Th>
            <Table.Th>Roles</Table.Th>
            <Table.Th>Actions</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {paginatedGroupList.map((group: AppUserGroup) => (
            <Table.Tr key={group.appUserGroupId}>
              <Table.Td>{group.appUserGroupId}</Table.Td>
              <Table.Td>{group.groupName}</Table.Td>
              <Table.Td maw={300}>
                 <Flex gap="xs" wrap="wrap">
                   {group.userGrantList.map((id) => (
                     <Badge key={id} variant="light" size="sm">
                       {grants.idToName(id)}
                     </Badge>
                   ))}
                 </Flex>
               </Table.Td>
               <Table.Td>
                 <Group gap="xs">
                   {canUpdateUserGroup && (
                     <Button
                       size="xs"
                       onClick={() =>
                         void navigate(
                           `${ROUTES.APP_USER_GROUP_EDIT}?appUserGroupId=${group.appUserGroupId}`,
                         )
                       }
                     >
                       Edit
                     </Button>
                   )}
                   {canDeleteUserGroup && (
                     <Button
                       size="xs"
                       color="red"
                       onClick={() => handleDeleteClick(group.appUserGroupId)}
                     >
                       Delete
                     </Button>
                   )}
                 </Group>
               </Table.Td>
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
      {totalPageCount > 1 && (
        <Group justify="center" mt="lg">
          <Pagination total={totalPageCount} value={page} onChange={setPage} />
        </Group>
      )}
    </Container>
  );
}

export default AppUserGroupListPage;
