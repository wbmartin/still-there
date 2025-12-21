import React, { Component } from 'react';
import type { ReactNode } from 'react';
import { Alert, Button, Collapse, Group, Stack, Text, Title, Container } from '@mantine/core';

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
  reportingStatus: 'idle' | 'reporting' | 'reported' | 'failed';
}

interface ErrorBoundaryProps {
  children: ReactNode;
}

class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      reportingStatus: 'idle'
    };
  }

  async reportError(error: Error) {
    this.setState({ reportingStatus: 'reporting' });
    try {
      const response = await fetch('/error-report', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          message: error.message,
          stack: error.stack,
          userAgent: navigator.userAgent,
          url: window.location.href,
          timestamp: new Date().toISOString()
        })
      });
      if (response.ok) {
        this.setState({ reportingStatus: 'reported' });
      } else {
        throw new Error('Report failed');
      }
    } catch (reportError) {
      console.error('Failed to report error:', reportError);
      this.setState({ reportingStatus: 'failed' });
    }
  }

  static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryState> {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('Application Error:', error, errorInfo);
    void this.reportError(error);
  }

  render() {
    if (this.state.hasError) {
      const { error, reportingStatus } = this.state;

      return (
        <Container p="2rem" maw="600px" m="0 auto">
          <Stack gap="lg">
            <Title order={1} ta="center">Something went wrong</Title>

            {reportingStatus === 'reported' && (
              <Alert color="green" variant="light">
                <Text>
                  We've automatically reported this issue to help us fix it.
                </Text>
              </Alert>
            )}

            {reportingStatus === 'failed' && error && (
               <Alert color="red" variant="light">
                 <Text>
                   We tried to report this issue but couldn't.
                   {/* eslint-disable-next-line max-len */}
                   Please take a screenshot of this page and send it to your administrator to help us troubleshoot.
                 </Text>
               </Alert>
            )}

            {reportingStatus === 'failed' && error && (
              <Collapse in={reportingStatus === 'failed'}>
                <Alert
                  color="orange"
                  variant="outline"
                  title="Error Details (for administrator)"
                >
                  <Text size="sm" style={{ fontFamily: 'monospace', whiteSpace: 'pre-wrap' }}>
                    {error.message}
                    {'\n\n'}
                    {error.stack}
                  </Text>
                </Alert>
              </Collapse>
            )}

            <Group justify="center" gap="md">
              <Button onClick={() => window.location.reload()}>
                Refresh Page
              </Button>
              <Button
                variant="outline"
                onClick={() => window.location.href = '/dashboard'}
              >
                Go to Dashboard
              </Button>
            </Group>
          </Stack>
        </Container>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;