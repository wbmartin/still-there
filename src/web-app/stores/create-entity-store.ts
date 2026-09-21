import { create, type StateCreator } from "zustand";
import { apiRequest, throwForFailedResponse } from "../utils/api";
import { registerCacheClearer } from "../utils/cache-utils";

// Anything the API returns carries these. deleteTs only ever arrives on an
// incremental read, where the server sends soft-deleted rows as tombstones.
export interface EntityBase {
  updateTs: number;
  deleteTs?: number;
}

export interface EntityStoreState<TEntity> {
  itemList: TEntity[];
  maxUpdateTs: number;
  loading: boolean;
  error: string | null;

  readList: (options?: ReadListOptions) => Promise<void>;
  createItem: (item: Partial<TEntity>) => Promise<TEntity>;
  updateItem: (item: Partial<TEntity>) => Promise<TEntity>;
  deleteItem: (id: number, updateTs: number) => Promise<void>;
  clear: () => void;
}

export interface ReadListOptions {
  // Fetch only what changed since maxUpdateTs and merge it, rather than
  // replacing the list. Falls back to a full read when nothing is cached yet.
  incremental?: boolean;
  // Extra query parameters, e.g. { appUserIdList: "1,2" }.
  queryParams?: Record<string, string | number>;
}

export interface EntityStoreConfig<TEntity> {
  // Query-string name for this entity's id, e.g. "exampleId".
  idParamName: string;
  // Field on the model holding the primary key, e.g. "exampleId".
  idField: keyof TEntity;
  listEndpoint: string;
  itemEndpoint: string;
  // Pulls the entity out of a create response that returns more than the row.
  unwrapCreated?: (payload: unknown) => TEntity;
}

// Domain stores add their own actions on top of the CRUD base. `set`/`get`
// see the combined store, so an extension can read `itemList` directly.
export type EntityStoreExtension<TEntity, TExtra> = StateCreator<
  EntityStoreState<TEntity> & TExtra,
  [],
  [],
  TExtra
>;

export const readErrorMessage = async (response: Response): Promise<string> => {
  try {
    const body = (await response.json()) as {
      error?: string;
      message?: string;
    };
    return body.error || body.message || `Request failed (${response.status})`;
  } catch {
    return `Request failed (${response.status})`;
  }
};

/**
 * Builds a Zustand store implementing the entity cache-management pattern.
 *
 * The rules that matter, all of which are easy to get subtly wrong per entity:
 *
 *   - maxUpdateTs advances ONLY on a full list read. Create, update, delete and
 *     incremental merges leave it alone. Advancing it on a mutation would skip
 *     over changes other people made in the same interval, and those changes
 *     would never arrive.
 *   - An incremental read merges rather than replaces, so rows the filter did
 *     not return stay in the cache.
 *   - A row arriving with deleteTs set is a TOMBSTONE and is removed from the
 *     cache rather than stored. Without it a deleted row would linger in the
 *     UI forever.
 *   - Every store registers its own cache clearer, so sign-out and context
 *     switches cannot leave another tenant's rows in memory.
 */
export const createEntityStore = <
  TEntity extends EntityBase,
  TExtra extends object = Record<never, never>,
>(
  config: EntityStoreConfig<TEntity>,
  extend?: EntityStoreExtension<TEntity, TExtra>,
) => {
  type Store = EntityStoreState<TEntity> & TExtra;

  const useStore = create<Store>()((set, get, api) => {
    const base: EntityStoreState<TEntity> = {
      itemList: [],
      maxUpdateTs: 0,
      loading: false,
      error: null,

      readList: async (options: ReadListOptions = {}) => {
        set({ loading: true, error: null } as Partial<Store>);
        try {
          const searchParams = new URLSearchParams();
          Object.entries(options.queryParams ?? {}).forEach(([key, value]) => {
            searchParams.set(key, String(value));
          });

          const isIncremental =
            options.incremental === true && get().maxUpdateTs > 0;
          if (isIncremental) {
            searchParams.set("afterUpdateTs", String(get().maxUpdateTs));
          }

          const query = searchParams.toString();
          const response = await apiRequest(
            `${config.listEndpoint}${query ? `?${query}` : ""}`,
          );
          if (!response.ok) {
            await throwForFailedResponse(response, readErrorMessage);
          }
          const body = (await response.json()) as { data?: TEntity[] };
          const returnedList: TEntity[] = body.data ?? [];

          if (!isIncremental) {
            // Full read. The server excludes deleted rows here, so whatever
            // comes back is the complete live set.
            const highestUpdateTs = returnedList.reduce(
              (highest, item) => Math.max(highest, item.updateTs),
              0,
            );
            set({
              itemList: returnedList,
              maxUpdateTs: highestUpdateTs,
              loading: false,
            } as Partial<Store>);
            return;
          }

          // Incremental read. Merge changes, drop tombstones, and deliberately
          // leave maxUpdateTs untouched.
          const mergedById = new Map<unknown, TEntity>();
          get().itemList.forEach((item) => {
            mergedById.set(item[config.idField], item);
          });
          returnedList.forEach((item) => {
            if (item.deleteTs) {
              mergedById.delete(item[config.idField]);
            } else {
              mergedById.set(item[config.idField], item);
            }
          });
          set({
            itemList: Array.from(mergedById.values()),
            loading: false,
          } as Partial<Store>);
        } catch (err) {
          set({
            loading: false,
            error: err instanceof Error ? err.message : "Failed to load",
          } as Partial<Store>);
          throw err;
        }
      },

      createItem: async (item: Partial<TEntity>) => {
        const response = await apiRequest(config.itemEndpoint, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(item),
        });
        if (!response.ok) {
          await throwForFailedResponse(response, readErrorMessage);
        }
        const payload = ((await response.json()) as { data: unknown }).data;
        const created: TEntity = config.unwrapCreated
          ? config.unwrapCreated(payload)
          : (payload as TEntity);
        // Local cache only; maxUpdateTs stays where it is.
        set({ itemList: [...get().itemList, created] } as Partial<Store>);
        return created;
      },

      updateItem: async (item: Partial<TEntity>) => {
        const response = await apiRequest(config.itemEndpoint, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(item),
        });
        if (!response.ok) {
          await throwForFailedResponse(response, readErrorMessage);
        }
        const updated = ((await response.json()) as { data: TEntity }).data;
        set({
          itemList: get().itemList.map((existing) =>
            existing[config.idField] === updated[config.idField]
              ? updated
              : existing,
          ),
        } as Partial<Store>);
        return updated;
      },

      deleteItem: async (id: number, updateTs: number) => {
        const searchParams = new URLSearchParams({
          [config.idParamName]: String(id),
          updateTs: String(updateTs),
        });
        const response = await apiRequest(
          `${config.itemEndpoint}?${searchParams.toString()}`,
          { method: "DELETE" },
        );
        if (!response.ok) {
          await throwForFailedResponse(response, readErrorMessage);
        }
        set({
          itemList: get().itemList.filter(
            (existing) => existing[config.idField] !== id,
          ),
        } as Partial<Store>);
      },

      clear: () =>
        set({ itemList: [], maxUpdateTs: 0, error: null } as Partial<Store>),
    };

    const extra = extend ? extend(set, get, api) : ({} as TExtra);
    return { ...base, ...extra } as Store;
  });

  // Sign-out and context switches must not leave another tenant's rows in
  // memory. Registering here means a new entity cannot forget to.
  registerCacheClearer(() => useStore.getState().clear());

  return useStore;
};
