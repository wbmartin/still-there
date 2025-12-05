import { Title, Text, Card } from "@mantine/core";
import { useGetExamplesQuery } from "../services/example-svc";
import useUnauthRedirect from "../hooks/unauth-redirect";

function Dashboard() {
  useUnauthRedirect("/dashboard");
  const { data, isLoading } = useGetExamplesQuery();

  return (
    <div style={{ padding: "2rem" }}>
      <Title order={1} mb="lg">
        Dashboard
      </Title>
      <Card shadow="sm" p="lg">
        <Text size="lg">Total Examples: {isLoading ? "Loading..." : data?.examples?.length || 0}</Text>
      </Card>
    </div>
  );
}

export default Dashboard;