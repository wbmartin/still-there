// Example Types
export interface Example {
  exampleId: number;
  exampleText: string;
  exampleInt?: number;
  exampleFloat?: number;
  exampleIntArray?: number[];
  exampleFloatArray?: number[];
  // Set by the server on create; never changes. Absent on a create body.
  exampleUuid?: string;
  // Any JSON value, or absent.
  exampleJsonb?: unknown;
  securityContext: number;
  createTs: number;
  createUserId: number;
  updateTs: number;
  updateUserId: number;
  deleteTs?: number;
  deleteUserId?: number;
}




export interface ExampleFormData {
  exampleText: string;
  exampleInt: string;
  exampleFloat: string;
  exampleIntArray: string;
  exampleFloatArray: string;
  // JSON text as typed; parsed on submit. Empty means absent.
  exampleJsonb: string;
  updateTs: string;
}

export const defaultExampleFormData: ExampleFormData = {
  exampleText: "",
  exampleInt: "",
  exampleFloat: "",
  exampleIntArray: "",
  exampleFloatArray: "",
  exampleJsonb: "",
  updateTs: "",
};
