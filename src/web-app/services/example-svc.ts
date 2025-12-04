import { apiSvc } from "./api-svc";

export interface Example {
  example_id: number;
  example_text: string;
  example_int?: number;
  example_float?: number;
  example_int_array?: number[];
  example_float_array?: number[];
  security_context: number;
  create_ts: number;
  create_user_id: number;
  update_ts: number;
  update_user_id: number;
  delete_ts?: number;
  delete_user_id?: number;
}

export const exampleApi = apiSvc.injectEndpoints({
  endpoints: (build) => ({
    getExamples: build.query<{ examples: Example[]; count: number }, void>({
      query: () => "/examples",
    }),
    getExample: build.query<Example, number>({
      query: (id) => `/examples/${id}`,
    }),
    createExample: build.mutation<Example, Partial<Example>>({
      query: (body) => ({
        url: "/examples",
        method: "POST",
        body,
      }),
    }),
    updateExample: build.mutation<Example, { id: number; data: Partial<Example> }>({
      query: ({ id, data }) => ({
        url: `/examples/${id}`,
        method: "PUT",
        body: data,
      }),
    }),
    deleteExample: build.mutation<void, number>({
      query: (id) => ({
        url: `/examples/${id}`,
        method: "DELETE",
      }),
    }),
  }),
});

export const {
  useGetExamplesQuery,
  useGetExampleQuery,
  useCreateExampleMutation,
  useUpdateExampleMutation,
  useDeleteExampleMutation,
} = exampleApi;