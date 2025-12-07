import { create } from 'zustand';
import { Example } from '../features/example/example-type';

interface ExamplesState {
  examples: Example[];
  maxUpdateTs: number;
  loading: boolean;
  error: string | null;
  fetchExamples: () => Promise<void>;
  createExample: (data: Partial<Example>) => Promise<void>;
  updateExample: (id: number, data: Partial<Example>) => Promise<void>;
  syncUpdates: () => Promise<void>;
}

const getCsrfToken = () => sessionStorage.getItem('csrfToken');

const apiFetch = async (url: string, options: RequestInit = {}) => {
  const headers = new Headers(options.headers);
  const csrfToken = sessionStorage.getItem('csrfToken');
  if (csrfToken) {
    headers.set('X-CSRF-Token', csrfToken);
  }
  return fetch(`/api${url}`, {
    ...options,
    headers,
    credentials: 'include',
  });
};

export const useExamplesStore = create<ExamplesState>((set, get) => ({
  examples: [],
  maxUpdateTs: 0,
  loading: false,
  error: null,
  fetchExamples: async () => {
    set({ loading: true, error: null });
    try {
      const response = await apiFetch('/exampleList');
      if (!response.ok) throw new Error('Failed to fetch examples');
      const data = await response.json();
      const examples = data.data.map((e: any) => ({
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
      const maxUpdateTs = examples.length > 0 ? Math.max(...examples.map(e => e.updateTs)) : 0;
      set({ examples, maxUpdateTs });
    } catch (err: any) {
      set({ error: err.message });
    } finally {
      set({ loading: false });
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
      const response = await apiFetch('/example', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      if (!response.ok) throw new Error('Failed to create example');
      // Refetch to update list
      await get().fetchExamples();
    } catch (err: any) {
      set({ error: err.message });
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
      const response = await apiFetch('/example', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      if (!response.ok) throw new Error('Failed to update example');
      // Optionally refetch or update local state
      await get().fetchExamples();
    } catch (err: any) {
      set({ error: err.message });
    }
  },
  syncUpdates: async () => {
    const { maxUpdateTs } = get();
    if (maxUpdateTs === 0) {
      // If no data, do full fetch
      await get().fetchExamples();
      return;
    }
    try {
      const response = await apiFetch(`/exampleList?afterUpdateTs=${maxUpdateTs}`);
      if (!response.ok) throw new Error('Failed to sync examples');
      const data = await response.json();
      const newExamples = data.data.map((e: any) => ({
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
        set(state => {
          const updatedExamples = [...state.examples];
          newExamples.forEach(newEx => {
            const index = updatedExamples.findIndex(ex => ex.exampleId === newEx.exampleId);
            if (index >= 0) {
              updatedExamples[index] = newEx;
            } else {
              updatedExamples.push(newEx);
            }
          });
          const newMaxUpdateTs = Math.max(state.maxUpdateTs, ...newExamples.map(e => e.updateTs));
          return { examples: updatedExamples, maxUpdateTs: newMaxUpdateTs };
        });
      }
    } catch (err: any) {
      // On error, do full fetch
      await get().fetchExamples();
    }
  },
}));