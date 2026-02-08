// Custom hook for fetching network graph data
import { useState, useEffect, useCallback, useRef } from 'react';
import { isAuthenticated } from '../auth';
import { getGraph } from '../api/graph';
import { GraphData } from '../types/graph';
import { handleFetchError } from '../utils/errorHandler';

interface UseGraphResult {
  data: GraphData | null;
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

export function useGraph(): UseGraphResult {
  const [data, setData] = useState<GraphData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const requestRef = useRef(0);

  const fetchGraph = useCallback(async () => {
    const requestId = requestRef.current + 1;
    requestRef.current = requestId;
    setLoading(true);
    setError(null);

    try {
      if (!isAuthenticated()) {
        throw new Error('No authentication token found');
      }

      const response = await getGraph();

      if (requestRef.current !== requestId) {
        return;
      }

      setData({
        nodes: response.nodes || [],
        edges: response.edges || [],
      });
    } catch (err) {
      if (requestRef.current !== requestId) {
        return;
      }
      const message = handleFetchError(err, 'fetching network graph');
      setError(message);
    } finally {
      if (requestRef.current === requestId) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    fetchGraph();
  }, [fetchGraph]);

  return {
    data,
    loading,
    error,
    refetch: fetchGraph,
  };
}
