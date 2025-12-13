import { useState, useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { Table, Button, Title, Group, Pagination } from "@mantine/core";
import { useAppUserStore } from "../stores/app-user-store";
import { useUserGroupsStore } from "../stores/user-groups-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { DeleteConfirmModal } from "../components/DeleteConfirmModal";
import { ROUTES } from "../constants/routes";
import type { User } from "../types/user-type";

function UserAdminPage() {
  const authState = useAuthGuard();
  const { userList, loading, readUserList, deleteUser } = useAppUserStore();
  const { getGroupName, readUserGroupList } = useUserGroupsStore();
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const itemsPerPage = 20;
  const hasFetched = useRef(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deleteId, setDeleteId] = useState<number | null>(null);

  useEffect(() => {
    if (!hasFetched.current && authState === "YES") {
      hasFetched.current = true;
      void readUserList();
      void readUserGroupList();
    }
  }, [authState, readUserList, readUserGroupList]);
  //============================^HOOOK LINE ^====================

  if (authState !== "YES") {
    return <div>Checking Auth...</div>;
  }

  if (loading) return <div>Loading...</div>;

  const totalPages = Math.ceil(userList.length / itemsPerPage);
  const paginatedUserList = userList.slice(
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

  return (
    <div style={{ padding: "2rem" }}>
      <Group justify="space-between" mb="lg">
        <Title order={1}>User Admin</Title>
        <Button
          onClick={() => { navigate(`/${ROUTES.APP_USER_EDIT}?appUserId=new`); }}
        >
          Add User
        </Button>
      </Group>
      <Table>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>ID</Table.Th>
            <Table.Th>Username</Table.Th>
            <Table.Th>Name</Table.Th>
            <Table.Th>Email</Table.Th>
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
                {user.userGroupMembershipList
                  .map((id) => getGroupName(id))
                  .join(", ")}
              </Table.Td>
              <Table.Td>
                <Button
                  size="xs"
                  onClick={() => {
                    navigate(
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
      {totalPages > 1 && (
        <Group justify="center" mt="lg">
          <Pagination total={totalPages} value={page} onChange={setPage} />
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
