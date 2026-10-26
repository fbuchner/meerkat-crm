import { useState } from 'react';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  Alert,
  Avatar,
  Box,
  Button,
  Card,
  CardActions,
  CardContent,
  Chip,
  CircularProgress,
  Link,
  Stack,
  Typography,
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import CallMergeIcon from '@mui/icons-material/CallMerge';
import { dismissDuplicate, DuplicateContactSummary, DuplicatePair } from './api/contacts';
import MergeContactsDialog, { contactDisplayName } from './components/MergeContactsDialog';
import { useDuplicateContacts } from './hooks/useDuplicateContacts';
import { useSnackbar } from './context/SnackbarContext';
import { getErrorMessage } from './utils/errorHandler';

function ContactSummary({ contact }: { contact: DuplicateContactSummary }) {
  const { t } = useTranslation();
  const details = [contact.email, contact.phone, contact.organization].filter(Boolean);
  return (
    <Stack
      direction="row"
      spacing={1.5}
      sx={{
        alignItems: "center",
        flex: 1,
        minWidth: 0
      }}>
      <Avatar src={contact.photo_thumbnail}>{contact.firstname?.[0]}</Avatar>
      <Box sx={{ minWidth: 0 }}>
        <Stack direction="row" spacing={1} sx={{
          alignItems: "center"
        }}>
          <Link component={RouterLink} to={`/contacts/${contact.ID}`} underline="hover" noWrap>
            {contactDisplayName(contact)}
          </Link>
          {contact.archived && <Chip size="small" label={t('contacts.archived')} />}
        </Stack>
        {details.map((detail) => (
          <Typography key={detail} variant="body2" noWrap sx={{
            color: "text.secondary"
          }}>
            {detail}
          </Typography>
        ))}
      </Box>
    </Stack>
  );
}

export default function DuplicatesPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { showSnackbar, showError } = useSnackbar();
  const { data: pairs, loading, error, refetch } = useDuplicateContacts();
  const [merging, setMerging] = useState<DuplicatePair | null>(null);

  const handleDismiss = async (pair: DuplicatePair) => {
    try {
      await dismissDuplicate(pair.contacts[0].ID, pair.contacts[1].ID);
      await refetch();
    } catch (err) {
      showError(getErrorMessage(err));
    }
  };

  const handleMerged = async () => {
    setMerging(null);
    showSnackbar(t('contacts.merge.success'), 'success');
    await refetch();
  };

  return (
    <Box sx={{ maxWidth: 1200, mx: 'auto', mt: 2, p: 2 }}>
      <Button startIcon={<ArrowBackIcon />} onClick={() => navigate('/contacts')} sx={{ mb: 1 }}>
        {t('contacts.title')}
      </Button>
      <Typography variant="h5" gutterBottom>
        {t('contacts.duplicates.title')}
      </Typography>
      <Typography
        variant="body2"
        sx={{
          color: "text.secondary",
          mb: 2
        }}>
        {t('contacts.duplicates.description')}
      </Typography>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      {loading && pairs.length === 0 ? (
        <Box
          sx={{
            display: "flex",
            justifyContent: "center",
            mt: 4
          }}>
          <CircularProgress />
        </Box>
      ) : pairs.length === 0 && !error ? (
        <Alert severity="success">{t('contacts.duplicates.none')}</Alert>
      ) : (
        <Stack spacing={1.5}>
          {pairs.map((pair) => (
            <Card key={`${pair.contacts[0].ID}-${pair.contacts[1].ID}`} variant="outlined">
              <CardContent sx={{ pb: 1 }}>
                <Stack direction="row" spacing={0.5} sx={{ mb: 1.5 }}>
                  {pair.reasons.map((reason) => (
                    <Chip key={reason} size="small" color="primary" variant="outlined" label={t(`contacts.duplicates.reason.${reason}`)} />
                  ))}
                </Stack>
                <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
                  <ContactSummary contact={pair.contacts[0]} />
                  <ContactSummary contact={pair.contacts[1]} />
                </Stack>
              </CardContent>
              <CardActions sx={{ justifyContent: 'flex-end' }}>
                <Button size="small" onClick={() => handleDismiss(pair)}>
                  {t('contacts.duplicates.dismiss')}
                </Button>
                <Button size="small" variant="contained" startIcon={<CallMergeIcon />} onClick={() => setMerging(pair)}>
                  {t('contacts.merge.button')}
                </Button>
              </CardActions>
            </Card>
          ))}
        </Stack>
      )}

      {merging && (
        <MergeContactsDialog
          open
          onClose={() => setMerging(null)}
          contactId={merging.contacts[0].ID}
          otherContactId={merging.contacts[1].ID}
          onMerged={handleMerged}
        />
      )}
    </Box>
  );
}
