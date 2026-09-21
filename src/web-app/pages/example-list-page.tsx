import { useState, useEffect, useRef, useMemo } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import {
  Table,
  Button,
  Title,
  Group,
  Pagination,
  Container,
  Loader,
  Alert,
} from "@mantine/core";
import { useExamplesStore } from "../stores/examples-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { confirmDelete } from "../utils/confirm-delete";
import { ROUTES } from "../constants/routes";
import { grants } from "../constants/grants";
import { reportError } from "../utils/report-error";

function ExampleListPage() {
  const authState = useAuthGuard([grants.READ_EXAMPLE]);
  const {
    itemList: exampleList,
    loading,
    error,
    readList,
    deleteItem,
  } = useExamplesStore();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [page, setPage] = useState(1);
  const itemsPerPage = 15;
  const hasFetched = useRef(false);

  // Sort state from URL params
  const sortColumn = searchParams.get("sort") || "exampleId";
  const sortDirection = (searchParams.get("dir") || "asc") as "asc" | "desc";

  useEffect(() => {
    if (!hasFetched.current && authState === "YES") {
      hasFetched.current = true;
      void readList({ incremental: true }).catch((err) => reportError(err, "example list"));
    }
  }, [authState, readList]);

  // Sort examples
  const sortedExampleList = useMemo(() => {
    // Skip sorting for very large lists to prevent stack overflow
    if (exampleList.length > 1000) {
      return exampleList;
    }
    return [...exampleList].sort((a, b) => {
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
  }, [exampleList, sortColumn, sortDirection]);

  //============================^ HOOK LINE ^====================

  if (authState === "CHECKING") {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }
  if (authState !== "YES") {
    return <Container size={1280} px={32} py={32} mx="auto"><Alert color="red">Authentication required.</Alert></Container>;
  }

  const totalPageCount = Math.ceil(sortedExampleList.length / itemsPerPage);
  const paginatedExampleList = sortedExampleList.slice(
    (page - 1) * itemsPerPage,
    page * itemsPerPage,
  );

  if (loading) return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;

  if (error) return <Container size={1280} px={32} py={32} mx="auto"><Alert color="red">Error loading examples: {error}</Alert></Container>;

  const handleDeleteClick = (id: number) => {
    confirmDelete("example", () => void handleDeleteConfirm(id));
  };

  const handleDeleteConfirm = async (id: number) => {
    const example = exampleList.find((ex) => ex.exampleId === id);
    if (example) {
      try {
        await deleteItem(example.exampleId, example.updateTs);
        // List is updated locally, but refresh to ensure sync
        await readList({ incremental: true });
      } catch (error) {
        reportError(error, "deleting example");
        // Error is handled in store
      }
    }
  };

  const handleSort = (column: string) => {
    const newDirection =
      sortColumn === column && sortDirection === "asc" ? "desc" : "asc";
    setSearchParams({ sort: column, dir: newDirection });
    setPage(1); // Reset to first page when sorting
  };

  return (
    <Container size={1280} px={32} py={32} mx="auto">
      <Group justify="space-between" mb="lg">
        <Title order={1}>Examples</Title>
        <Button
          onClick={() => void navigate(`${ROUTES.EXAMPLE_EDIT}?exampleId=new`)}
        >
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
              ID{" "}
              {sortColumn === "exampleId" &&
                (sortDirection === "asc" ? "↑" : "↓")}
            </Table.Th>
            <Table.Th
              style={{ cursor: "pointer" }}
              onClick={() => handleSort("exampleText")}
            >
              Text{" "}
              {sortColumn === "exampleText" &&
                (sortDirection === "asc" ? "↑" : "↓")}
            </Table.Th>
            <Table.Th
              style={{ cursor: "pointer" }}
              onClick={() => handleSort("exampleInt")}
            >
              Int{" "}
              {sortColumn === "exampleInt" &&
                (sortDirection === "asc" ? "↑" : "↓")}
            </Table.Th>
            <Table.Th
              style={{ cursor: "pointer" }}
              onClick={() => handleSort("exampleFloat")}
            >
              Float{" "}
              {sortColumn === "exampleFloat" &&
                (sortDirection === "asc" ? "↑" : "↓")}
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
                    void navigate(
                      `${ROUTES.EXAMPLE_EDIT}?exampleId=${example.exampleId}`,
                    )
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
    </Container>
  );
}

export default ExampleListPage;
