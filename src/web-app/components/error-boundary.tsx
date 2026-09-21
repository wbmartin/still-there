import { Component } from 'react';
import type { ErrorInfo, ReactNode } from 'react';
import {
  Alert,
  Button,
  Group,
  Stack,
  Text,
  Title,
  Container,
} from '@mantine/core';
import { reportError } from '../utils/report-error';

// A thrown render error should cost one view, not the whole application.
//
// React's default for an uncaught render error is to unmount the whole tree.
// This boundary turns that into a message with a way back. A class component
// because `getDerivedStateFromError` and `componentDidCatch` have no hook
// equivalent.

interface ErrorBoundaryProps {
  children: ReactNode;
  /** Names the view in the fallback and the console log. */
  viewName?: string;
}

interface ErrorBoundaryState {
  error: Error | null;
}

class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    // The component stack is what makes a render error diagnosable, and React
    // only hands it to this method. Logged rather than POSTed to the server:
    // an unauthenticated error endpoint is a spam target.
    reportError(
      error,
      `${this.props.viewName ?? "view"} failed to render: ${errorInfo.componentStack ?? ""}`,
    );
  }

  private reset = () => {
    this.setState({ error: null });
  };

  render() {
    const { error } = this.state;
    if (!error) return this.props.children;

    return (
      <Container size={1280} px={32} py={32} mx="auto" maw="600px">
        <Stack gap="lg">
          <Title order={1} ta="center">
            This page ran into a problem
          </Title>

          <Alert color="red" variant="light" role="alert">
            <Text>
              Something on this screen failed to load. Nothing you have
              already saved is affected.
            </Text>
          </Alert>

          <Group justify="center" gap="md">
            <Button onClick={this.reset}>Try again</Button>
            <Button variant="outline" component="a" href="/dashboard">
              Back to dashboard
            </Button>
          </Group>

          {/* The message only. A stack trace on screen helps nobody here and
              is in the console for anyone it does help. */}
          <Text
            size="xs"
            c="dimmed"
            ta="center"
            style={{ wordBreak: 'break-word' }}
          >
            {error.message}
          </Text>
        </Stack>
      </Container>
    );
  }
}

export default ErrorBoundary;
