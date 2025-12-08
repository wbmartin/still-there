import { create } from "zustand";
import type { Example, ApiExampleModel, ApiResponse } from "../types/example-type";
import { apiRequest } from "../utils/api";

interface ExamplesState {
  examples: Example[];
  maxUpdateTs: number;
  loading: boolean;
  error: string | null;
  getExample: (id: number) => Promise<Example | null>;
  createExample: (data: Partial<Example>) => Promise<void>;
  updateExample: (id: number, data: Partial<Example>) => Promise<void>;
  getExampleList: () => Promise<void>;
  clearCache: () => void;
}

export const useExamplesStore = create<ExamplesState>((set, get) => ({
  examples: [],
  maxUpdateTs: 0,
  loading: false,
  error: null,
  getExampleList: async () => {
    // console.log("getExampleList called");
    const { maxUpdateTs } = get();
    set({ loading: true });
    try {
      if (maxUpdateTs === 0) {
        // Fetch all examples
        const csrfToken = localStorage.getItem("csrfToken");
        const response = await apiRequest(
          "/exampleList",
          {},
          csrfToken || undefined,
        );
        if (!response.ok) throw new Error("Failed to fetch examples");
        const data: ApiResponse<ApiExampleModel[]> = await response.json();
        const examples = data.data.map((e) => ({
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
        }));
        const newMaxUpdateTs =
          examples.length > 0
            ? Math.max(...examples.map((e) => e.updateTs))
            : 0;
        set({ examples, maxUpdateTs: newMaxUpdateTs, loading: false });
      } else {
        // Fetch updates
        const csrfToken = localStorage.getItem("csrfToken");
        const response = await apiRequest(
          `/exampleList?afterUpdateTs=${maxUpdateTs}`,
          {},
          csrfToken || undefined,
        );
        if (!response.ok) throw new Error("Failed to sync examples");
        const data: ApiResponse<ApiExampleModel[]> = await response.json();
        const newExamples = data.data.map((e) => ({
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
        }));
        if (newExamples.length > 0) {
          set((state) => {
            const updatedExamples = [...state.examples];
            newExamples.forEach((newEx) => {
              const index = updatedExamples.findIndex(
                (ex) => ex.exampleId === newEx.exampleId,
              );
              if (index >= 0) {
                updatedExamples[index] = newEx;
              } else {
                updatedExamples.push(newEx);
              }
            });
            const newMaxUpdateTs = Math.max(
              state.maxUpdateTs,
              ...newExamples.map((e) => e.updateTs),
            );
            return {
              examples: updatedExamples,
              maxUpdateTs: newMaxUpdateTs,
              loading: false,
            };
          });
        } else {
          set({ loading: false });
        }
      }
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : "Unknown error";
      set({ loading: false, error: errorMessage });
    } finally {
      // console.log("getExampleList finally setting loading to false");
      // Note: loading is already set in try/catch, but ensure
    }
  },
  getExample: async (id: number): Promise<Example | null> => {
    try {
      const csrfToken = localStorage.getItem("csrfToken");
      const response = await apiRequest(
        `/example?exampleId=${id}`,
        {},
        csrfToken || undefined,
      );
      if (!response.ok) {
        if (response.status === 404 || response.status === 400) return null;
        throw new Error("Failed to fetch example");
      }
      const data: ApiResponse<ApiExampleModel> = await response.json();
      const apiExample = data.data;
      const example: Example = {
        exampleId: apiExample.example_id,
        exampleText: apiExample.example_text,
        exampleInt: apiExample.example_int,
        exampleFloat: apiExample.example_float,
        exampleIntArray: apiExample.example_int_array,
        exampleFloatArray: apiExample.example_float_array,
        securityContext: apiExample.security_context,
        createTs: apiExample.create_ts,
        createUserId: apiExample.create_user_id,
        updateTs: apiExample.update_ts,
        updateUserId: apiExample.update_user_id,
        deleteTs: apiExample.delete_ts,
        deleteUserId: apiExample.delete_user_id,
      };
      // Update cached examples list if this item exists
      set((state) => {
        const index = state.examples.findIndex(
          (e) => e.exampleId === example.exampleId,
        );
        if (index >= 0) {
          const updatedExamples = [...state.examples];
          updatedExamples[index] = example;
          return { examples: updatedExamples };
        }
        return state;
      });
      return example;
    } catch (err: unknown) {
      console.error("Error fetching example:", err);
      return null;
    }
  },
  createExample: async (data) => {
    set({ error: null });
    try {
      const body = {
        example_text: data.exampleText,
        example_int: data.exampleInt,
        example_float: data.exampleFloat,
        example_int_array: data.exampleIntArray,
        example_float_array: data.exampleFloatArray,
      };
      const csrfToken = localStorage.getItem("csrfToken");
      const response = await apiRequest(
        "/example",
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        },
        csrfToken || undefined,
      );
      if (!response.ok) throw new Error("Failed to create example");
      const result: ApiResponse<ApiExampleModel> = await response.json();
      const apiExample = result.data;
      const newExample: Example = {
        exampleId: apiExample.example_id,
        exampleText: apiExample.example_text,
        exampleInt: apiExample.example_int,
        exampleFloat: apiExample.example_float,
        exampleIntArray: apiExample.example_int_array,
        exampleFloatArray: apiExample.example_float_array,
        securityContext: apiExample.security_context,
        createTs: apiExample.create_ts,
        createUserId: apiExample.create_user_id,
        updateTs: apiExample.update_ts,
        updateUserId: apiExample.update_user_id,
        deleteTs: apiExample.delete_ts,
        deleteUserId: apiExample.delete_user_id,
      };
      set((state) => ({
        examples: [...state.examples, newExample],
        maxUpdateTs: Math.max(state.maxUpdateTs, newExample.updateTs),
      }));
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : "Unknown error";
      set({ error: errorMessage });
      throw new Error(errorMessage);
    }
  },
  updateExample: async (id, data) => {
    set({ error: null });
    try {
      const body = {
        example_id: id,
        example_text: data.exampleText,
        example_int: data.exampleInt,
        example_float: data.exampleFloat,
        example_int_array: data.exampleIntArray,
        example_float_array: data.exampleFloatArray,
        update_ts: data.updateTs,
      };
      const csrfToken = localStorage.getItem("csrfToken");
      const response = await apiRequest(
        "/example",
        {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        },
        csrfToken || undefined,
      );
      if (!response.ok) throw new Error("Failed to update example");
      // Optionally refetch or update local state
      await get().getExampleList();
    } catch (err: unknown) {
      set({ error: err instanceof Error ? err.message : "Unknown error" });
    }
  },
  clearCache: () => {
    set({ examples: [], maxUpdateTs: 0 });
  },
}));
