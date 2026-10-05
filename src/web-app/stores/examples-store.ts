import type { Example } from "../types/example-type";
import type { ApiResponse } from "../types/api";
import { apiJson, isApiStatus } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";
import { createEntityStore } from "./create-entity-store";

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
      let example: Example;
      try {
        ({ data: example } = await apiJson<ApiResponse<Example>>(
          `${API_ENDPOINTS.EXAMPLE}?exampleId=${id}`,
        ));
      } catch (err) {
        if (isApiStatus(err, 404, 400)) return null;
        throw err;
      }
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
