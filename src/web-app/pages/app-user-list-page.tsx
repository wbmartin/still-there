import { useState, useEffect, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Table, Button, Title, Group, Pagination } from "@mantine/core";
import { useAppUserStore } from "../stores/app-user-store";
import { useAppUserGroupStore } from "../stores/app-user-group-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { DeleteConfirmModal } from "../components/DeleteConfirmModal";
import { ROUTES } from "../constants/routes";
import { grants } from "../constants/grants";
import type { User } from "../types/user-type";

function UserAdminPage() {
  const authState = useAuthGuard([grants.READ_USER]);
  const { userList, loading, retrieveUserList, deleteUser } = useAppUserStore();
  const { getGroupName, retrieveAppUserGroupList } = useAppUserGroupStore();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [page, setPage] = useState(1);
  const itemsPerPage = 15;
  const hasFetched = useRef(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deleteId, setDeleteId] = useState<number | null>(null);

  // Sort state from URL params
  const sortColumn = searchParams.get("sort") || "appUserId";
  const sortDirection = (searchParams.get("dir") || "asc") as "asc" | "desc";

  useEffect(() => {
    if (!hasFetched.current && authState === "YES") {
      hasFetched.current = true;
      void retrieveUserList();
      void retrieveAppUserGroupList();
    }
    }, [authState, retrieveUserList, retrieveAppUserGroupList]);
  //============================^HOOOK LINE ^====================

  if (authState === "CHECKING") {
    return <div>Checking permissions...</div>;
  }
  if (authState !== "YES") {
    return <div>Checking Auth...</div>;
  }

  if (loading) return <div>Loading...</div>;

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
  const paginatedUserList = sortedUserList.slice(
    (page - 1) * itemsPerPage,
    page * itemsPerPage,
  );

  const handleDeleteClick = (id: number) => {
    setDeleteId(id);
    setDeleteModalOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (deleteId !== null) {
      try {
        await deleteUser(deleteId);
      } catch (error) {
        console.error("Error deleting user:", error);
      }
    }
    setDeleteModalOpen(false);
    setDeleteId(null);
  };

  const handleDeleteCancel = () => {
    setDeleteModalOpen(false);
    setDeleteId(null);
  };

  const handleSort = (column: string) => {
    const newDirection = sortColumn === column && sortDirection === "asc" ? "desc" : "asc";
    setSearchParams({ sort: column, dir: newDirection });
    setPage(1); // Reset to first page when sorting
  };

  return (
    <div style={{ padding: "2rem" }}>
      <Group justify="space-between" mb="lg">
        <Title order={1}>User Admin</Title>
        <Button
          onClick={() => { void navigate(`/${ROUTES.APP_USER_EDIT}?appUserId=new`); }}
        >
          Add User
        </Button>
      </Group>
      <Table>
        <Table.Thead>
          <Table.Tr>
            <Table.Th
              style={{ cursor: "pointer" }}
              onClick={() => handleSort("appUserId")}
            >
              ID {sortColumn === "appUserId" && (sortDirection === "asc" ? "↑" : "↓")}
            </Table.Th>
            <Table.Th
              style={{ cursor: "pointer" }}
              onClick={() => handleSort("userName")}
            >
              Username {sortColumn === "userName" && (sortDirection === "asc" ? "↑" : "↓")}
            </Table.Th>
            <Table.Th
              style={{ cursor: "pointer" }}
              onClick={() => handleSort("name")}
            >
              Name {sortColumn === "name" && (sortDirection === "asc" ? "↑" : "↓")}
            </Table.Th>
            <Table.Th
              style={{ cursor: "pointer" }}
              onClick={() => handleSort("email")}
            >
              Email {sortColumn === "email" && (sortDirection === "asc" ? "↑" : "↓")}
            </Table.Th>
            <Table.Th>Groups</Table.Th>
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
                <Button
                  size="xs"
                  onClick={() => {
                    void navigate(
                      `/${ROUTES.APP_USER_EDIT}?appUserId=${user.appUserId}`,
                    );
                  }}
                >
                  Edit
                </Button>
                <Button
                  size="xs"
                  color="red"
                  ml="xs"
                  onClick={() => { handleDeleteClick(user.appUserId); }}
                >
                  Delete
                </Button>
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
      <DeleteConfirmModal
        isOpen={deleteModalOpen}
        onConfirm={() => void handleDeleteConfirm()}
        onCancel={handleDeleteCancel}
        entityType="user"
      />
    </div>
  );
}

export default UserAdminPage;
