// Help usage, for the Help insights page (handler/help_handler.go). Fire
// and forget: never awaited, never shown; a failure is dropped, since
// usage data is best-effort (doc/151-help-module.md).
import { API_ENDPOINTS } from "../constants/api";
import { apiJson, jsonBody } from "../utils/api";

export type HelpEventName =
  | "search_no_result"
  | "entry_view"
  | "tour_complete"
  | "tour_skip";

interface HelpEventField {
  route: string;
  // A help entry or tour id; every event but search_no_result.
  entryId?: string;
  // What was typed; search_no_result only.
  query?: string;
}

export function logHelpEvent(
  event: HelpEventName,
  field: HelpEventField,
): void {
  apiJson(API_ENDPOINTS.HELP_EVENT, {
    method: "POST",
    ...jsonBody({ event, ...field }),
  }).catch(() => undefined);
}
