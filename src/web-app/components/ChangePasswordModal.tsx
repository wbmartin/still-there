import { Modal, PasswordInput, Button, Group, Alert } from "@mantine/core";
import { useState } from "react";
import { PasswordGenerator } from "./PasswordGenerator";

interface ChangePasswordModalProps {
  isOpen: boolean;
  onConfirm: (oldPassword: string, newPassword: string) => void;
  onCancel: () => void;
  mode: "own" | "other"; // "own" for changing own password, "other" for changing another user's password
  loading?: boolean;
  error?: string;
}

export function ChangePasswordModal({
  isOpen,
  onConfirm,
  onCancel,
  mode,
  loading = false,
  error,
}: ChangePasswordModalProps) {
  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [validationError, setValidationError] = useState("");

  const handleConfirmClick = () => {
    // Reset validation error
    setValidationError("");

    // Validate passwords
    if (mode === "own" && !oldPassword.trim()) {
      setValidationError("Old password is required");
      return;
    }

    if (!newPassword.trim()) {
      setValidationError("New password is required");
      return;
    }

    if (newPassword.length < 8) {
      setValidationError("New password must be at least 8 characters long");
      return;
    }

    if (newPassword !== confirmPassword) {
      setValidationError("New passwords do not match");
      return;
    }
    onConfirm(oldPassword, newPassword);

    // Reset form
    setOldPassword("");
    setNewPassword("");
    setConfirmPassword("");
    setValidationError("");
  };

  const handleCancel = () => {
    // Reset form
    setOldPassword("");
    setNewPassword("");
    setConfirmPassword("");
    setValidationError("");
    onCancel();
  };

  return (
    <Modal
      opened={isOpen}
      onClose={handleCancel}
      title="Change Password"
      centered
      size="lg"
    >
      <PasswordInput
        label={mode === "own" ? "Current Password" : "New Password"}
        value={mode === "own" ? oldPassword : newPassword}
        onChange={(e) => {
          if (mode === "own") {
            setOldPassword(e.target.value);
          } else {
            setNewPassword(e.target.value);
          }
        }}
        required
        mb="md"
        disabled={loading}
      />

      {mode === "own" && (
        <PasswordInput
          label="New Password"
          value={newPassword}
          onChange={(e) => setNewPassword(e.target.value)}
          required
          mb="md"
        />
      )}

      <PasswordInput
        label="Confirm New Password"
        value={confirmPassword}
        onChange={(e) => setConfirmPassword(e.target.value)}
        required
        mb="md"
        disabled={loading}
      />

      {validationError && (
        <Alert color="red" mb="md">
          {validationError}
        </Alert>
      )}

      {error && (
        <Alert color="red" mb="md">
          {error}
        </Alert>
      )}

      <PasswordGenerator
        onPasswordGenerated={(password) => {
          setNewPassword(password);
          setConfirmPassword(password);
        }}
      />

      <Group justify="center" mt="md" grow>
        <Button variant="outline" onClick={handleCancel} disabled={loading}>
          Cancel
        </Button>
        <Button onClick={handleConfirmClick} loading={loading}>
          Change Password
        </Button>
      </Group>
    </Modal>
  );
}
