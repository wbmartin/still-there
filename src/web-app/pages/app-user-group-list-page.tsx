import { useState, useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { Table, Button, Title, Group, Pagination, Badge, Flex } from "@mantine/core";
import { useAppUserGroupStore } from "../stores/app-user-group-store";
import { useAuthStore } from "../stores/auth-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { DeleteConfirmModal } from "../components/DeleteConfirmModal";
import { ROUTES } from "../constants/routes";
import { grants } from "../constants/grants";
import type { AppUserGroup } from "../types/app-user-group-type";

function AppUserGroupListPage() {
  const authState = useAuthGuard();
  const { grantList } = useAuthStore();
  const { appUserGroupList, loading, retrieveAppUserGroupList, deleteAppUserGroup } =
    useAppUserGroupStore();
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const itemsPerPage = 15;
  const hasFetched = useRef(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deleteId, setDeleteId] = useState<number | null>(null);

  useEffect(() => {
    if (!hasFetched.current && authState === "YES") {
      hasFetched.current = true;
      void retrieveAppUserGroupList();
    }
    }, [authState, retrieveAppUserGroupList]);

  //============================^HOOOK LINE ^====================

  if (authState !== "YES") {
    return <div>Checking Auth...</div>;
  }

  if (loading) return <div>Loading...</div>;

  const totalPageCount = Math.ceil(appUserGroupList.length / itemsPerPage);
  const paginatedGroupList = appUserGroupList.slice(
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
        await deleteAppUserGroup(deleteId);
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

  // Authorization checks
  const canCreateUserGroup = grantList.includes(grants.CREATE_USER_GROUP);
  const canUpdateUserGroup = grantList.includes(grants.UPDATE_USER_GROUP);
  const canDeleteUserGroup = grantList.includes(grants.DELETE_USER_GROUP);

  return (
    <div style={{ padding: "2rem" }}>
      <Group justify="space-between" mb="lg">
        <Title order={1}>User Group Admin</Title>
        {canCreateUserGroup && (
          <Button
            onClick={() => void navigate(`/${ROUTES.APP_USER_GROUP_EDIT}?appUserGroupId=new`)}
          >
            Add User Group
          </Button>
        )}
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
          {paginatedGroupList.map((group: AppUserGroup) => (
            <Table.Tr key={group.appUserGroupId}>
              <Table.Td>{group.appUserGroupId}</Table.Td>
              <Table.Td>{group.groupName}</Table.Td>
               <Table.Td>
                 <Flex gap="xs" wrap="wrap">
                   {group.userGrantList.map(id => (
                     <Badge key={id} variant="light" size="sm">
                       {grants.idToName(id)}
                     </Badge>
                   ))}
                 </Flex>
               </Table.Td>
               <Table.Td>
                 {canUpdateUserGroup && (
                   <Button
                     size="xs"
                     onClick={() =>
                       void navigate(
                         `/${ROUTES.APP_USER_GROUP_EDIT}?appUserGroupId=${group.appUserGroupId}`,
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
                     ml="xs"
                     onClick={() => handleDeleteClick(group.appUserGroupId)}
                   >
                     Delete
                   </Button>
                 )}
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

export default AppUserGroupListPage;
