import { useState, useEffect, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { ROUTES } from "../constants/routes";
import {
  TextInput,
  Button,
  Title,
  Group,
  MultiSelect,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useUserGroupsStore } from "../stores/user-groups-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { DeleteConfirmModal } from "../components/DeleteConfirmModal";

function UserGroupEditPage() {
  const authState = useAuthGuard();
  const [searchParams] = useSearchParams();
  const { createUserGroup, updateUserGroup, deleteUserGroup, readGrantList } = useUserGroupsStore();
  const navigate = useNavigate();

  const userGroupId = searchParams.get("userGroupId") || "";
  const isNew = userGroupId === "new";
  const numericId = isNaN(Number(userGroupId)) ? null : Number(userGroupId);

  const [formData, setFormData] = useState({
    groupName: "",
    userGrantList: [] as number[],
    updateTs: "",
  });

  const [grants, setGrants] = useState<{ id: number; text: string }[]>([]);

  const hasFetched = useRef(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);

  useEffect(() => {
    const fetchGrants = async () => {
      try {
        const grantList = await readGrantList();
        setGrants(grantList);
      } catch (error) {
        console.error("Failed to fetch grants:", error);
      }
    };
    fetchGrants();
  }, [readGrantList]);

  useEffect(() => {
    if (!hasFetched.current && numericId) {
      hasFetched.current = true;
      // Find group from store
      const groups = useUserGroupsStore.getState().userGroups;
      const existingGroup = groups.find((g) => g.userGroupId === numericId);
      if (existingGroup) {
        setFormData({
          groupName: existingGroup.groupName,
          userGrantList: existingGroup.userGrantList,
          updateTs: existingGroup.updateTs.toString(),
        });
      }
    }
  }, [numericId]);

  if (authState !== "YES") {
    return <div>Checking Auth...</div>;
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const data = {
      groupName: formData.groupName,
      userGrantList: formData.userGrantList,
      updateTs: Number(formData.updateTs),
    };

    try {
      if (isNew) {
        await createUserGroup({
          groupName: formData.groupName,
          userGrantList: formData.userGrantList,
        });
        notifications.show({
          title: "Success",
          message: "User group created successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
      } else {
        await updateUserGroup(Number(userGroupId), data);
        notifications.show({
          title: "Success",
          message: "User group updated successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
      }
      navigate(`/${ROUTES.USER_GROUP_LIST}`);
    } catch (error) {
      console.error("Error saving user group:", error);
      notifications.show({
        title: "Error",
        message: "Failed to save user group",
        color: "red",
        autoClose: 5000,
        withCloseButton: true,
      });
    }
  };

  const handleDeleteClick = () => {
    setDeleteModalOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (numericId && !isNew) {
      try {
        await deleteUserGroup(numericId);
        notifications.show({
          title: "Success",
          message: "User group deleted successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
      navigate(`/${ROUTES.USER_GROUP_LIST}`);
      } catch (error) {
        console.error("Error deleting user group:", error);
        notifications.show({
          title: "Error",
          message: "Failed to delete user group",
          color: "red",
          autoClose: 5000,
          withCloseButton: true,
        });
      }
    }
    setDeleteModalOpen(false);
  };

  const handleDeleteCancel = () => {
    setDeleteModalOpen(false);
  };



  return (
    <div style={{ padding: "2rem" }}>
      <Title order={1} mb="lg">
        Edit User Group
      </Title>
      <form onSubmit={handleSubmit}>
        <TextInput
          label="Group Name"
          value={formData.groupName}
          onChange={(e) =>
            setFormData({ ...formData, groupName: e.target.value })
          }
          required
          mb="md"
        />
        <MultiSelect
          label="Roles"
          data={grants.map((g) => ({ value: g.id.toString(), label: g.text }))}
          value={formData.userGrantList.map(String)}
          onChange={(values) =>
            setFormData({ ...formData, userGrantList: values.map(Number) })
          }
          mb="md"
        />
        <Group>
          <Button type="submit">Save</Button>
          <Button color="red" onClick={handleDeleteClick}>
            Delete
          </Button>
           <Button
             variant="outline"
             onClick={() => navigate(`/${ROUTES.USER_GROUP_LIST}`)}
           >
             Cancel
           </Button>
        </Group>
      </form>
      <DeleteConfirmModal
        isOpen={deleteModalOpen}
        onConfirm={handleDeleteConfirm}
        onCancel={handleDeleteCancel}
        entityType="user group"
      />
    </div>
  );
}

export default UserGroupEditPage;