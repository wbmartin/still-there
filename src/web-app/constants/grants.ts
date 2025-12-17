export const grants = {
  // Grant ID constants (matching API short codes)
  READ_EXAMPLE: 1001,
  CREATE_EXAMPLE: 1002,
  UPDATE_EXAMPLE: 1003,
  DELETE_EXAMPLE: 1004,
  CHG_OWN_PW: 8001,
  CREATE_USER: 9001,
  BLOCK_USER: 9002,
  CHANGE_USER_PW: 9003,
  READ_USER_GROUP: 9010,
  CREATE_USER_GROUP: 9011,
  UPDATE_USER_GROUP: 9012,
  DELETE_USER_GROUP: 9013,
  ADD_USER_TO_GROUP: 9014,
  REMOVE_USER_FROM_GROUP: 9015,

  // Utility methods
  idToName(id: number): string {
    const name = Object.keys(this).find(
      (key) => this[key as keyof typeof this] === id,
    );
    return name || `???`;
  },

  nameToId(name: string): number | undefined {
    return this[name as keyof typeof this] as number;
  },

  // Get all grant IDs
  getAllIds(): number[] {
    return Object.values(this).filter((val) => typeof val === "number");
  },
};
