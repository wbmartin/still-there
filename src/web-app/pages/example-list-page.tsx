import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Table, Button, Title, Group, Pagination } from "@mantine/core";
import { useExamplesStore } from "../stores/examples-store";
import { useExamplesSync } from "../hooks/use-examples-sync";
import useUnauthRedirect from "../hooks/unauth-redirect";

function ExampleListPage() {
  useUnauthRedirect("/exampleList");
  const { examples, loading, fetchExamples } = useExamplesStore();
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const itemsPerPage = 20;

  useExamplesSync(); // Start sync

  useEffect(() => {
    fetchExamples();
  }, [fetchExamples]);

  if (loading) return <div>Loading...</div>;

  const totalPages = Math.ceil(examples.length / itemsPerPage);
  const paginatedExamples = examples.slice((page - 1) * itemsPerPage, page * itemsPerPage);

  return (
    <div style={{ padding: "2rem" }}>
      <Group justify="space-between" mb="lg">
        <Title order={1}>Examples</Title>
        <Button onClick={() => navigate("/example?exampleId=new")}>Add Example</Button>
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
          {paginatedExamples.map((example) => (
            <Table.Tr key={example.exampleId}>
              <Table.Td>{example.exampleId}</Table.Td>
              <Table.Td>{example.exampleText}</Table.Td>
              <Table.Td>{example.exampleInt || ""}</Table.Td>
              <Table.Td>{example.exampleFloat || ""}</Table.Td>
              <Table.Td>
                <Button
                  size="xs"
                  onClick={() => navigate(`/example?exampleId=${example.exampleId}`)}
                >
                  Edit
                </Button>
              </Table.Td>
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
      {totalPages > 1 && (
        <Group justify="center" mt="lg">
          <Pagination total={totalPages} value={page} onChange={setPage} />
        </Group>
      )}
    </div>
  );
}

export default ExampleListPage;