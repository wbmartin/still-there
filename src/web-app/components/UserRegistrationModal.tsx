import { useState } from "react";
import { Modal, TextInput, PasswordInput, Button, Group, Text } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { apiRequest } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";
import { PasswordGenerator } from "./PasswordGenerator";

interface UserRegistrationModalProps {
  opened: boolean;
  onClose: () => void;
  onSuccess?: () => void;
}

export function UserRegistrationModal({ opened, onClose, onSuccess }: UserRegistrationModalProps) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");



  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (password !== confirmPassword) {
      setError("Passwords do not match");
      return;
    }

    if (password.length < 8) {
      setError("Password must be at least 8 characters long");
      return;
    }

    setLoading(true);
    try {
      const response = await apiRequest(API_ENDPOINTS.USER_SELF_CREATE, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      });

      if (!response.ok) {
        const errorData = await response.json();
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
    setUsername("");
    setPassword("");
    setConfirmPassword("");
    setError("");
  };

  const handleClose = () => {
    resetForm();
    onClose();
  };

  return (
    <Modal opened={opened} onClose={handleClose} title="Create Account" size="md">
      <form onSubmit={(e) => void handleSubmit(e)}>
        <TextInput
          label="Username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          required
          mb="md"
          placeholder="Enter your username"
        />

        <PasswordInput
          label="Password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          mb="md"
          placeholder="Enter your password"
        />

        <PasswordInput
          label="Confirm Password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          required
          mb="md"
          placeholder="Confirm your password"
        />

            <PasswordGenerator
              onPasswordGenerated={(pwd) => {
                setPassword(pwd);
                setConfirmPassword(pwd);
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