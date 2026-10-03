import { useEffect, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import {
  Table,
  Button,
  Title,
  Group,
  Pagination,
  Container,
  Loader,
  Alert,
  Badge,
} from "@mantine/core";
import { useAppUserStore } from "../stores/app-user-store";
import { useAppUserGroupStore } from "../stores/app-user-group-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { useRowsPerPage } from "../hooks/use-preference";
import { confirmDelete } from "../utils/confirm-delete";
import { ROUTES } from "../constants/routes";
import { grants } from "../constants/grants";
import type { User } from "../types/user-type";
import { reportError } from "../utils/report-error";

function UserAdminPage() {
  const authState = useAuthGuard([grants.READ_USER]);
  const {
    itemList: userList,
    loading,
    readList: readUserList,
    deleteItem: deleteUser,
  } = useAppUserStore();
  const { getGroupName, readList: readAppUserGroupList } =
    useAppUserGroupStore();
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
  const sortColumn = searchParams.get("sort") || "appUserId";
  const sortDirection = searchParams.get("dir") === "desc" ? "desc" : "asc";

  useEffect(() => {
    if (!hasFetched.current && authState === "YES") {
      hasFetched.current = true;
      void readUserList().catch((err) => reportError(err, "user list"));
      void readAppUserGroupList().catch((err) => reportError(err, "user list"));
    }
  }, [authState, readUserList, readAppUserGroupList]);
  //============================^ HOOK LINE ^====================

  if (authState === "CHECKING") {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }
  if (authState !== "YES") {
    return <Container size={1280} px={32} py={32} mx="auto"><Alert color="red">Authentication required.</Alert></Container>;
  }

  if (loading) return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;

  // Sort users
  const sortedUserList = [...userList].sort((a, b) => {
    let aValue: string | number;
    let bValue: string | number;

    switch (sortColumn) {
      case "appUserId":
        aValue = a.appUserId;
        bValue = b.appUserId;
        break;
      case "userName":
        aValue = a.userName.toLowerCase();
        bValue = b.userName.toLowerCase();
        break;
      case "name":
        aValue = `${a.firstName} ${a.lastName}`.toLowerCase();
        bValue = `${b.firstName} ${b.lastName}`.toLowerCase();
        break;
      case "email":
        aValue = a.email.toLowerCase();
        bValue = b.email.toLowerCase();
        break;
      default:
        return 0;
    }

    if (aValue < bValue) return sortDirection === "asc" ? -1 : 1;
    if (aValue > bValue) return sortDirection === "asc" ? 1 : -1;
    return 0;
  });

  const totalPageCount = Math.ceil(sortedUserList.length / itemsPerPage);
  // A stale link, or deleting the last row of the last page, can ask for a
  // page past the end: show the last page rather than an empty table.
  const page = Math.min(requestedPage, Math.max(1, totalPageCount));
  const paginatedUserList = sortedUserList.slice(
    (page - 1) * itemsPerPage,
    page * itemsPerPage,
  );

  const handleDeleteClick = (id: number) => {
    confirmDelete("user", () => void handleDeleteConfirm(id));
  };

  const handleDeleteConfirm = async (id: number) => {
    const user = userList.find((u) => u.appUserId === id);
    if (user) {
      try {
        await deleteUser(user.appUserId, user.updateTs);
      } catch (error) {
        reportError(error, "deleting user");
      }
    }
  };

  const handleSort = (column: string) => {
    const newDirection =
      sortColumn === column && sortDirection === "asc" ? "desc" : "asc";
    // A new order starts from the first page.
    setListParams({ sort: column, dir: newDirection, page: "" });
  };

  return (
    <Container size={1280} px={32} py={32} mx="auto">
      <Group justify="space-between" mb="lg">
        <Title order={1}>User Admin</Title>
        <Button
          onClick={() => {
            void navigate(`${ROUTES.APP_USER_EDIT}?appUserId=new`);
          }}
        >
          Add User
        </Button>
      </Group>
      <Table>
        <Table.Thead>
          <Table.Tr>
            <Table.Th
              className="sortable-column"
              onClick={() => handleSort("appUserId")}
            >
              ID{" "}
              {sortColumn === "appUserId" &&
                (sortDirection === "asc" ? "↑" : "↓")}
            </Table.Th>
            <Table.Th
              className="sortable-column"
              onClick={() => handleSort("userName")}
            >
              Username{" "}
              {sortColumn === "userName" &&
                (sortDirection === "asc" ? "↑" : "↓")}
            </Table.Th>
            <Table.Th
              className="sortable-column"
              onClick={() => handleSort("name")}
            >
              Name{" "}
              {sortColumn === "name" && (sortDirection === "asc" ? "↑" : "↓")}
            </Table.Th>
            <Table.Th
              className="sortable-column"
              onClick={() => handleSort("email")}
            >
              Email{" "}
              {sortColumn === "email" && (sortDirection === "asc" ? "↑" : "↓")}
            </Table.Th>
            <Table.Th>Groups</Table.Th>
            <Table.Th>Status</Table.Th>
            <Table.Th>Actions</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {paginatedUserList.map((user: User) => (
            <Table.Tr key={user.appUserId}>
              <Table.Td>{user.appUserId}</Table.Td>
              <Table.Td>{user.userName}</Table.Td>
              <Table.Td>
                {user.firstName} {user.lastName}
              </Table.Td>
              <Table.Td>{user.email}</Table.Td>
              <Table.Td>
                {user.appUserGroupMembershipList
                  .map((id) => getGroupName(id))
                  .join(", ")}
              </Table.Td>
              <Table.Td>
                {user.accountStatus === "blocked" ? (
                  <Badge color="red" variant="light">Blocked</Badge>
                ) : user.accountStatus === "invited" ? (
                  <Badge color="blue" variant="light">Invited</Badge>
                ) : (
                  <Badge color="gray" variant="light">Active</Badge>
                )}
              </Table.Td>
              <Table.Td>
                <Group gap="xs" wrap="nowrap">
                  <Button
                    size="xs"
                    onClick={() => {
                      void navigate(
                        `${ROUTES.APP_USER_EDIT}?appUserId=${user.appUserId}`,
                      );
                    }}
                  >
                    Edit
                  </Button>
                  <Button
                    size="xs"
                    color="red"
                    onClick={() => {
                      handleDeleteClick(user.appUserId);
                    }}
                  >
                    Delete
                  </Button>
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

export default UserAdminPage;
