import type { Example, ApiResponse } from "../types/example-type";
import { apiRequest, throwForFailedResponse } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";
import { createEntityStore } from "./create-entity-store";
import { readErrorMessage } from "../utils/api";

interface ExampleExtras {
  // GET /example?exampleId=N. Returns null when the row does not exist.
  readExampleById: (id: number) => Promise<Example | null>;
}

export const useExamplesStore = createEntityStore<Example, ExampleExtras>(
  {
    idParamName: "exampleId",
    idField: "exampleId",
    listEndpoint: API_ENDPOINTS.EXAMPLE_LIST,
    itemEndpoint: API_ENDPOINTS.EXAMPLE,
  },
  (set, get) => ({
    readExampleById: async (id: number) => {
      const response = await apiRequest(
        `${API_ENDPOINTS.EXAMPLE}?exampleId=${id}`,
      );
      if (response.status === 404 || response.status === 400) return null;
      if (!response.ok) {
        await throwForFailedResponse(response, readErrorMessage);
      }
      const example = ((await response.json()) as ApiResponse<Example>).data;
      // Refresh the cached row if it is present; do not add it otherwise, or
      // the list would grow without a matching maxUpdateTs.
      set({
        itemList: get().itemList.map((e) =>
          e.exampleId === example.exampleId ? example : e,
        ),
      });
      return example;
    },
  }),
);
