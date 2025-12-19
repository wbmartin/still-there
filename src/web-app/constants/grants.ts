// Grant ID constants (matching API short codes)
const grantConstants = {
  LOGON: 1,
  READ_EXAMPLE: 1001,
  CREATE_EXAMPLE: 1002,
  UPDATE_EXAMPLE: 1003,
  DELETE_EXAMPLE: 1004,
  CHG_OWN_PW: 8001,
  CREATE_USER: 9001,
  READ_USER: 9002,
  UPDATE_USER: 9003,
  DELETE_USER: 9004,
  BLOCK_USER: 9008,
  CHANGE_USER_PW: 9009,
  READ_USER_GROUP: 9010,
  CREATE_USER_GROUP: 9011,
  UPDATE_USER_GROUP: 9012,
  DELETE_USER_GROUP: 9013,
  ADD_USER_TO_GROUP: 9014,
  REMOVE_USER_FROM_GROUP: 9015,
} as const;

export const grants = {
  ...grantConstants,

  // Utility methods
  idToName(id: number): string {
    const name = Object.keys(grantConstants).find(
      (key) => grantConstants[key as keyof typeof grantConstants] === id,
    );
    return name || `???`;
  },

  nameToId(name: string): number | undefined {
    return grantConstants[name as keyof typeof grantConstants];
  },

  // Get all grant IDs
  getAllIds(): number[] {
    return Object.values(grantConstants);
  },

  // Get grant list in UI format
  getGrantList(): { id: number; text: string }[] {
    return Object.keys(grantConstants).map((key) => ({
      id: grantConstants[key as keyof typeof grantConstants],
      text: key
        .replace(/_/g, " ")
        .toLowerCase()
        .replace(/\b\w/g, (l) => l.toUpperCase()),
    }));
  },
};
