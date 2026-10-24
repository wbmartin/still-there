import type { Contact } from "../types/contact-type";
import type { ApiResponse } from "../types/api";
import { apiJson } from "../utils/api";
import { API_ENDPOINTS } from "../constants/api";
import { createEntityStore } from "./create-entity-store";

// The people texted about a device. readList takes
// { queryParams: { deviceId } } for one device's; a page showing one device
// filters itemList by deviceId too.
interface ContactExtras {
  // There is no GET /contact; read through the list filtered to one id.
  // Returns null when it does not exist.
  readContactById: (id: number) => Promise<Contact | null>;
}

export const useContactStore = createEntityStore<Contact, ContactExtras>(
  {
    idParamName: "contactId",
    idField: "contactId",
    listEndpoint: API_ENDPOINTS.CONTACT_LIST,
    itemEndpoint: API_ENDPOINTS.CONTACT,
  },
  (set, get) => ({
    readContactById: async (id: number) => {
      const searchParams = new URLSearchParams({ contactIdList: String(id) });
      const body = await apiJson<ApiResponse<Contact[]> | null | undefined>(
        `${API_ENDPOINTS.CONTACT_LIST}?${searchParams.toString()}`,
      );
      const contact = body?.data?.[0];
      if (!contact) return null;
      set({
        itemList: get().itemList.map((c) =>
          c.contactId === contact.contactId ? contact : c,
        ),
      });
      return contact;
    },
  }),
);
