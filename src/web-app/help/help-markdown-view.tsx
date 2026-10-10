import { Anchor, List, Stack, Text } from "@mantine/core";
import { Link } from "react-router-dom";
import { useHelpStore } from "../stores/help-store";
import {
  parseHelpMarkdown,
  type HelpBlock,
  type HelpInline,
} from "./help-markdown";

type HelpTextSize = "xs" | "sm" | "md";

// One run of inline content. Text is a React text node, so it is escaped:
// help content can never become markup.
function InlineList({ inlineList }: { inlineList: HelpInline[] }) {
  const openHelp = useHelpStore((state) => state.openHelp);
  return (
    <>
      {inlineList.map((inline, index) => {
        switch (inline.kind) {
          case "text":
            return <span key={index}>{inline.text}</span>;
          case "bold":
            return <strong key={index}>{inline.text}</strong>;
          case "help-link":
            return (
              <Anchor
                key={index}
                component="button"
                type="button"
                fz="inherit"
                onClick={() => openHelp(inline.id)}
              >
                {inline.label}
              </Anchor>
            );
          case "route-link":
            return (
              <Anchor key={index} component={Link} to={inline.to} fz="inherit">
                {inline.label}
              </Anchor>
            );
        }
      })}
    </>
  );
}

function Block({ block, size }: { block: HelpBlock; size: HelpTextSize }) {
  if (block.kind === "paragraph") {
    return (
      <Text size={size}>
        <InlineList inlineList={block.inlineList} />
      </Text>
    );
  }
  return (
    <List
      size={size}
      spacing={4}
      type={block.kind === "number-list" ? "ordered" : "unordered"}
    >
      {block.itemList.map((item, index) => (
        <List.Item key={index}>
          <InlineList inlineList={item} />
        </List.Item>
      ))}
    </List>
  );
}

/** A help body (help-markdown.ts's subset) as React elements, never HTML. */
export default function HelpMarkdownView({
  body,
  size = "sm",
}: {
  body: string;
  size?: HelpTextSize;
}) {
  return (
    <Stack gap="xs">
      {parseHelpMarkdown(body).map((block, index) => (
        <Block key={index} block={block} size={size} />
      ))}
    </Stack>
  );
}
