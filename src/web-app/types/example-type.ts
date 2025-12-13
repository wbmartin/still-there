// Example Types
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

export interface ApiExampleModel {
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

export interface ApiResponse<T> {
  data: T;
}

export interface ExampleListResponse {
  examples: Example[];
}

export interface ExampleFormData {
  exampleText: string;
  exampleInt: string;
  exampleFloat: string;
  exampleIntArray: string;
  exampleFloatArray: string;
  updateTs: string;
}

export const defaultExampleFormData: ExampleFormData = {
  exampleText: "",
  exampleInt: "",
  exampleFloat: "",
  exampleIntArray: "",
  exampleFloatArray: "",
  updateTs: "",
};

export const shellExample: Example = {
  exampleId: 0,
  exampleText: "",
  exampleInt: undefined,
  exampleFloat: undefined,
  exampleIntArray: undefined,
  exampleFloatArray: undefined,
  securityContext: 0,
  createTs: 0,
  createUserId: 0,
  updateTs: 0,
  updateUserId: 0,
  deleteTs: undefined,
  deleteUserId: undefined,
};