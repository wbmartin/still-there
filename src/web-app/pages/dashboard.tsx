import { useEffect, useRef } from "react";
import { Title, Text, Card } from "@mantine/core";
import { useExamplesStore } from "../stores/examples-store";
import useAuthGuard from "../hooks/use-auth-guard";

function Dashboard() {
  const authState = useAuthGuard([]);
  const { exampleList, loading, retrieveExampleList } = useExamplesStore();
  const hasFetched = useRef(false);

  useEffect(() => {
    if (!hasFetched.current && authState === "YES") {
      hasFetched.current = true;
       void retrieveExampleList();
    }
  }, [authState, retrieveExampleList]);

  //============================^HOOOK LINE ^====================
  if (authState !== "YES") {
    return <div>Checking Auth...</div>;
  }
  if (loading) {
    return <div>Loading...</div>;
  }

  return (
    <div style={{ padding: "2rem" }}>
      <Title order={1} mb="lg">
        Dashboard
      </Title>
      <Card shadow="sm" p="lg">
        <Text size="lg">Total Examples: {exampleList.length}</Text>
      </Card>
    </div>
  );
}

export default Dashboard;
