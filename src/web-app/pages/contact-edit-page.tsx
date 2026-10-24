import { useState, useEffect, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useForm } from "@mantine/form";
import { zod4Resolver } from "mantine-form-zod-resolver";
import {
  Alert,
  Button,
  Checkbox,
  Container,
  Group,
  Loader,
  Stack,
  Text,
  TextInput,
  Title,
} from "@mantine/core";
import { ROUTES } from "../constants/routes";
import { grants } from "../constants/grants";
import { useContactStore } from "../stores/contact-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { notifyFailure, notifySuccess } from "../utils/notify";
import { isE164, normalisePhone } from "../utils/phone";
import { contactSchema, type ContactFormData } from "../schemas/form-schemas";
import {
  CONTACT_NOTIFY_EVENT_LIST,
  type Contact,
} from "../types/contact-type";
import { ContactStatusBadge } from "../components/contact-status-badge";

// A new contact is texted about everything until someone narrows it.
const DEFAULT_CONTACT_FORM: ContactFormData = {
  contactName: "",
  phone: "",
  notifyEventList: CONTACT_NOTIFY_EVENT_LIST.map((event) => event.value),
};

// A person texted about a device: `?contactId=N` edits one, `?deviceId=N`
// adds one to that device. Either way the page returns to the device.
function ContactEditPage() {
  const [searchParams] = useSearchParams();
  const id = searchParams.get("contactId") || "";
  const isNew = id === "";
  const newDeviceId = Number(searchParams.get("deviceId")) || null;
  const requiredGrant = isNew ? grants.CREATE_CONTACT : grants.UPDATE_CONTACT;
  const authState = useAuthGuard([requiredGrant]);
  const { readContactById, createItem, updateItem } = useContactStore();
  const navigate = useNavigate();

  const numericId = isNaN(Number(id)) ? null : Number(id);

  const form = useForm<ContactFormData>({
    initialValues: DEFAULT_CONTACT_FORM,
    validate: zod4Resolver(contactSchema),
  });
  // The route id the form was last loaded for, and what that load found
  // (example-edit-page.tsx): the row, or null.
  const requestedId = useRef<string | null>(null);
  const [loadResult, setLoadResult] = useState<{
    id: string;
    contact: Contact | null;
  } | null>(null);

  useEffect(() => {
    if (authState !== "YES" || requestedId.current === id) return;
    requestedId.current = id;
    if (isNew || !numericId) {
      form.reset();
      return;
    }
    void readContactById(numericId)
      .then((existingContact) => {
        if (requestedId.current !== id) return; // a later id took over
        if (existingContact) {
          form.setValues({
            contactName: existingContact.contactName,
            phone: existingContact.phoneE164,
            notifyEventList: existingContact.notifyEventList,
          });
        }
        setLoadResult({ id, contact: existingContact });
      })
      .catch((error: unknown) => {
        if (requestedId.current !== id) return;
        notifyFailure(error, "loading contact", "Failed to load contact");
        setLoadResult({ id, contact: null });
      });
    // `form` is a new object every render; requestedId makes this run once
    // per route id.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [authState, id, isNew, numericId, readContactById]);

  //============================^ HOOK LINE ^====================
  const contact =
    !isNew && loadResult?.id === id ? loadResult.contact : null;
  if (authState === "CHECKING") {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }
  if (authState !== "YES") {
    return <Container size={1280} px={32} py={32} mx="auto"><Alert color="red">Authentication required.</Alert></Container>;
  }

  if (!isNew && numericId && loadResult?.id !== id) {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }
  // Never render the form for a record that was not loaded, nor a new
  // contact with no device to add them to.
  const deviceId = contact?.deviceId ?? newDeviceId;
  if (
    (!isNew && (!numericId || !contact)) ||
    deviceId === null
  ) {
    return (
      <Container size={1280} px={32} py={32} mx="auto">
        <Title order={1} mb="lg">{isNew ? "Add Contact" : "Edit Contact"}</Title>
        <Alert color="red">This contact could not be loaded.</Alert>
      </Container>
    );
  }
  const devicePath = `${ROUTES.DEVICE_EDIT}?deviceId=${deviceId}`;
  const normalisedPhone = normalisePhone(form.values.phone);

  const handleSubmit = async (values: ContactFormData) => {
    const data = {
      contactName: values.contactName.trim(),
      phoneE164: normalisePhone(values.phone),
      notifyEventList: values.notifyEventList,
    };
    try {
      if (contact) {
        await updateItem({
          contactId: contact.contactId,
          deviceId: contact.deviceId,
          ...data,
          updateTs: contact.updateTs,
        });
        notifySuccess("Contact updated successfully");
      } else {
        await createItem({ deviceId, ...data });
        notifySuccess("Contact created: an invite text is on its way");
      }
      void navigate(devicePath);
    } catch (error) {
      // A contact who has replied keeps their number: the API's 409 says
      // so, and notifyFailure shows its message.
      notifyFailure(error, "saving contact", "Failed to save contact");
    }
  };

  return (
    <Container size={1280} px={32} py={32} mx="auto">
      <Title order={1} mb="lg">
        {isNew ? "Add Contact" : "Edit Contact"}
      </Title>
      {contact && (
        <Group mb="md" gap="xs">
          <Text size="sm">Status:</Text>
          <ContactStatusBadge status={contact.contactStatus} />
        </Group>
      )}
      {contact?.contactStatus === "opted_out" && (
        <Alert color="red" variant="light" mb="md">
          They texted STOP. Only their texting START re-enables alerts.
        </Alert>
      )}
      <form onSubmit={form.onSubmit((values) => void handleSubmit(values))}>
        <TextInput
          label="Name"
          {...form.getInputProps("contactName")}
          mb="md"
        />
        <TextInput
          label="Phone"
          placeholder="704.860.9088 or +447700900123"
          description={
            contact && contact.contactStatus !== "invited"
              ? "They have replied, so this number cannot change; add a new contact for a new number."
              : isE164(normalisedPhone)
                ? `Saved as ${normalisedPhone}`
                : "10 digits for the US, or + and the country code"
          }
          {...form.getInputProps("phone")}
          mb="md"
        />
        <Checkbox.Group
          label="Text them when"
          {...form.getInputProps("notifyEventList")}
          mb="md"
        >
          <Stack gap="xs" mt="xs">
            {CONTACT_NOTIFY_EVENT_LIST.map((event) => (
              <Checkbox
                key={event.value}
                value={event.value}
                label={event.label}
              />
            ))}
          </Stack>
        </Checkbox.Group>
        <Group>
          <Button type="submit">Save</Button>
          <Button variant="outline" onClick={() => void navigate(devicePath)}>
            Cancel
          </Button>
        </Group>
      </form>
    </Container>
  );
}

export default ContactEditPage;
