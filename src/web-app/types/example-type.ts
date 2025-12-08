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
  example_id: number;
  example_text: string;
  example_int?: number;
  example_float?: number;
  example_int_array?: number[];
  example_float_array?: number[];
  security_context: number;
  create_ts: number;
  create_user_id: number;
  update_ts: number;
  update_user_id: number;
  delete_ts?: number;
  delete_user_id?: number;
}

export interface ApiResponse<T> {
  data: T;
}

export interface ExampleListResponse {
  examples: Example[];
}

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