import { useState } from "react";
import {
  Box,
  Group,
  Paper,
  SegmentedControl,
  Text,
  Textarea,
} from "@mantine/core";
import { MarkdownView } from "./markdown-view";

/** A textarea with Write / Preview tabs, for a report or a reply. */
export function MarkdownEditor({
  label,
  value,
  onChange,
  placeholder,
  minRows = 5,
  maxLength,
}: {
  label?: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  minRows?: number;
  maxLength?: number;
}) {
  const [mode, setMode] = useState<"write" | "preview">("write");
  return (
    <Box>
      <Group justify="space-between" mb={4}>
        {label ? (
          <Text size="sm" fw={500}>
            {label}
          </Text>
        ) : (
          <span />
        )}
        <SegmentedControl
          size="xs"
          value={mode}
          onChange={(next) => setMode(next as "write" | "preview")}
          data={[
            { value: "write", label: "Write" },
            { value: "preview", label: "Preview" },
          ]}
        />
      </Group>
      {mode === "write" ? (
        <Textarea
          aria-label={label}
          value={value}
          onChange={(event) => onChange(event.currentTarget.value)}
          placeholder={placeholder}
          autosize
          minRows={minRows}
          maxRows={16}
          maxLength={maxLength}
          description="Markdown works: **bold**, - lists, `code`, [links](https://…)"
        />
      ) : (
        <Paper withBorder p="sm" mih={minRows * 22}>
          {value.trim() ? (
            <MarkdownView text={value} />
          ) : (
            <Text size="sm" c="dimmed">
              Nothing to preview yet.
            </Text>
          )}
        </Paper>
      )}
    </Box>
  );
}
