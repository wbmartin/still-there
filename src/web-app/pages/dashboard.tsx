import { useEffect, useRef } from "react";
import { Title, Text, Card, Container, Loader, Alert } from "@mantine/core";
import { useExamplesStore } from "../stores/examples-store";
import useAuthGuard from "../hooks/use-auth-guard";
import { reportError } from "../utils/report-error";

function Dashboard() {
  const authState = useAuthGuard([]);
  const { itemList: exampleList, loading, readList } = useExamplesStore();
  const hasFetched = useRef(false);

  useEffect(() => {
    if (!hasFetched.current && authState === "YES") {
      hasFetched.current = true;
      void readList({ incremental: true }).catch((err) => reportError(err, "dashboard"));
    }
  }, [authState, readList]);

  //============================^ HOOK LINE ^====================
  if (authState === "CHECKING") {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }
  if (authState !== "YES") {
    return <Container size={1280} px={32} py={32} mx="auto"><Alert color="red">Authentication required.</Alert></Container>;
  }
  if (loading) {
    return <Container size={1280} px={32} py={32} mx="auto" ta="center"><Loader /></Container>;
  }

  return (
    <Container size={1280} px={32} py={32} mx="auto">
      <Title order={1} mb="lg">
        Dashboard
      </Title>
      <Card shadow="sm" p="lg">
        <Text size="lg">Total Examples: {exampleList.length}</Text>
      </Card>
    </Container>
  );
}

export default Dashboard;
