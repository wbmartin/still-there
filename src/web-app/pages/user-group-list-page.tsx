import { useState, useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { Table, Button, Title, Group, Pagination } from "@mantine/core";
import { useUserGroupsStore } from "../stores/user-groups-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { DeleteConfirmModal } from "../components/DeleteConfirmModal";
import { ROUTES } from "../constants/routes";
import type { UserGroup } from "../types/user-group-type";

function UserGroupAdminPage() {
  const authState = useAuthGuard();
  const { userGroupList, loading, retrieveUserGroupList, deleteUserGroup } =
    useUserGroupsStore();
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const itemsPerPage = 20;
  const hasFetched = useRef(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deleteId, setDeleteId] = useState<number | null>(null);

  useEffect(() => {
    if (!hasFetched.current && authState === "YES") {
      hasFetched.current = true;
      void retrieveUserGroupList();
    }
    }, [authState, retrieveUserGroupList]);

  //============================^HOOOK LINE ^====================

  if (authState !== "YES") {
    return <div>Checking Auth...</div>;
  }

  if (loading) return <div>Loading...</div>;

  const totalPageCount = Math.ceil(userGroupList.length / itemsPerPage);
  const paginatedGroupList = userGroupList.slice(
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
        await deleteUserGroup(deleteId);
      } catch (error) {
        console.error("Error deleting user group:", error);
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
        <Title order={1}>User Group Admin</Title>
        <Button
          onClick={() => void navigate(`/${ROUTES.USER_GROUP_EDIT}?userGroupId=new`)}
        >
          Add User Group
        </Button>
      </Group>
      <Table>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>ID</Table.Th>
            <Table.Th>Group Name</Table.Th>
            <Table.Th>Roles</Table.Th>
            <Table.Th>Actions</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {paginatedGroupList.map((group: UserGroup) => (
            <Table.Tr key={group.userGroupId}>
              <Table.Td>{group.userGroupId}</Table.Td>
              <Table.Td>{group.groupName}</Table.Td>
              <Table.Td>{group.userGrantList.join(", ")}</Table.Td>
              <Table.Td>
                <Button
                  size="xs"
                  onClick={() =>
                    void navigate(
                      `/${ROUTES.USER_GROUP_EDIT}?userGroupId=${group.userGroupId}`,
                    )
                  }
                >
                  Edit
                </Button>
                <Button
                  size="xs"
                  color="red"
                  ml="xs"
                  onClick={() => handleDeleteClick(group.userGroupId)}
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
        entityType="user group"
      />
    </div>
  );
}

export default UserGroupAdminPage;
