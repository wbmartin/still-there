import { useState, useEffect, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useForm } from "@mantine/form";
import {
  TextInput,
  NumberInput,
  Textarea,
  Button,
  Title,
  Group,
  Container,
  Loader,
  Alert,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { ROUTES } from "../constants/routes";
import { useExamplesStore } from "../stores/examples-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { confirmDelete } from "../utils/confirm-delete";
import { grants } from "../constants/grants";
import { defaultExampleFormData } from "../types/example-type";
import { LabelWithHelp } from "../components/help-tip";
import { reportError } from "../utils/report-error";

const isJsonOrEmpty = (text: string) => {
  if (!text.trim()) return true;
  try {
    JSON.parse(text);
    return true;
  } catch {
    return false;
  }
};

function ExampleEditPage() {
  const [searchParams] = useSearchParams();
  const id = searchParams.get("exampleId") || "";
  const isNew = id === "new";
  const requiredGrant = isNew ? grants.CREATE_EXAMPLE : grants.UPDATE_EXAMPLE;
  const authState = useAuthGuard([requiredGrant]);
  const { readExampleById, createItem, updateItem, deleteItem } =
    useExamplesStore();
  const navigate = useNavigate();

  const numericId = isNaN(Number(id)) ? null : Number(id);

  const form = useForm<typeof defaultExampleFormData>({
    initialValues: defaultExampleFormData,
    validate: {
      exampleText: (value) => (value.length < 1 ? 'Example text is required' : null),
      exampleJsonb: (value) => (isJsonOrEmpty(value) ? null : 'Must be valid JSON (or empty)'),
    },
  });
  const [exampleUuid, setExampleUuid] = useState("");
  const [exampleLoading, setExampleLoading] = useState(
    !isNew && numericId !== null,
  );
  const hasFetched = useRef(false);

  useEffect(() => {
    if (!hasFetched.current && authState === "YES" && numericId && !isNew) {
      hasFetched.current = true;
      void readExampleById(numericId)
        .then((existingExample) => {
          if (existingExample) {
            form.setValues({
              exampleText: existingExample.exampleText,
              exampleInt: existingExample.exampleInt?.toString() || "",
              exampleFloat: existingExample.exampleFloat?.toString() || "",
              exampleIntArray:
                existingExample.exampleIntArray?.join(", ") || "",
              exampleFloatArray:
                existingExample.exampleFloatArray?.join(", ") || "",
              exampleJsonb:
                existingExample.exampleJsonb === undefined
                  ? ""
                  : JSON.stringify(existingExample.exampleJsonb, null, 2),
              updateTs: existingExample.updateTs.toString(),
            });
            setExampleUuid(existingExample.exampleUuid ?? "");
          }
        })
        .catch((error: unknown) => {
          reportError(error, "loading example");
        })
        .finally(() => {
          setExampleLoading(false);
        });
    }
    // `form` is a new object every render; hasFetched makes this a one-shot.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [authState, numericId, isNew, readExampleById]);

  //============================^ HOOK LINE ^====================
  if (authState === "CHECKING") {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }
  if (authState !== "YES") {
    return <Container size={1280} px={32} py={32} mx="auto"><Alert color="red">Authentication required.</Alert></Container>;
  }

  if (exampleLoading) {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }

  const handleSubmit = async (values: typeof defaultExampleFormData) => {
    const data = {
      exampleText: values.exampleText,
      exampleInt: values.exampleInt ? Number(values.exampleInt) : undefined,
      exampleFloat: values.exampleFloat
        ? Number(values.exampleFloat)
        : undefined,
      exampleIntArray: values.exampleIntArray
        ? values.exampleIntArray.split(",").map(Number)
        : undefined,
      exampleFloatArray: values.exampleFloatArray
        ? values.exampleFloatArray.split(",").map(Number)
        : undefined,
      // Validated as JSON by the form; empty means no value.
      exampleJsonb: values.exampleJsonb.trim()
        ? (JSON.parse(values.exampleJsonb) as unknown)
        : undefined,
      updateTs: !isNew ? Number(values.updateTs) : undefined,
    };

    try {
      if (isNew) {
        await createItem(data);
        notifications.show({
          title: "Success",
          message: "Example created successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
      } else {
        await updateItem({ exampleId: Number(id), ...data });
        notifications.show({
          title: "Success",
          message: "Example updated successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
      }
      void navigate(ROUTES.EXAMPLE_LIST);
    } catch (error) {
      reportError(error, "saving example");
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
    confirmDelete("example", () => void handleDeleteConfirm());
  };

  const handleDeleteConfirm = async () => {
    if (numericId) {
      try {
        await deleteItem(numericId, Number(form.values.updateTs));
        notifications.show({
          title: "Success",
          message: "Example deleted successfully",
          color: "green",
          autoClose: 3000,
          withCloseButton: false,
        });
      void navigate(ROUTES.EXAMPLE_LIST);
      } catch (error) {
        reportError(error, "deleting example");
        notifications.show({
          title: "Error",
          message: "Failed to delete example",
          color: "red",
          autoClose: 5000,
          withCloseButton: true,
        });
      }
    }
  };

  return (
    <Container size={1280} px={32} py={32} mx="auto">
      <Title order={1} mb="lg">
        {isNew ? "Add Example" : "Edit Example"}
      </Title>
      <form onSubmit={form.onSubmit(handleSubmit)}>
        <TextInput
          label="Text"
          {...form.getInputProps('exampleText')}
          mb="md"
        />
        <NumberInput
          label="Int"
          {...form.getInputProps('exampleInt')}
          mb="md"
        />
        <NumberInput
          label="Float"
          {...form.getInputProps('exampleFloat')}
          mb="md"
        />
        <Textarea
          label="Int Array (comma-separated)"
          {...form.getInputProps('exampleIntArray')}
          mb="md"
        />
        <Textarea
          label="Float Array (comma-separated)"
          {...form.getInputProps('exampleFloatArray')}
          mb="md"
        />
        <Textarea
          label={
            <LabelWithHelp
              label="JSON"
              help="Any JSON value; leave empty for none"
            />
          }
          autosize
          minRows={3}
          ff="monospace"
          {...form.getInputProps('exampleJsonb')}
          mb="md"
        />
        {!isNew && (
          <TextInput
            label={
              <LabelWithHelp
                label="UUID"
                help="Assigned by the server on create; read-only"
              />
            }
            value={exampleUuid}
            readOnly
            ff="monospace"
            mb="md"
          />
        )}
        <Group>
          <Button type="submit">Save</Button>
          {!isNew && (
            <Button color="red" onClick={handleDeleteClick}>
              Delete
            </Button>
          )}
          <Button
            variant="outline"
            onClick={() => void navigate(ROUTES.EXAMPLE_LIST)}
          >
            Cancel
          </Button>
        </Group>
      </form>
    </Container>
  );
}

export default ExampleEditPage;
