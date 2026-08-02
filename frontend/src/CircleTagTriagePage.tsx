import { useState, useCallback, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box,
  Typography,
  Paper,
  Button,
  Stepper,
  Step,
  StepLabel,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  FormControl,
  Select,
  MenuItem,
  Chip,
  LinearProgress,
  Alert,
} from '@mui/material';
import { getCircles, getContacts } from './api/contacts';
import { createCircle, addCircleMember, Circle, listCircles } from './api/circles';
import { createTag, addContactTag, Tag, listTags } from './api/tags';
import { handleFetchError } from './utils/errorHandler';

type Classification = 'circle' | 'tag' | 'skip';

interface TriagedItem {
  original: string;
  name: string;
  classification: Classification;
  contactCount: number;
}

export default function CircleTagTriagePage() {
  const { t } = useTranslation();
  const [activeStep, setActiveStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [items, setItems] = useState<TriagedItem[]>([]);
  const [applying, setApplying] = useState(false);
  const [applyProgress, setApplyProgress] = useState({ current: 0, total: 0 });
  const [applyResult, setApplyResult] = useState<string[]>([]);

  const steps = [
    t('triage.stepClassify'),
    t('triage.stepPreview'),
    t('triage.stepApply'),
  ];

  // Step 0: Collect distinct strings from legacy circles
  const collect = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const allCircles = await getCircles();
      const distinct = Array.isArray(allCircles) ? allCircles : [];

      if (distinct.length === 0) {
        setItems([]);
        setLoading(false);
        return;
      }

      const triaged: TriagedItem[] = [];
      for (const name of distinct) {
        try {
          const resp = await getContacts({ circle: name, limit: 200 });
          triaged.push({
            original: name,
            name,
            classification: 'circle' as Classification,
            contactCount: resp.total ?? (resp.contacts?.length ?? 0),
          });
        } catch {
          triaged.push({ original: name, name, classification: 'circle' as Classification, contactCount: 0 });
        }
      }
      // Sort by contact count descending so the user sees the biggest groups first.
      triaged.sort((a, b) => b.contactCount - a.contactCount);
      setItems(triaged);
    } catch (err) {
      setError(handleFetchError(err, 'collecting legacy circles'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    collect();
  }, [collect]);

  const handleClassificationChange = (index: number, classification: Classification) => {
    setItems((prev) => {
      const next = [...prev];
      next[index] = { ...next[index], classification };
      return next;
    });
  };

  const handleNameChange = (index: number, name: string) => {
    setItems((prev) => {
      const next = [...prev];
      next[index] = { ...next[index], name };
      return next;
    });
  };

  const circles = items.filter((i) => i.classification === 'circle');
  const tags = items.filter((i) => i.classification === 'tag');
  const skipped = items.filter((i) => i.classification === 'skip');

  const handleApply = async () => {
    setApplying(true);
    setApplyProgress({ current: 0, total: circles.length + tags.length });
    setApplyResult([]);

    const results: string[] = [];

    // Create circles
    for (const item of circles) {
      try {
        await createCircle(item.name);
        results.push(`Created Circle: ${item.name}`);
      } catch (err: any) {
        // 409 on duplicate = success
        if (err?.response?.status === 409 || err?.code === 409 || err?.status === 409) {
          results.push(`Circle "${item.name}" already exists, skipping`);
        } else {
          results.push(`Failed to create circle "${item.name}": ${err}`);
        }
      }
      setApplyProgress((p) => ({ ...p, current: p.current + 1 }));
    }

    // Create tags
    for (const item of tags) {
      try {
        await createTag(item.name);
        results.push(`Created Tag: ${item.name}`);
      } catch (err: any) {
        if (err?.response?.status === 409 || err?.code === 409 || err?.status === 409) {
          results.push(`Tag "${item.name}" already exists, skipping`);
        } else {
          results.push(`Failed to create tag "${item.name}": ${err}`);
        }
      }
      setApplyProgress((p) => ({ ...p, current: p.current + 1 }));
    }

    setApplyResult(results);
    setApplying(false);
  };

  const handleAddMembers = async () => {
    setApplying(true);
    const memeTotal = circles.length + tags.length;
    setApplyProgress({ current: 0, total: memeTotal });
    const results: string[] = [];

    // Resolve existing circle and tag IDs via list endpoints
    let allCircles: Circle[] = [];
    let allTags: Tag[] = [];
    try {
      const [cr, tr] = await Promise.all([
        listCircles({ limit: 200 }),
        listTags({ limit: 200 }),
      ]);
      allCircles = cr.circles || [];
      allTags = tr.tags || [];
    } catch (err) {
      results.push(`Failed to fetch circles/tags: ${err}`);
      setApplyResult(results);
      setApplying(false);
      return;
    }

    for (const item of items.filter((i) => i.classification === 'circle')) {
      const entity = allCircles.find((c) => c.name === item.name);
      if (!entity) {
        results.push(`Circle "${item.name}" not found — create it first`);
        setApplyProgress((p) => ({ ...p, current: p.current + 1 }));
        continue;
      }

      try {
        const resp = await getContacts({ circle: item.original, limit: 500 });
        const contacts = resp.contacts || [];
        for (const c of contacts) {
          if (!c.uid) continue;
          try {
            await addCircleMember(entity.id, c.uid);
          } catch (e: any) {
            if (e?.status === 409 || e?.response?.status === 409) continue;
            results.push(`Failed to add ${c.firstname} ${c.lastname} to circle "${item.name}": ${e}`);
          }
        }
        results.push(`Added ${contacts.filter((c) => c.uid).length} members to Circle "${item.name}"`);
      } catch (err) {
        results.push(`Failed to query contacts for circle "${item.name}": ${err}`);
      }
      setApplyProgress((p) => ({ ...p, current: p.current + 1 }));
    }

    for (const item of items.filter((i) => i.classification === 'tag')) {
      const entity = allTags.find((t) => t.name === item.name);
      if (!entity) {
        results.push(`Tag "${item.name}" not found — create it first`);
        setApplyProgress((p) => ({ ...p, current: p.current + 1 }));
        continue;
      }

      try {
        const resp = await getContacts({ circle: item.original, limit: 500 });
        const contacts = resp.contacts || [];
        for (const c of contacts) {
          if (!c.uid) continue;
          try {
            await addContactTag(entity.id, c.uid);
          } catch {
            // 409 = already tagged, OK
          }
        }
        results.push(`Tagged ${contacts.filter((c) => c.uid).length} contacts with "${item.name}"`);
      } catch (err) {
        results.push(`Failed to query contacts for tag "${item.name}": ${err}`);
      }
      setApplyProgress((p) => ({ ...p, current: p.current + 1 }));
    }

    setApplyResult((prev) => [...prev, ...results]);
    setApplying(false);
  };

  return (
    <Box sx={{ maxWidth: 960, mx: 'auto', mt: 2, p: 2 }}>
      <Typography variant="h5" gutterBottom>
        {t('triage.title')}
      </Typography>
      <Typography variant="body2" color="text.secondary" paragraph>
        {t('triage.description')}
      </Typography>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Stepper activeStep={activeStep} sx={{ mb: 3 }}>
        {steps.map((label) => (
          <Step key={label}><StepLabel>{label}</StepLabel></Step>
        ))}
      </Stepper>

      {loading && <LinearProgress sx={{ mb: 2 }} />}

      {/* Step 0: Classify */}
      {activeStep === 0 && !loading && (
        <>
          {items.length === 0 ? (
            <Paper sx={{ p: 3, textAlign: 'center' }}>
              <Typography color="text.secondary">{t('triage.noLegacyCircles')}</Typography>
              <Button variant="outlined" sx={{ mt: 2 }} onClick={collect}>
                {t('triage.refresh')}
              </Button>
            </Paper>
          ) : (
            <>
              <TableContainer component={Paper}>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>{t('triage.colCircleName')}</TableCell>
                      <TableCell width={100}>{t('triage.colContactCount')}</TableCell>
                      <TableCell width={140}>{t('triage.colClassification')}</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {items.map((item, index) => (
                      <TableRow key={item.original}>
                        <TableCell>
                          <TextField
                            size="small"
                            value={item.name}
                            onChange={(e) => handleNameChange(index, e.target.value)}
                            fullWidth
                            variant="standard"
                          />
                        </TableCell>
                        <TableCell>
                          <Chip label={item.contactCount} size="small" />
                        </TableCell>
                        <TableCell>
                          <FormControl size="small" fullWidth>
                            <Select
                              value={item.classification}
                              onChange={(e) => handleClassificationChange(index, e.target.value as Classification)}
                            >
                              <MenuItem value="circle">{t('triage.classificationCircle')}</MenuItem>
                              <MenuItem value="tag">{t('triage.classificationTag')}</MenuItem>
                              <MenuItem value="skip">{t('triage.classificationSkip')}</MenuItem>
                            </Select>
                          </FormControl>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
              <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>
                {t('triage.summary', { total: items.length, circles: circles.length, tags: tags.length, skipped: skipped.length })}
              </Typography>
              <Box display="flex" justifyContent="flex-end" mt={2}>
                <Button variant="contained" onClick={() => setActiveStep(1)}>
                  {t('triage.next')}
                </Button>
              </Box>
            </>
          )}
        </>
      )}

      {/* Step 1: Preview */}
      {activeStep === 1 && (
        <Paper sx={{ p: 2 }}>
          <Typography variant="h6" gutterBottom>{t('triage.previewTitle')}</Typography>

          {circles.length > 0 && (
            <Box mb={2}>
              <Typography variant="subtitle1">
                {t('triage.circlesToCreate', { count: circles.length })}
              </Typography>
              <Box display="flex" flexWrap="wrap" gap={0.5} mt={0.5}>
                {circles.map((c) => (
                  <Chip key={c.name} label={`${c.name} (${c.contactCount})`} color="primary" size="small" />
                ))}
              </Box>
            </Box>
          )}

          {tags.length > 0 && (
            <Box mb={2}>
              <Typography variant="subtitle1">
                {t('triage.tagsToCreate', { count: tags.length })}
              </Typography>
              <Box display="flex" flexWrap="wrap" gap={0.5} mt={0.5}>
                {tags.map((t) => (
                  <Chip key={t.name} label={`${t.name} (${t.contactCount})`} color="secondary" size="small" />
                ))}
              </Box>
            </Box>
          )}

          {skipped.length > 0 && (
            <Box mb={2}>
              <Typography variant="subtitle1">
                {t('triage.skipped', { count: skipped.length })}
              </Typography>
              <Box display="flex" flexWrap="wrap" gap={0.5} mt={0.5}>
                {skipped.map((s) => (
                  <Chip key={s.name} label={s.name} variant="outlined" size="small" />
                ))}
              </Box>
            </Box>
          )}

          <Box display="flex" justifyContent="space-between" mt={2}>
            <Button onClick={() => setActiveStep(0)}>{t('triage.back')}</Button>
            <Button variant="contained" color="success" onClick={() => setActiveStep(2)} disabled={circles.length === 0 && tags.length === 0}>
              {t('triage.apply')}
            </Button>
          </Box>
        </Paper>
      )}

      {/* Step 2: Apply */}
      {activeStep === 2 && (
        <Paper sx={{ p: 2 }}>
          <Typography variant="h6" gutterBottom>{t('triage.applyTitle')}</Typography>

          {applying && (
            <Box mb={2}>
              <LinearProgress variant="determinate" value={(applyProgress.current / Math.max(applyProgress.total, 1)) * 100} />
              <Typography variant="body2" color="text.secondary" mt={0.5}>
                {applyProgress.current} / {applyProgress.total}
              </Typography>
            </Box>
          )}

          {!applying && applyResult.length === 0 && (
            <Box textAlign="center" py={3}>
              <Typography color="text.secondary" paragraph>
                {t('triage.applyDescription')}
              </Typography>
              <Box display="flex" gap={2} justifyContent="center">
                <Button variant="contained" onClick={handleApply}>
                  {t('triage.createEntities')}
                </Button>
                <Button variant="contained" onClick={handleAddMembers}>
                  {t('triage.addMembers')}
                </Button>
              </Box>
            </Box>
          )}

          {applyResult.length > 0 && (
            <Paper variant="outlined" sx={{ p: 1.5, maxHeight: 300, overflow: 'auto' }}>
              {applyResult.map((r, i) => (
                <Typography key={i} variant="body2" sx={{ fontFamily: 'monospace', fontSize: '0.85rem' }}>
                  {r}
                </Typography>
              ))}
            </Paper>
          )}
        </Paper>
      )}
    </Box>
  );
}
