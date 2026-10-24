// Grant ID constants. Mirrors src/api-srv/internal/model/grants.go by hand;
// when a grant is added there, run bin/305-update-grants-fixture and add it
// here. tests/contract.test.ts fails while the two disagree. Numbering
// convention for new ids (never renumber shipped ones): see the header of
// grants.go and doc/810.
const grantConstants = {
  LOGON: 1,
  READ_EXAMPLE: 1001,
  CREATE_EXAMPLE: 1002,
  UPDATE_EXAMPLE: 1003,
  DELETE_EXAMPLE: 1004,
  READ_DEVICE: 1010,
  CREATE_DEVICE: 1011,
  UPDATE_DEVICE: 1012,
  DELETE_DEVICE: 1013,
  ISSUE_DEVICE_TOKEN: 1014,
  READ_BEACON: 1020,
  CREATE_BEACON: 1021,
  UPDATE_BEACON: 1022,
  DELETE_BEACON: 1023,
  READ_CONTACT: 1030,
  CREATE_CONTACT: 1031,
  UPDATE_CONTACT: 1032,
  DELETE_CONTACT: 1033,
  READ_ALERT: 1040,
  UPDATE_ALERT: 1042,
  READ_TENANT_SETTING: 1050,
  UPDATE_TENANT_SETTING: 1052,
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
  RESET_MFA_TOKEN: 9020,
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
