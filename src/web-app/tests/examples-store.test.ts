import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useExamplesStore } from "../stores/examples-store";
import type { Example } from "../types/example-type";

// The store test a new entity copies (src/web-app/README.md, Scaffolding).
// Go's JSON decoder drops a body key it does not know, so a key that
// differs from the json tag reaches the handler as a zero value and nothing
// fails until a user tries it (B013). Each write's URL and body is asserted
// against the tags in src/api-srv/internal/model/example_model.go.
// readExampleById's 404/400 -> null and throw-on-anything-else are in
// api-json.test.ts.

const row: Example = {
  exampleId: 7,
  exampleText: "seven",
  exampleInt: 3,
  exampleUuid: "00000000-0000-4000-8000-000000000007",
  securityContext: 2,
  createTs: 100,
  createUserId: 1,
  updateTs: 200,
  updateUserId: 1,
};

const recordFetch = (data: unknown) => {
  const fetchMock = vi.fn<
    (input: string, init?: RequestInit) => Promise<Response>
  >(() =>
    Promise.resolve(
      new Response(JSON.stringify({ data }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    ),
  );
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
};

const sentBody = (init?: RequestInit): unknown =>
  JSON.parse(init?.body as string);

describe("examples store requests", () => {
  beforeEach(() => {
    useExamplesStore.getState().clear();
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("readExampleById reads ?exampleId= and refreshes a cached row",
    async () => {
      useExamplesStore.setState({
        itemList: [{ ...row, exampleText: "stale", updateTs: 150 }],
      });
      const fetchMock = recordFetch(row);

      const loaded = await useExamplesStore.getState().readExampleById(7);

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/example?exampleId=7");
      expect(init?.method ?? "GET").toBe("GET");
      expect(loaded).toEqual(row);
      expect(useExamplesStore.getState().itemList).toEqual([row]);
    });

  it("createItem posts the model's json keys", async () => {
    const fetchMock = recordFetch(row);
    const data = {
      exampleText: "seven",
      exampleInt: 3,
      exampleFloatArray: [1.5],
      exampleJsonb: { colour: "teal" },
    };

    await useExamplesStore.getState().createItem(data);

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/example");
    expect(init?.method).toBe("POST");
    expect(sentBody(init)).toEqual(data);
  });

  it("updateItem puts exampleId and updateTs with the fields", async () => {
    const fetchMock = recordFetch(row);

    await useExamplesStore
      .getState()
      .updateItem({ exampleId: 7, exampleText: "seven", updateTs: 200 });

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/example");
    expect(init?.method).toBe("PUT");
    expect(sentBody(init)).toEqual({
      exampleId: 7,
      exampleText: "seven",
      updateTs: 200,
    });
  });

  it("deleteItem sends exampleId and updateTs as query parameters",
    async () => {
      const fetchMock = recordFetch(null);

      await useExamplesStore.getState().deleteItem(7, 200);

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/example?exampleId=7&updateTs=200");
      expect(init?.method).toBe("DELETE");
      expect(init?.body).toBeUndefined();
    });
});
