import { beforeEach, describe, expect, it, vi } from "vitest";
import { createEntityStore } from "../stores/create-entity-store";

// The cache rules are the subtle part of this store: advancing
// maxUpdateTs at the wrong moment silently drops other people's changes, and
// ignoring a tombstone leaves deleted rows on screen forever. Neither failure
// is visible in a typecheck, and neither would show up until a user noticed
// stale data -- so they are tested directly.

interface Widget {
  widgetId: number;
  label: string;
  updateTs: number;
  deleteTs?: number;
}

const buildStore = () =>
  createEntityStore<Widget>({
    idParamName: "widgetId",
    idField: "widgetId",
    listEndpoint: "/widgetList",
    itemEndpoint: "/widget",
  });

const mockJson = (data: unknown, status = 200) =>
  Promise.resolve(
    new Response(JSON.stringify({ data }), {
      status,
      headers: { "Content-Type": "application/json" },
    }),
  );

describe("entity store cache management", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("advances maxUpdateTs on a full read", async () => {
    const useStore = buildStore();
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        mockJson([
          { widgetId: 1, label: "a", updateTs: 100 },
          { widgetId: 2, label: "b", updateTs: 250 },
        ]),
      ),
    );

    await useStore.getState().readList();

    expect(useStore.getState().itemList).toHaveLength(2);
    expect(useStore.getState().maxUpdateTs).toBe(250);
  });

  it("does NOT advance maxUpdateTs on an incremental read", async () => {
    const useStore = buildStore();
    vi.stubGlobal(
      "fetch",
      vi.fn(() => mockJson([{ widgetId: 1, label: "a", updateTs: 100 }])),
    );
    await useStore.getState().readList();
    expect(useStore.getState().maxUpdateTs).toBe(100);

    // A newer row arrives incrementally. The watermark must stay put: advancing
    // it here would skip past changes made by others in the same interval.
    vi.stubGlobal(
      "fetch",
      vi.fn(() => mockJson([{ widgetId: 2, label: "b", updateTs: 900 }])),
    );
    await useStore.getState().readList({ incremental: true });

    expect(useStore.getState().maxUpdateTs).toBe(100);
    expect(useStore.getState().itemList).toHaveLength(2);
  });

  it("merges an incremental read rather than replacing the list", async () => {
    const useStore = buildStore();
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        mockJson([
          { widgetId: 1, label: "a", updateTs: 100 },
          { widgetId: 2, label: "b", updateTs: 100 },
        ]),
      ),
    );
    await useStore.getState().readList();

    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        mockJson([{ widgetId: 2, label: "b-edited", updateTs: 400 }]),
      ),
    );
    await useStore.getState().readList({ incremental: true });

    const labelById = Object.fromEntries(
      useStore.getState().itemList.map((w) => [w.widgetId, w.label]),
    );
    expect(labelById).toEqual({ 1: "a", 2: "b-edited" });
  });

  it("removes a row that arrives as a tombstone", async () => {
    const useStore = buildStore();
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        mockJson([
          { widgetId: 1, label: "a", updateTs: 100 },
          { widgetId: 2, label: "b", updateTs: 100 },
        ]),
      ),
    );
    await useStore.getState().readList();
    expect(useStore.getState().itemList).toHaveLength(2);

    // The server sends deleted rows with deleteTs set on incremental reads.
    // Without this the row would stay on screen indefinitely.
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        mockJson([{ widgetId: 1, label: "a", updateTs: 500, deleteTs: 500 }]),
      ),
    );
    await useStore.getState().readList({ incremental: true });

    expect(useStore.getState().itemList.map((w) => w.widgetId)).toEqual([2]);
  });

  it("leaves maxUpdateTs alone on create, update and delete", async () => {
    const useStore = buildStore();
    vi.stubGlobal(
      "fetch",
      vi.fn(() => mockJson([{ widgetId: 1, label: "a", updateTs: 100 }])),
    );
    await useStore.getState().readList();
    expect(useStore.getState().maxUpdateTs).toBe(100);

    vi.stubGlobal(
      "fetch",
      vi.fn(() => mockJson({ widgetId: 2, label: "new", updateTs: 999 })),
    );
    await useStore.getState().createItem({ label: "new" });
    expect(useStore.getState().maxUpdateTs).toBe(100);

    vi.stubGlobal(
      "fetch",
      vi.fn(() => mockJson({ widgetId: 2, label: "edited", updateTs: 1200 })),
    );
    await useStore.getState().updateItem({ widgetId: 2, label: "edited" });
    expect(useStore.getState().maxUpdateTs).toBe(100);

    vi.stubGlobal(
      "fetch",
      vi.fn(() => mockJson({ widgetId: 2 })),
    );
    await useStore.getState().deleteItem(2, 1200);
    expect(useStore.getState().maxUpdateTs).toBe(100);
    expect(useStore.getState().itemList.map((w) => w.widgetId)).toEqual([1]);
  });

  it("unwraps a create response that carries more than the row", async () => {
    // Some create endpoints answer with the row alongside side-effect data.
    // Storing the wrapper would put a non-entity object in itemList.
    const useStore = createEntityStore<Widget>({
      idParamName: "widgetId",
      idField: "widgetId",
      listEndpoint: "/widgetList",
      itemEndpoint: "/widget",
      unwrapCreated: (payload) =>
        (payload as { widget: Widget }).widget ?? (payload as Widget),
    });

    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        mockJson({
          widget: { widgetId: 7, label: "created", updateTs: 500 },
          extraCount: 3,
        }),
      ),
    );

    const created = await useStore.getState().createItem({ label: "created" });
    expect(created.widgetId).toBe(7);
    expect(useStore.getState().itemList).toEqual([
      { widgetId: 7, label: "created", updateTs: 500 },
    ]);
  });

  it("sends afterUpdateTs only on an incremental read", async () => {
    const useStore = buildStore();
    const fetchMock = vi.fn((...args: unknown[]) => {
      void args;
      return mockJson([{ widgetId: 1, label: "a", updateTs: 100 }]);
    });
    vi.stubGlobal("fetch", fetchMock);

    await useStore.getState().readList();
    expect(fetchMock.mock.calls[0]?.[0]).toBe("/api/widgetList");

    await useStore.getState().readList({ incremental: true });
    expect(fetchMock.mock.calls[1]?.[0]).toBe(
      "/api/widgetList?afterUpdateTs=100",
    );
  });
});
