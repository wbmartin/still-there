import { useState, useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
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
  const [page, setPage] = useState(1);
  const itemsPerPage = 15;
  const hasFetched = useRef(false);

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

  const totalPageCount = Math.ceil(appUserGroupList.length / itemsPerPage);
  const paginatedGroupList = appUserGroupList.slice(
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
              void navigate(`/${ROUTES.APP_USER_GROUP_EDIT}?appUserGroupId=new`)
            }
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
               <Table.Td style={{ maxWidth: '300px' }}>
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
