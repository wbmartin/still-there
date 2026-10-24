import { notifications } from "@mantine/notifications";
import { isApiStatus } from "./api";
import { reportError } from "./report-error";

// The toasts the monitoring pages show after a save, delete or action
// (doc/810, UI feedback), in the example pages' colours and timings.

export const notifySuccess = (message: string): void => {
  notifications.show({
    title: "Success",
    message,
    color: "green",
    autoClose: 3000,
    withCloseButton: false,
  });
};

/**
 * Report a failed call and tell the user. A 400 or 409 carries the API's
 * own reason (a rule the form could not check, a change someone else made
 * first), which says more than `fallback`.
 */
export const notifyFailure = (
  error: unknown,
  context: string,
  fallback: string,
): void => {
  reportError(error, context);
  notifications.show({
    title: "Error",
    message:
      isApiStatus(error, 400, 409) && error instanceof Error
        ? error.message
        : fallback,
    color: "red",
    autoClose: 8000,
    withCloseButton: true,
  });
};
