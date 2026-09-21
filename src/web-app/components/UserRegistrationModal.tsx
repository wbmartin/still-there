import { useState } from "react";
import { Modal, TextInput, PasswordInput, Button, Group, Text } from "@mantine/core";
import { useForm } from "@mantine/form";
import { notifications } from "@mantine/notifications";
import { apiRequest } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";
import { PasswordGenerator } from "./PasswordGenerator";

interface UserRegistrationModalProps {
  opened: boolean;
  onClose: () => void;
  onSuccess?: () => void;
}

export function UserRegistrationModal({
  opened,
  onClose,
  onSuccess,
}: UserRegistrationModalProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const form = useForm({
    initialValues: {
      username: '',
      password: '',
      confirmPassword: '',
    },
    validate: {
      username: (value) => (value.length < 1 ? 'Username is required' : null),
      password: (value) => (value.length < 8 ? 'Password must be at least 8 characters' : null),
      confirmPassword: (value, values) =>
        value !== values.password ? 'Passwords do not match' : null,
    },
  });



  const handleSubmit = async (values: {
    username: string;
    password: string;
    confirmPassword: string;
  }) => {
    setError("");
    setLoading(true);

    try {
      const response = await apiRequest(API_ENDPOINTS.USER_SELF_CREATE, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          username: values.username,
          password: values.password,
        }),
      });

       if (!response.ok) {
         const errorData = (await response.json()) as { message: string };
         throw new Error(errorData.message || "Registration failed");
       }

      notifications.show({
        title: "Registration Successful",
        message: "Login with your new password to setup MFA.",
        color: "green",
      });

      onClose();
      onSuccess?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  };

  const resetForm = () => {
    form.reset();
    setError("");
  };

  const handleClose = () => {
    resetForm();
    onClose();
  };

  return (
    <Modal opened={opened} onClose={handleClose} title="Create Account" size="md">
      <form onSubmit={form.onSubmit(handleSubmit)}>
        <TextInput
          label="Username"
          {...form.getInputProps('username')}
          mb="md"
          placeholder="Enter your username"
        />

        <PasswordInput
          label="Password"
          {...form.getInputProps('password')}
          mb="md"
          placeholder="Enter your password"
        />

        <PasswordInput
          label="Confirm Password"
          {...form.getInputProps('confirmPassword')}
          mb="md"
          placeholder="Confirm your password"
        />

            <PasswordGenerator
              onPasswordGenerated={(pwd) => {
                form.setFieldValue('password', pwd);
                form.setFieldValue('confirmPassword', pwd);
              }}
            />

        {error && (
          <Text c="red" size="sm" mb="md">
            {error}
          </Text>
        )}

        <Group justify="flex-end">
          <Button variant="default" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" loading={loading}>
            Create Account
          </Button>
        </Group>
      </form>
    </Modal>
  );
}