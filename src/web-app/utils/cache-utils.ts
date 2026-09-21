// Cache invalidation on sign-out and security-context switches.
//
// Stores register themselves here (createEntityStore does it automatically),
// so adding an entity does not mean editing this file -- and forgetting to
// register cannot leave another tenant's rows in memory after a context
// switch.
type CacheClearer = () => void;

const cacheClearerList: CacheClearer[] = [];

export const registerCacheClearer = (clearer: CacheClearer): void => {
  cacheClearerList.push(clearer);
};

export const clearAllCaches = (): void => {
  cacheClearerList.forEach((clear) => clear());
};
