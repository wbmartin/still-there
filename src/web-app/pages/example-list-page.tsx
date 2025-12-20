import { useState, useEffect, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Table, Button, Title, Group, Pagination } from "@mantine/core";
import { useExamplesStore } from "../stores/examples-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { DeleteConfirmModal } from "../components/DeleteConfirmModal";
import { ROUTES } from "../constants/routes";
import { grants } from "../constants/grants";

function ExampleListPage() {
  const authState = useAuthGuard([grants.READ_EXAMPLE]);
  const {
    exampleList,
    loading,
    retrieveExampleList,
    deleteExample,
  } = useExamplesStore();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [page, setPage] = useState(1);
  const itemsPerPage = 15;
  const hasFetched = useRef(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deleteId, setDeleteId] = useState<number | null>(null);

  // Sort state from URL params
  const sortColumn = searchParams.get("sort") || "exampleId";
  const sortDirection = (searchParams.get("dir") || "asc") as "asc" | "desc";

   useEffect(() => {
     if (!hasFetched.current && authState === "YES") {
       hasFetched.current = true;
       void retrieveExampleList();
     }
   }, [authState, retrieveExampleList]);

  //============================^HOOOK LINE ^====================

  if (authState === "CHECKING") {
    return <div>Checking permissions...</div>;
  }
  if (authState !== "YES") {
    return <div>Loading...</div>;
  }

  if (loading) return <div>Loading...</div>;

  // Sort examples
  const sortedExampleList = [...exampleList].sort((a, b) => {
    let aValue: string | number;
    let bValue: string | number;

    switch (sortColumn) {
      case "exampleId":
        aValue = a.exampleId;
        bValue = b.exampleId;
        break;
      case "exampleText":
        aValue = a.exampleText.toLowerCase();
        bValue = b.exampleText.toLowerCase();
        break;
      case "exampleInt":
        aValue = a.exampleInt || 0;
        bValue = b.exampleInt || 0;
        break;
      case "exampleFloat":
        aValue = a.exampleFloat || 0;
        bValue = b.exampleFloat || 0;
        break;
      default:
        return 0;
    }

    if (aValue < bValue) return sortDirection === "asc" ? -1 : 1;
    if (aValue > bValue) return sortDirection === "asc" ? 1 : -1;
    return 0;
  });

  const totalPageCount = Math.ceil(sortedExampleList.length / itemsPerPage);
  const paginatedExampleList = sortedExampleList.slice(
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
        await retrieveExampleList();
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

  const handleSort = (column: string) => {
    const newDirection = sortColumn === column && sortDirection === "asc" ? "desc" : "asc";
    setSearchParams({ sort: column, dir: newDirection });
    setPage(1); // Reset to first page when sorting
  };

  return (
    <div style={{ padding: "2rem" }}>
      <Group justify="space-between" mb="lg">
        <Title order={1}>Examples</Title>
        <Button onClick={() => void navigate(`/${ROUTES.EXAMPLE_EDIT}?exampleId=new`)}>
          Add Example
        </Button>
      </Group>
      <Table>
        <Table.Thead>
          <Table.Tr>
            <Table.Th
              style={{ cursor: "pointer" }}
              onClick={() => handleSort("exampleId")}
            >
              ID {sortColumn === "exampleId" && (sortDirection === "asc" ? "↑" : "↓")}
            </Table.Th>
            <Table.Th
              style={{ cursor: "pointer" }}
              onClick={() => handleSort("exampleText")}
            >
              Text {sortColumn === "exampleText" && (sortDirection === "asc" ? "↑" : "↓")}
            </Table.Th>
            <Table.Th
              style={{ cursor: "pointer" }}
              onClick={() => handleSort("exampleInt")}
            >
              Int {sortColumn === "exampleInt" && (sortDirection === "asc" ? "↑" : "↓")}
            </Table.Th>
            <Table.Th
              style={{ cursor: "pointer" }}
              onClick={() => handleSort("exampleFloat")}
            >
              Float {sortColumn === "exampleFloat" && (sortDirection === "asc" ? "↑" : "↓")}
            </Table.Th>
            <Table.Th>Actions</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {paginatedExampleList.map((example) => (
            <Table.Tr key={example.exampleId}>
              <Table.Td>{example.exampleId}</Table.Td>
              <Table.Td>{example.exampleText}</Table.Td>
              <Table.Td>{example.exampleInt || ""}</Table.Td>
              <Table.Td>{example.exampleFloat || ""}</Table.Td>
               <Table.Td>
                 <Button
                   size="xs"
                    onClick={() =>
                       void navigate(`/${ROUTES.EXAMPLE_EDIT}?exampleId=${example.exampleId}`)
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
      {totalPageCount > 1 && (
        <Group justify="center" mt="lg">
          <Pagination total={totalPageCount} value={page} onChange={setPage} />
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
