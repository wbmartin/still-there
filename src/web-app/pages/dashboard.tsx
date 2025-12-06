import { useEffect } from "react";
import { Title, Text, Card } from "@mantine/core";
import { useExamplesStore } from "../stores/examples-store";
import useUnauthRedirect from "../hooks/unauth-redirect";

function Dashboard() {
  useUnauthRedirect("/dashboard");
  const { examples, loading, fetchExamples } = useExamplesStore();

  useEffect(() => {
    if (examples.length === 0) {
      fetchExamples();
    }
  }, [examples.length, fetchExamples]);

  return (
    <div style={{ padding: "2rem" }}>
      <Title order={1} mb="lg">
        Dashboard
      </Title>
      <Card shadow="sm" p="lg">
        <Text size="lg">Total Examples: {loading ? "Loading..." : examples.length}</Text>
      </Card>
    </div>
  );
}

export default Dashboard;