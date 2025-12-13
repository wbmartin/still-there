import { create } from "zustand";
import type { Example, ApiExampleModel, ApiResponse } from "../types/example-type";
import { apiRequest } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";

interface ExamplesState {
  examples: Example[];
  maxExamplesUpdateTs: number;
  loading: boolean;
  error: string | null;
  readExampleById: (id: number) => Promise<Example | null>;
  createExample: (data: Partial<Example>) => Promise<void>;
  updateExample: (id: number, data: Partial<Example>) => Promise<void>;
  deleteExample: (id: number) => Promise<void>;
  readExampleList: () => Promise<void>;
  clearCache: () => void;
}

export const useExamplesStore = create<ExamplesState>((set, get) => ({
  examples: [],
  maxExamplesUpdateTs: 0,
  loading: false,
  error: null,
  //============================^FUNCTION LINE ^============================
  readExampleList: async () => {
    // console.log("getExampleList called");
    const { maxExamplesUpdateTs } = get();
    set({ loading: true });
    try {
      if (maxExamplesUpdateTs === 0) {
        // Fetch all examples
        const response = await apiRequest(API_ENDPOINTS.EXAMPLE_LIST);
        if (!response.ok) throw new Error("Failed to fetch examples");
        const data = await response.json() as ApiResponse<ApiExampleModel[]>;
        const examples = data.data.map((e) => ({
          exampleId: e.exampleId,
          exampleText: e.exampleText,
          exampleInt: e.exampleInt,
          exampleFloat: e.exampleFloat,
          exampleIntArray: e.exampleIntArray,
          exampleFloatArray: e.exampleFloatArray,
          securityContext: e.securityContext,
          createTs: e.createTs,
          createUserId: e.createUserId,
          updateTs: e.updateTs,
          updateUserId: e.updateUserId,
        }));
        const newMaxExamplesUpdateTs =
          examples.length > 0
            ? Math.max(...examples.map((e) => e.updateTs))
            : 0;
        set({ examples, maxExamplesUpdateTs: newMaxExamplesUpdateTs, loading: false });
      } else {
        // Fetch updates
        const response = await apiRequest(`/exampleList?afterUpdateTs=${maxExamplesUpdateTs}`);
        if (!response.ok) throw new Error("Failed to sync examples");
        const data = await response.json() as ApiResponse<ApiExampleModel[]>;
        const newExamples = data.data.map((e) => ({
          exampleId: e.exampleId,
          exampleText: e.exampleText,
          exampleInt: e.exampleInt,
          exampleFloat: e.exampleFloat,
          exampleIntArray: e.exampleIntArray,
          exampleFloatArray: e.exampleFloatArray,
          securityContext: e.securityContext,
          createTs: e.createTs,
          createUserId: e.createUserId,
          updateTs: e.updateTs,
          updateUserId: e.updateUserId,
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
  //============================^FUNCTION LINE ^============================
  readExampleById: async (id: number): Promise<Example | null> => {
    try {
      const response = await apiRequest(`/example?exampleId=${id}`);
      if (!response.ok) {
        if (response.status === 404 || response.status === 400) return null;
        throw new Error("Failed to fetch example");
      }
        const data = await response.json() as ApiResponse<ApiExampleModel>;
      const apiExample = data.data;
      const example: Example = {
        exampleId: apiExample.exampleId,
        exampleText: apiExample.exampleText,
        exampleInt: apiExample.exampleInt,
        exampleFloat: apiExample.exampleFloat,
        exampleIntArray: apiExample.exampleIntArray,
        exampleFloatArray: apiExample.exampleFloatArray,
        securityContext: apiExample.securityContext,
        createTs: apiExample.createTs,
        createUserId: apiExample.createUserId,
        updateTs: apiExample.updateTs,
        updateUserId: apiExample.updateUserId,
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
  //============================^FUNCTION LINE ^============================
  createExample: async (data) => {
    set({ error: null });
    try {
      const body = {
        exampleText: data.exampleText,
        exampleInt: data.exampleInt,
        exampleFloat: data.exampleFloat,
        exampleIntArray: data.exampleIntArray,
        exampleFloatArray: data.exampleFloatArray,
      };
      const response = await apiRequest(API_ENDPOINTS.EXAMPLE, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!response.ok) throw new Error("Failed to create example");
      const result = await response.json() as ApiResponse<ApiExampleModel>;
      const apiExample = result.data;
      const newExample: Example = {
        exampleId: apiExample.exampleId,
        exampleText: apiExample.exampleText,
        exampleInt: apiExample.exampleInt,
        exampleFloat: apiExample.exampleFloat,
        exampleIntArray: apiExample.exampleIntArray,
        exampleFloatArray: apiExample.exampleFloatArray,
        securityContext: apiExample.securityContext,
        createTs: apiExample.createTs,
        createUserId: apiExample.createUserId,
        updateTs: apiExample.updateTs,
        updateUserId: apiExample.updateUserId,
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
  //============================^FUNCTION LINE ^============================
  updateExample: async (id, data) => {
    set({ error: null });
    try {
      const body = {
        exampleId: id,
        exampleText: data.exampleText,
        exampleInt: data.exampleInt,
        exampleFloat: data.exampleFloat,
        exampleIntArray: data.exampleIntArray,
        exampleFloatArray: data.exampleFloatArray,
        updateTs: data.updateTs,
      };
      const response = await apiRequest(API_ENDPOINTS.EXAMPLE, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!response.ok) throw new Error("Failed to update example");
      // Optionally refetch or update local state
      await get().readExampleList();
    } catch (err: unknown) {
      set({ error: err instanceof Error ? err.message : "Unknown error" });
    }
  },
  //============================^FUNCTION LINE ^============================
  deleteExample: async (id) => {
    set({ error: null });
    try {
      const state = get();
      const example = state.examples.find((ex) => ex.exampleId === id);
      if (!example) throw new Error("Example not found");
      const response = await apiRequest(API_ENDPOINTS.EXAMPLE + `?exampleId=${id}&updateTs=${example.updateTs}`, {
        method: "DELETE",
      });
      if (!response.ok) throw new Error("Failed to delete example");
      // Remove from local state
      set((state) => ({
        examples: state.examples.filter((ex) => ex.exampleId !== id),
      }));
    } catch (err: unknown) {
      set({ error: err instanceof Error ? err.message : "Unknown error" });
      throw err; // Re-throw to handle in component
    }
  },
  //============================^FUNCTION LINE ^============================
  clearCache: () => {
    set({ examples: [], maxExamplesUpdateTs: 0 });
  },
}));
