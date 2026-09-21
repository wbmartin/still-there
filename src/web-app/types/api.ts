// Shapes every API response shares.

/** The envelope every JSON body arrives in. */
export interface ApiResponse<T> {
  data: T;
}

/**
 * Anything the API returns carries these. deleteTs only ever arrives on an
 * incremental read, where the server sends soft-deleted rows as tombstones.
 */
export interface EntityBase {
  updateTs: number;
  deleteTs?: number;
}
