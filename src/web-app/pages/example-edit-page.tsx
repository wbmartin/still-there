import { useState, useEffect, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import {
  TextInput,
  NumberInput,
  Textarea,
  Button,
  Title,
  Group,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { ROUTES } from "../constants/routes";
import { useExamplesStore } from "../stores/examples-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { DeleteConfirmModal } from "../components/DeleteConfirmModal";
import { grants } from "../constants/grants";
import type { ExampleFormData } from "../types/example-type";
import { defaultExampleFormData } from "../types/example-type";

function ExampleEditPage() {
  const [searchParams] = useSearchParams();
  const id = searchParams.get("exampleId") || "";
  const isNew = id === "new";
  const requiredGrant = isNew ? grants.CREATE_EXAMPLE : grants.UPDATE_EXAMPLE;
  const authState = useAuthGuard([requiredGrant]);
  const {
    retrieveExampleById,
    createExample,
    updateExample,
    deleteExample,
  } = useExamplesStore();
  const navigate = useNavigate();

  const numericId = isNaN(Number(id)) ? null : Number(id);

  const [formData, setFormData] =
    useState<ExampleFormData>(defaultExampleFormData);
  const [exampleLoading, setExampleLoading] = useState(
    !isNew && numericId !== null,
  );
  const hasFetched = useRef(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);

  useEffect(() => {
    if (numericId && !isNew && !hasFetched.current) {
      hasFetched.current = true;
      void retrieveExampleById(numericId)
        .then((existingExample) => {
          if (existingExample) {
            const newFormData = {
              exampleText: existingExample.exampleText,
              exampleInt: existingExample.exampleInt?.toString() || "",
              exampleFloat: existingExample.exampleFloat?.toString() || "",
              exampleIntArray:
                existingExample.exampleIntArray?.join(", ") || "",
              exampleFloatArray:
                existingExample.exampleFloatArray?.join(", ") || "",
              updateTs: existingExample.updateTs.toString(),
            };
            setFormData((prev) => {
              // Only update if different to prevent cascading renders
              if (
                prev.exampleText !== newFormData.exampleText ||
                prev.exampleInt !== newFormData.exampleInt ||
                prev.exampleFloat !== newFormData.exampleFloat ||
                prev.exampleIntArray !== newFormData.exampleIntArray ||
                prev.exampleFloatArray !== newFormData.exampleFloatArray ||
                prev.updateTs !== newFormData.updateTs
              ) {
                return newFormData;
              }
              return prev;
            });
          }
        })
        .finally(() => {
          setExampleLoading(false);
        });
    }
  }, [
    numericId,
    isNew,
    retrieveExampleById,
  ]);

  //============================^HOOOK LINE ^====================
  if (authState !== "YES") {
    return <div>Checking Auth...</div>;
  }

  if (exampleLoading) {
    return <div>Loading example...</div>;
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const data = {
      exampleText: formData.exampleText,
      exampleInt: formData.exampleInt ? Number(formData.exampleInt) : undefined,
      exampleFloat: formData.exampleFloat
        ? Number(formData.exampleFloat)
        : undefined,
      exampleIntArray: formData.exampleIntArray
        ? formData.exampleIntArray.split(",").map(Number)
        : undefined,
      exampleFloatArray: formData.exampleFloatArray
        ? formData.exampleFloatArray.split(",").map(Number)
        : undefined,
      updateTs: !isNew ? Number(formData.updateTs) : undefined,
    };

    try {
      if (isNew) {
        await createExample(data);
        notifications.show({
          title: "Success",
          message: "Example created successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
      } else {
        await updateExample(Number(id), data);
        notifications.show({
          title: "Success",
          message: "Example updated successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
      }
      void navigate("/exampleList");
    } catch (error) {
      console.error("Error saving example:", error);
      notifications.show({
        title: "Error",
        message: "Failed to save example",
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
    if (numericId) {
      try {
        await deleteExample(numericId);
        notifications.show({
          title: "Success",
          message: "Example deleted successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
        void navigate(`/${ROUTES.EXAMPLE_LIST}`);
      } catch (error) {
        console.error("Error deleting example:", error);
        notifications.show({
          title: "Error",
          message: "Failed to delete example",
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
        {isNew ? "Add Example" : "Edit Example"}
      </Title>
      <form onSubmit={(e) => void handleSubmit(e)}>
        <TextInput
          label="Text"
          value={formData.exampleText}
          onChange={(e) =>
            setFormData({ ...formData, exampleText: e.target.value })
          }
          required
          mb="md"
        />
        <NumberInput
          label="Int"
          value={formData.exampleInt}
          onChange={(value) =>
            setFormData({ ...formData, exampleInt: value?.toString() || "" })
          }
          mb="md"
        />
        <NumberInput
          label="Float"
          value={formData.exampleFloat}
          onChange={(value) =>
            setFormData({ ...formData, exampleFloat: value?.toString() || "" })
          }
          mb="md"
        />
        <Textarea
          label="Int Array (comma-separated)"
          value={formData.exampleIntArray}
          onChange={(e) =>
            setFormData({ ...formData, exampleIntArray: e.target.value })
          }
          mb="md"
        />
        <Textarea
          label="Float Array (comma-separated)"
          value={formData.exampleFloatArray}
          onChange={(e) =>
            setFormData({ ...formData, exampleFloatArray: e.target.value })
          }
          mb="md"
        />
        <Group>
           <Button type="submit">Save</Button>
           {!isNew && (
             <Button color="red" onClick={handleDeleteClick}>
               Delete
             </Button>
           )}
           <Button
             variant="outline"
              onClick={() => void navigate("/exampleList")}
           >
             Cancel
           </Button>
         </Group>
      </form>
      <DeleteConfirmModal
        isOpen={deleteModalOpen}
        onConfirm={() => void handleDeleteConfirm()}
        onCancel={handleDeleteCancel}
        entityType="example"
      />
    </div>
  );
}

export default ExampleEditPage;
