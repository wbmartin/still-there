import { useState } from "react";
import { Button, Group, Text, CopyButton, ActionIcon, Tooltip, Stack } from "@mantine/core";
import { IconCopy, IconCheck } from "@tabler/icons-react";
import { generateStrongPassword } from "../utils/password-utils";

interface PasswordGeneratorProps {
  onPasswordGenerated: (password: string) => void;
}

export function PasswordGenerator({
  onPasswordGenerated,
}: PasswordGeneratorProps) {
  const [generatedPassword, setGeneratedPassword] = useState("");

  const handleGenerate = () => {
    const password = generateStrongPassword();
    setGeneratedPassword(password);
    onPasswordGenerated(password);
  };

  return (
    <Stack>
      <Text>Password Generator Component</Text>
      <Group>
      <Button variant="filled" onClick={handleGenerate}>
        Generate Strong Password
      </Button>
      {generatedPassword && (
        <Group>
          <Text size="sm" c="dimmed">
            Generated: {generatedPassword}
          </Text>
          <CopyButton value={generatedPassword}>
            {({ copied, copy }) => (
              <Tooltip label={copied ? "Copied" : "Copy"}>
                <ActionIcon color={copied ? "teal" : "blue"} onClick={copy}>
                  {copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
                </ActionIcon>
              </Tooltip>
            )}
          </CopyButton>
        </Group>
      )}
    </Group>
    </Stack>
  );
}