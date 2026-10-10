import ReactMarkdown, { type Components } from "react-markdown";
import remarkBreaks from "remark-breaks";
import { Anchor, Box, Code, List, Text, Title } from "@mantine/core";

// Report markdown in Mantine typography (the issue-report module). Raw
// HTML is never rendered (skipHtml), and react-markdown drops unsafe URLs
// (javascript: and the like). An image becomes a link: shown inline it
// would fetch from any server (a tracking pixel); attachments are how a
// picture is shared.
const componentMap: Components = {
  h1: ({ children }) => (
    <Title order={4} mt="sm" mb={4}>
      {children}
    </Title>
  ),
  h2: ({ children }) => (
    <Title order={5} mt="sm" mb={4}>
      {children}
    </Title>
  ),
  h3: ({ children }) => (
    <Title order={6} mt="sm" mb={4}>
      {children}
    </Title>
  ),
  p: ({ children }) => (
    <Text size="sm" mb="xs" style={{ overflowWrap: "anywhere" }}>
      {children}
    </Text>
  ),
  a: ({ href, children }) => (
    <Anchor
      href={href}
      target="_blank"
      rel="noopener noreferrer nofollow"
      size="sm"
    >
      {children}
    </Anchor>
  ),
  img: ({ src, alt }) => (
    <Anchor
      href={typeof src === "string" ? src : undefined}
      target="_blank"
      rel="noopener noreferrer nofollow"
      size="sm"
    >
      {alt || "image"}
    </Anchor>
  ),
  ul: ({ children }) => (
    <List size="sm" mb="xs" withPadding>
      {children}
    </List>
  ),
  ol: ({ children }) => (
    <List type="ordered" size="sm" mb="xs" withPadding>
      {children}
    </List>
  ),
  li: ({ children }) => <List.Item>{children}</List.Item>,
  code: ({ children, className }) =>
    className ? (
      <Code block mb="xs">
        {children}
      </Code>
    ) : (
      <Code>{children}</Code>
    ),
  pre: ({ children }) => <>{children}</>,
  blockquote: ({ children }) => (
    <Box
      pl="sm"
      mb="xs"
      style={{ borderLeft: "3px solid var(--mantine-color-default-border)" }}
    >
      {children}
    </Box>
  ),
};

export function MarkdownView({ text }: { text: string }) {
  return (
    <Box>
      <ReactMarkdown
        remarkPlugins={[remarkBreaks]}
        components={componentMap}
        skipHtml
      >
        {text}
      </ReactMarkdown>
    </Box>
  );
}
