import { apiSvc } from "./api-svc";

export interface Example {
  exampleId: number;
  exampleText: string;
  exampleInt?: number;
  exampleFloat?: number;
  exampleIntArray?: number[];
  exampleFloatArray?: number[];
  securityContext: number;
  createTs: number;
  createUserId: number;
  updateTs: number;
  updateUserId: number;
  deleteTs?: number;
  deleteUserId?: number;
}

const transformExample = (e: any): Example => ({
  exampleId: e.example_id,
  exampleText: e.example_text,
  exampleInt: e.example_int,
  exampleFloat: e.example_float,
  exampleIntArray: e.example_int_array,
  exampleFloatArray: e.example_float_array,
  securityContext: e.security_context,
  createTs: e.create_ts,
  createUserId: e.create_user_id,
  updateTs: e.update_ts,
  updateUserId: e.update_user_id,
  deleteTs: e.delete_ts,
  deleteUserId: e.delete_user_id,
});

export const exampleApi = apiSvc.injectEndpoints({
  tagTypes: ["ExampleList", "Example"],
  endpoints: (build) => ({
    getExamples: build.query<{ examples: Example[] }, void>({
      query: () => "/exampleList",
      providesTags: ["ExampleList"],
      transformResponse: (response: any) => ({
        examples: response.data.map(transformExample),
      }),
    }),
    getExample: build.query<Example, number>({
      query: (id) => `/example?exampleId=${id}`,
      providesTags: ["Example"],
      transformResponse: (response: any) => transformExample(response.data),
    }),
    createExample: build.mutation<Example, Partial<Example>>({
      query: (body) => ({
        url: "/example",
        method: "POST",
        body: {
          example_text: body.exampleText,
          example_int: body.exampleInt,
          example_float: body.exampleFloat,
          example_int_array: body.exampleIntArray,
          example_float_array: data.exampleFloatArray,
        },
      }),
      invalidatesTags: ["ExampleList"],
      transformResponse: (response: any) => transformExample(response.data),
    }),
    updateExample: build.mutation<Example, { id: number; data: Partial<Example> }>({
      query: ({ id, data }) => ({
        url: "/example",
        method: "PUT",
        body: {
          example_id: id,
          example_text: data.exampleText,
          example_int: data.exampleInt,
          example_float: data.exampleFloat,
          example_int_array: data.exampleIntArray,
          example_float_array: data.exampleFloatArray,
          update_ts: data.updateTs,
        },
      }),
      invalidatesTags: ["ExampleList", "Example"],
      transformResponse: (response: any) => transformExample(response.data),
    }),
    deleteExample: build.mutation<void, { id: number; updateTs: number }>({
      query: ({ id, updateTs }) => ({
        url: `/example?exampleId=${id}&updateTs=${updateTs}`,
        method: "DELETE",
      }),
      invalidatesTags: ["ExampleList", "Example"],
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