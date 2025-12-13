import { useState, useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { Table, Button, Title, Group, Pagination } from "@mantine/core";
import { useExamplesStore } from "../stores/examples-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { DeleteConfirmModal } from "../components/DeleteConfirmModal";
import { ROUTES } from "../constants/routes";

function ExampleListPage() {
  const authState = useAuthGuard();
  const {
    examples,
    loading,
    readExampleList,
    deleteExample,
  } = useExamplesStore();
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const itemsPerPage = 20;
  const hasFetched = useRef(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deleteId, setDeleteId] = useState<number | null>(null);

   useEffect(() => {
     if (!hasFetched.current && authState === "YES") {
       hasFetched.current = true;
       void readExampleList();
     }
   }, [authState, readExampleList]);

  //============================^HOOOK LINE ^====================

  if (authState !== "YES") {
    return <div>Loading...</div>;
  }

  if (loading) return <div>Loading...</div>;

  const totalPages = Math.ceil(examples.length / itemsPerPage);
  const paginatedExamples = examples.slice(
    (page - 1) * itemsPerPage,
    page * itemsPerPage,
  );

  const handleDeleteClick = (id: number) => {
    setDeleteId(id);
    setDeleteModalOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (deleteId !== null) {
      try {
        await deleteExample(deleteId);
        // List is updated locally, but refresh to ensure sync
        await       readExampleList();
      } catch (error) {
        console.error("Error deleting example:", error);
        // Error is handled in store
      }
    }
    setDeleteModalOpen(false);
    setDeleteId(null);
  };

  const handleDeleteCancel = () => {
    setDeleteModalOpen(false);
    setDeleteId(null);
  };

  return (
    <div style={{ padding: "2rem" }}>
      <Group justify="space-between" mb="lg">
        <Title order={1}>Examples</Title>
        <Button onClick={() => navigate(`/${ROUTES.EXAMPLE_EDIT}?exampleId=new`)}>
          Add Example
        </Button>
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
                   onClick={() =>
                      navigate(`/${ROUTES.EXAMPLE_EDIT}?exampleId=${example.exampleId}`)
                   }
                 >
                   Edit
                 </Button>
                 <Button
                   size="xs"
                   color="red"
                   ml="xs"
                   onClick={() => handleDeleteClick(example.exampleId)}
                 >
                   Delete
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
      <DeleteConfirmModal
        isOpen={deleteModalOpen}
        onConfirm={() => void handleDeleteConfirm()}
        onCancel={handleDeleteCancel}
        entityType="example"
      />
    </div>
  );
}

export default ExampleListPage;
