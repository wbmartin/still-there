import { useState, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { TextInput, NumberInput, Textarea, Button, Title, Group } from "@mantine/core";
import { useGetExampleQuery, useCreateExampleMutation, useUpdateExampleMutation } from "../services/example-svc";
import useUnauthRedirect from "../hooks/unauth-redirect";

function ExampleEditPage() {
  useUnauthRedirect(`/examples/${id}`);
  const { id } = useParams<{ id: string }>();
  const isNew = id === "new";
  const navigate = useNavigate();

  const { data: example, isLoading } = useGetExampleQuery(Number(id), { skip: isNew });
  const [createExample] = useCreateExampleMutation();
  const [updateExample] = useUpdateExampleMutation();

  const [formData, setFormData] = useState({
    example_text: "",
    example_int: "",
    example_float: "",
    example_int_array: "",
    example_float_array: "",
  });

  useEffect(() => {
    if (example && !isNew) {
      setFormData({
        example_text: example.example_text,
        example_int: example.example_int?.toString() || "",
        example_float: example.example_float?.toString() || "",
        example_int_array: example.example_int_array?.join(", ") || "",
        example_float_array: example.example_float_array?.join(", ") || "",
      });
    }
  }, [example, isNew]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const data = {
      example_text: formData.example_text,
      example_int: formData.example_int ? Number(formData.example_int) : undefined,
      example_float: formData.example_float ? Number(formData.example_float) : undefined,
      example_int_array: formData.example_int_array ? formData.example_int_array.split(",").map(Number) : undefined,
      example_float_array: formData.example_float_array ? formData.example_float_array.split(",").map(Number) : undefined,
    };

    try {
      if (isNew) {
        await createExample(data).unwrap();
      } else {
        await updateExample({ id: Number(id), data }).unwrap();
      }
      navigate("/examples");
    } catch (error) {
      console.error("Error saving example:", error);
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
          value={formData.example_text}
          onChange={(e) => setFormData({ ...formData, example_text: e.target.value })}
          required
          mb="md"
        />
        <NumberInput
          label="Int"
          value={formData.example_int}
          onChange={(value) => setFormData({ ...formData, example_int: value?.toString() || "" })}
          mb="md"
        />
        <NumberInput
          label="Float"
          value={formData.example_float}
          onChange={(value) => setFormData({ ...formData, example_float: value?.toString() || "" })}
          mb="md"
        />
        <Textarea
          label="Int Array (comma-separated)"
          value={formData.example_int_array}
          onChange={(e) => setFormData({ ...formData, example_int_array: e.target.value })}
          mb="md"
        />
        <Textarea
          label="Float Array (comma-separated)"
          value={formData.example_float_array}
          onChange={(e) => setFormData({ ...formData, example_float_array: e.target.value })}
          mb="md"
        />
        <Group>
          <Button type="submit">Save</Button>
          <Button variant="outline" onClick={() => navigate("/examples")}>
            Cancel
          </Button>
        </Group>
      </form>
    </div>
  );
}

export default ExampleEditPage;