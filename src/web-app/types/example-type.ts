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
