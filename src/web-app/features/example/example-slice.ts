import { createSlice } from "@reduxjs/toolkit";
import { Example } from "../../services/example-svc";

interface ExampleState {
  examples: Example[];
  count: number;
}

const initialState: ExampleState = {
  examples: [],
  count: 0,
};

const exampleSlice = createSlice({
  name: "example",
  initialState,
  reducers: {},
});

export default exampleSlice.reducer;