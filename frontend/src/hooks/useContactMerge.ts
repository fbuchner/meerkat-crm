import { useState, useCallback, useMemo } from 'react';
import {
  previewContactMerge,
  commitContactMerge,
  ContactMergePreviewResponse,
} from '../api/contactMerge';
import { handleFetchError, handleError, ErrorNotifier } from '../utils/errorHandler';

// useContactMerge drives the preview -> resolve conflicts -> commit flow for
// ticket N1. One preview at a time (no list concept, unlike
// useRelationshipEdges) -- the caller supplies keepId/mergeId per call.
export function useContactMerge(notifier?: ErrorNotifier) {
  const [preview, setPreview] = useState<ContactMergePreviewResponse | null>(null);
  const [resolutions, setResolutions] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);
  const [committing, setCommitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadPreview = useCallback(async (keepId: number, mergeId: number) => {
    setLoading(true);
    setError(null);
    try {
      const result = await previewContactMerge(keepId, mergeId);
      // Backend sends a bare `null` (not `[]`) for an empty Go slice --
      // same convention as GetRelationshipEdges, normalized once here
      // rather than guarding every render-site access.
      setPreview({
        ...result,
        resolution: {
          ...result.resolution,
          emails: result.resolution.emails || [],
          phones: result.resolution.phones || [],
          addresses: result.resolution.addresses || [],
          urls: result.resolution.urls || [],
          impps: result.resolution.impps || [],
          custom_fields: result.resolution.custom_fields || {},
          resolved_scalars: result.resolution.resolved_scalars || {},
          conflicts: result.resolution.conflicts || [],
          field_value_conflicts: result.resolution.field_value_conflicts || [],
        },
      });
      setResolutions({});
    } catch (err) {
      setError(handleFetchError(err, 'previewing contact merge'));
      setPreview(null);
    } finally {
      setLoading(false);
    }
  }, []);

  const setResolution = useCallback((field: string, value: string) => {
    setResolutions((prev) => ({ ...prev, [field]: value }));
  }, []);

  // allConflictsResolved covers both conflict kinds (scalar + FieldValue) --
  // the commit button should stay disabled until every one has a choice, so
  // the user never discovers a missing resolution only after the 400 comes
  // back.
  const allConflictsResolved = useMemo(() => {
    if (!preview) return false;
    const all = [...preview.resolution.conflicts, ...preview.resolution.field_value_conflicts];
    return all.every((c) => !!resolutions[c.field]);
  }, [preview, resolutions]);

  const commit = useCallback(
    async (keepId: number, mergeId: number) => {
      setCommitting(true);
      try {
        return await commitContactMerge(keepId, mergeId, resolutions);
      } catch (err) {
        handleError(err, { operation: 'merging contacts' }, notifier);
        throw err;
      } finally {
        setCommitting(false);
      }
    },
    [resolutions, notifier]
  );

  const reset = useCallback(() => {
    setPreview(null);
    setResolutions({});
    setError(null);
  }, []);

  return {
    preview,
    resolutions,
    loading,
    committing,
    error,
    allConflictsResolved,
    loadPreview,
    setResolution,
    commit,
    reset,
  };
}
