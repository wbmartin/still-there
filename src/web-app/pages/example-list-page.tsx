import { useNavigate } from "react-router-dom";
import { Table, Button, Title, Group } from "@mantine/core";
import { useGetExamplesQuery } from "../services/example-svc";
import useUnauthRedirect from "../hooks/unauth-redirect";

function ExampleListPage() {
  useUnauthRedirect("/exampleList");
  const { data, isLoading } = useGetExamplesQuery();
  const navigate = useNavigate();

  if (isLoading) return <div>Loading...</div>;

  const examples = data?.examples || [];

  return (
    <div style={{ padding: "2rem" }}>
      <Group justify="space-between" mb="lg">
        <Title order={1}>Examples</Title>
        <Button onClick={() => navigate("/example/new")}>Add Example</Button>
      </Group>
      <Table>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>ID</Table.Th>
            <Table.Th>Text</Table.Th>
            <Table.Th>Int</Table.Th>
            <Table.Th>Float</Table.Th>
            <Table.Th>Actions</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {examples.map((example) => (
            <Table.Tr key={example.exampleId}>
              <Table.Td>{example.exampleId}</Table.Td>
              <Table.Td>{example.exampleText}</Table.Td>
              <Table.Td>{example.exampleInt || ""}</Table.Td>
              <Table.Td>{example.exampleFloat || ""}</Table.Td>
              <Table.Td>
                <Button
                  size="xs"
                  onClick={() => navigate(`/example/${example.exampleId}`)}
                >
                  Edit
                </Button>
              </Table.Td>
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
    </div>
  );
}

export default ExampleListPage;