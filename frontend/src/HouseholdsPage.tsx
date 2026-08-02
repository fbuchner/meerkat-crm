import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Box, Typography, Button, Alert, LinearProgress } from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import HouseholdDialog, { HouseholdFormData } from './components/HouseholdDialog';
import HouseholdList from './components/HouseholdList';
import { useHouseholds } from './hooks/useHouseholds';
import { useSnackbar } from './context/SnackbarContext';
import { Household, HouseholdMember } from './api/households';
import { getContactsByUid, Contact } from './api/contacts';
import { handleFetchError } from './utils/errorHandler';

export default function HouseholdsPage() {
  const { t } = useTranslation();
  const { showError, showSuccess, showInfo } = useSnackbar();

  const {
    households,
    members,
    loading,
    error,
    refresh,
    handleCreate,
    handleUpdate,
    handleDelete,
    handleAddMember,
    handleRemoveMember,
    handleSuggestRelationships,
  } = useHouseholds({ showError });

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingHousehold, setEditingHousehold] = useState<Household | null>(null);
  const [suggestPendingId, setSuggestPendingId] = useState<string | null>(null);

  // Resolve member VCardUIDs to contact names for display. Re-resolved on
  // every membership change (the async result replaces the map wholesale).
  const [contactsByUid, setContactsByUid] = useState<Map<string, Contact>>(new Map());
  useEffect(() => {
    const uids = members.map((m) => m.member_vcard_uid);
    getContactsByUid(uids)
      .then(setContactsByUid)
      .catch((err) => {
        handleFetchError(err, 'resolving household members');
        setContactsByUid(new Map());
      });
  }, [members]);

  const handleOpenCreate = () => {
    setEditingHousehold(null);
    setDialogOpen(true);
  };

  const handleOpenEdit = (household: Household) => {
    setEditingHousehold(household);
    setDialogOpen(true);
  };

  const handleSave = async (data: HouseholdFormData) => {
    if (editingHousehold) {
      await handleUpdate(editingHousehold.id, data);
      showSuccess(t('household.updated'));
    } else {
      await handleCreate(data);
      showSuccess(t('household.created'));
    }
  };

  const handleConfirmDelete = async (id: string) => {
    if (!window.confirm(t('household.deleteMessage'))) return;
    try {
      await handleDelete(id);
      showSuccess(t('household.deleted'));
    } catch {
      // Error already surfaced by the hook.
    }
  };

  const handleSuggest = async (household: Household) => {
    setSuggestPendingId(household.id);
    try {
      const count = await handleSuggestRelationships(household.id);
      if (count > 0) {
        showSuccess(t('household.suggestionsGenerated', { count }));
      } else {
        showInfo(t('household.noNewSuggestions'));
      }
    } catch {
      // Error already surfaced by the hook.
    } finally {
      setSuggestPendingId(null);
    }
  };

  // Changing a member's role has no dedicated endpoint — the role is set at
  // add time, so editing means remove + re-add with the new role.
  const handleRoleChange = async (householdId: string, member: HouseholdMember, role: string) => {
    try {
      await handleRemoveMember(householdId, member.member_vcard_uid);
      await handleAddMember(householdId, member.member_vcard_uid, role || undefined);
    } catch {
      await refresh();
    }
  };

  return (
    <Box sx={{ maxWidth: 960, mx: 'auto', mt: 2, p: 2 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
        <Typography variant="h5">{t('household.title')}</Typography>
        <Button variant="contained" startIcon={<AddIcon />} onClick={handleOpenCreate}>
          {t('household.newHousehold')}
        </Button>
      </Box>
      <Typography variant="body2" color="text.secondary" paragraph>
        {t('household.description')}
      </Typography>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      {loading && <LinearProgress sx={{ mb: 2 }} />}

      <HouseholdList
        households={households}
        members={members}
        contactsByUid={contactsByUid}
        suggestPendingId={suggestPendingId}
        onEdit={handleOpenEdit}
        onDelete={handleConfirmDelete}
        onSuggest={handleSuggest}
        onAddMember={(householdId, uid, role) => {
          handleAddMember(householdId, uid, role).catch(() => {});
        }}
        onRemoveMember={(householdId, uid) => {
          handleRemoveMember(householdId, uid).catch(() => {});
        }}
        onRoleChange={handleRoleChange}
      />

      <HouseholdDialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
        onSave={handleSave}
        household={editingHousehold}
      />
    </Box>
  );
}
