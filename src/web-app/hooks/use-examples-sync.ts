import { useEffect } from 'react';
import { useExamplesStore } from '../stores/examples-store';

export const useExamplesSync = () => {
  const syncUpdates = useExamplesStore(state => state.syncUpdates);

  useEffect(() => {
    const interval = setInterval(() => {
      syncUpdates();
    }, 120000); // 2 minutes

    return () => clearInterval(interval);
  }, [syncUpdates]);
};