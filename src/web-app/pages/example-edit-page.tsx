import { useState, useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { TextInput, NumberInput, Textarea, Button, Title, Group } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useGetExampleQuery, useCreateExampleMutation, useUpdateExampleMutation } from "../services/example-svc";
import useUnauthRedirect from "../hooks/unauth-redirect";

function ExampleEditPage() {
  const [searchParams] = useSearchParams();
  const id = searchParams.get('example_id') || '';
  const isNew = id === "new";
  useUnauthRedirect(`/example?example_id=${id}`);
  const navigate = useNavigate();

  const { data: example, isLoading } = useGetExampleQuery(Number(id), { skip: isNew });
  const [createExample] = useCreateExampleMutation();
  const [updateExample] = useUpdateExampleMutation();

  const [formData, setFormData] = useState({
    exampleText: "",
    exampleInt: "",
    exampleFloat: "",
    exampleIntArray: "",
    exampleFloatArray: "",
    updateTs: "",
  });

  useEffect(() => {
    if (example && !isNew) {
      setFormData({
        exampleText: example.exampleText,
        exampleInt: example.exampleInt?.toString() || "",
        exampleFloat: example.exampleFloat?.toString() || "",
        exampleIntArray: example.exampleIntArray?.join(", ") || "",
        exampleFloatArray: example.exampleFloatArray?.join(", ") || "",
        updateTs: example.updateTs.toString(),
      });
    }
  }, [example, isNew]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const data = {
      exampleText: formData.exampleText,
      exampleInt: formData.exampleInt ? Number(formData.exampleInt) : undefined,
      exampleFloat: formData.exampleFloat ? Number(formData.exampleFloat) : undefined,
      exampleIntArray: formData.exampleIntArray ? formData.exampleIntArray.split(",").map(Number) : undefined,
      exampleFloatArray: formData.exampleFloatArray ? formData.exampleFloatArray.split(",").map(Number) : undefined,
      updateTs: !isNew ? Number(formData.updateTs) : undefined,
    };

    try {
      if (isNew) {
        await createExample(data).unwrap();
        notifications.show({
          title: "Success",
          message: "Example created successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
      } else {
        await updateExample({ id: Number(id), data }).unwrap();
        notifications.show({
          title: "Success",
          message: "Example updated successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
      }
      navigate("/exampleList");
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

  if (isLoading) return <div>Loading...</div>;

  return (
    <div style={{ padding: "2rem" }}>
      <Title order={1} mb="lg">
        {isNew ? "Add Example" : "Edit Example"}
      </Title>
      <form onSubmit={handleSubmit}>
        <TextInput
          label="Text"
          value={formData.exampleText}
          onChange={(e) => setFormData({ ...formData, exampleText: e.target.value })}
          required
          mb="md"
        />
        <NumberInput
          label="Int"
          value={formData.exampleInt}
          onChange={(value) => setFormData({ ...formData, exampleInt: value?.toString() || "" })}
          mb="md"
        />
        <NumberInput
          label="Float"
          value={formData.exampleFloat}
          onChange={(value) => setFormData({ ...formData, exampleFloat: value?.toString() || "" })}
          mb="md"
        />
        <Textarea
          label="Int Array (comma-separated)"
          value={formData.exampleIntArray}
          onChange={(e) => setFormData({ ...formData, exampleIntArray: e.target.value })}
          mb="md"
        />
        <Textarea
          label="Float Array (comma-separated)"
          value={formData.exampleFloatArray}
          onChange={(e) => setFormData({ ...formData, exampleFloatArray: e.target.value })}
          mb="md"
        />
        <Group>
          <Button type="submit">Save</Button>
          <Button variant="outline" onClick={() => navigate("/exampleList")}>
            Cancel
          </Button>
        </Group>
      </form>
    </div>
  );
}

export default ExampleEditPage;