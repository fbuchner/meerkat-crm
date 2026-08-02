import { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Typography,
  Paper,
  Chip,
  IconButton,
  Button,
  Stack,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Autocomplete,
  TextField,
  CircularProgress,
  Tooltip,
} from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import PersonAddIcon from '@mui/icons-material/PersonAdd';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import { useTranslation } from 'react-i18next';
import { Household, HouseholdMember, HouseholdRole, HOUSEHOLD_ROLES } from '../api/households';
import { Contact, getContacts } from '../api/contacts';
import { handleFetchError } from '../utils/errorHandler';

interface ContactBrief {
  uid: string;
  firstname: string;
  lastname: string;
}

function toContactBrief(c: Contact): ContactBrief | null {
  if (!c.uid) return null;
  return { uid: c.uid, firstname: c.firstname, lastname: c.lastname };
}

function contactLabel(c: ContactBrief): string {
  return `${c.firstname} ${c.lastname}`.trim();
}

// ContactAutocomplete is a debounced server-side contact search used by the
// add-member row, mirroring LifeEventDialog's autocomplete idiom.
function ContactAutocomplete({
  excludeUids,
  onSelect,
}: {
  excludeUids: Set<string>;
  onSelect: (contact: ContactBrief) => void;
}) {
  const { t } = useTranslation();
  const [contacts, setContacts] = useState<ContactBrief[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchInput, setSearchInput] = useState('');
  const [value, setValue] = useState<ContactBrief | null>(null);

  useEffect(() => {
    if (value) {
      onSelect(value);
      setValue(null);
    }
  }, [value, onSelect]);

  useEffect(() => {
    const timer = setTimeout(async () => {
      setLoading(true);
      try {
        const response = await getContacts({ limit: 40, search: searchInput });
        setContacts(
          (response.contacts || [])
            .map(toContactBrief)
            .filter((c): c is ContactBrief => c !== null && !excludeUids.has(c.uid))
        );
      } catch (err) {
        handleFetchError(err, 'searching contacts');
        setContacts([]);
      } finally {
        setLoading(false);
      }
    }, 300);
    return () => clearTimeout(timer);
  }, [searchInput, excludeUids]);

  return (
    <Autocomplete
      size="small"
      options={contacts}
      getOptionLabel={contactLabel}
      value={value}
      onChange={(_, newValue) => setValue(newValue)}
      onInputChange={(_, newValue) => setSearchInput(newValue)}
      loading={loading}
      filterOptions={(x) => x}
      renderInput={(params) => (
        <TextField
          {...params}
          placeholder={t('household.memberSearch')}
          size="small"
          fullWidth
          InputProps={{
            ...params.InputProps,
            endAdornment: (
              <>
                {loading ? <CircularProgress color="inherit" size={18} /> : null}
                {params.InputProps.endAdornment}
              </>
            ),
          }}
        />
      )}
      isOptionEqualToValue={(option, selected) => option.uid === selected.uid}
      noOptionsText={searchInput ? t('household.noContactsFound') : t('household.typeToSearch')}
    />
  );
}

interface HouseholdListProps {
  households: Household[];
  members: HouseholdMember[];
  contactsByUid: Map<string, Contact>;
  suggestPendingId: string | null;
  onEdit: (household: Household) => void;
  onDelete: (id: string) => void;
  onSuggest: (household: Household) => void;
  onAddMember: (householdId: string, vcardUid: string, role?: string) => void;
  onRemoveMember: (householdId: string, vcardUid: string) => void;
  onRoleChange: (householdId: string, member: HouseholdMember, role: string) => void;
}

export default function HouseholdList({
  households,
  members,
  contactsByUid,
  suggestPendingId,
  onEdit,
  onDelete,
  onSuggest,
  onAddMember,
  onRemoveMember,
  onRoleChange,
}: HouseholdListProps) {
  const { t } = useTranslation();
  const [addOpenFor, setAddOpenFor] = useState<string | null>(null);
  const [newMemberRole, setNewMemberRole] = useState<HouseholdRole>('head');

  const membersFor = useCallback(
    (householdId: string): HouseholdMember[] =>
      members.filter((m) => m.household_id === householdId),
    [members]
  );

  const memberName = (member: HouseholdMember): string => {
    const contact = contactsByUid.get(member.member_vcard_uid);
    return contact ? `${contact.firstname} ${contact.lastname}`.trim() : member.member_vcard_uid;
  };

  if (households.length === 0) {
    return (
      <Paper sx={{ p: 3, textAlign: 'center' }}>
        <Typography variant="body1" color="text.secondary">
          {t('household.empty')}
        </Typography>
      </Paper>
    );
  }

  return (
    <Stack spacing={2}>
      {households.map((household) => {
        const householdMembers = membersFor(household.id);
        const excludeUids = new Set(householdMembers.map((m) => m.member_vcard_uid));
        const isSuggestPending = suggestPendingId === household.id;
        const isAddOpen = addOpenFor === household.id;
        return (
          <Paper key={household.id} variant="outlined" sx={{ p: 2 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
              <Box>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Typography variant="h6" component="span">
                    {household.name}
                  </Typography>
                  <Chip
                    label={t(`household.types.${household.type}`, household.type)}
                    size="small"
                    variant="outlined"
                  />
                </Box>
                <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                  {t('household.members', { count: householdMembers.length })}
                </Typography>
              </Box>
              <Box sx={{ display: 'flex', gap: 0.5 }}>
                <Tooltip title={t('household.suggestRelationships')}>
                  <span>
                    <IconButton
                      size="small"
                      color="primary"
                      disabled={isSuggestPending || householdMembers.length < 2}
                      onClick={() => onSuggest(household)}
                      aria-label={t('household.suggestRelationships')}
                    >
                      {isSuggestPending ? (
                        <CircularProgress size={18} />
                      ) : (
                        <AutoAwesomeIcon fontSize="small" />
                      )}
                    </IconButton>
                  </span>
                </Tooltip>
                <IconButton
                  size="small"
                  onClick={() => onEdit(household)}
                  aria-label={t('common.edit')}
                >
                  <EditIcon fontSize="small" />
                </IconButton>
                <IconButton
                  size="small"
                  color="error"
                  onClick={() => onDelete(household.id)}
                  aria-label={t('common.delete')}
                >
                  <DeleteIcon fontSize="small" />
                </IconButton>
              </Box>
            </Box>

            <Box sx={{ mt: 1.5, display: 'flex', flexDirection: 'column', gap: 1 }}>
              {householdMembers.length === 0 && (
                <Typography variant="body2" color="text.secondary">
                  {t('household.noMembers')}
                </Typography>
              )}
              {householdMembers.map((member) => (
                <Box
                  key={member.id}
                  sx={{ display: 'flex', alignItems: 'center', gap: 1 }}
                >
                  <Typography variant="body2" sx={{ flex: 1 }}>
                    {memberName(member)}
                  </Typography>
                  <FormControl size="small" sx={{ minWidth: 120 }}>
                    <InputLabel id={`role-label-${member.id}`}>{t('household.role')}</InputLabel>
                    <Select
                      size="small"
                      labelId={`role-label-${member.id}`}
                      label={t('household.role')}
                      value={member.role || ''}
                      onChange={(e) => onRoleChange(household.id, member, e.target.value as string)}
                    >
                      <MenuItem value="">{t('household.noRole')}</MenuItem>
                      {HOUSEHOLD_ROLES.map((role) => (
                        <MenuItem key={role} value={role}>
                          {t(`household.roles.${role}`, role)}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                  <IconButton
                    size="small"
                    color="error"
                    onClick={() => onRemoveMember(household.id, member.member_vcard_uid)}
                    aria-label={t('household.removeMember')}
                  >
                    <DeleteIcon fontSize="small" />
                  </IconButton>
                </Box>
              ))}
            </Box>

            <Box sx={{ mt: 2 }}>
              {isAddOpen ? (
                <Box sx={{ display: 'flex', gap: 1, alignItems: 'flex-start' }}>
                  <Box sx={{ flex: 1 }}>
                    <ContactAutocomplete
                      excludeUids={excludeUids}
                      onSelect={(c) => {
                        setAddOpenFor(null);
                        onAddMember(household.id, c.uid, newMemberRole);
                      }}
                    />
                  </Box>
                  <FormControl size="small" sx={{ minWidth: 120 }}>
                    <InputLabel>{t('household.role')}</InputLabel>
                    <Select
                      size="small"
                      label={t('household.role')}
                      value={newMemberRole}
                      onChange={(e) => setNewMemberRole(e.target.value as HouseholdRole)}
                    >
                      {HOUSEHOLD_ROLES.map((role) => (
                        <MenuItem key={role} value={role}>
                          {t(`household.roles.${role}`, role)}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                  <Button
                    size="small"
                    onClick={() => setAddOpenFor(null)}
                    disabled={isSuggestPending}
                  >
                    {t('household.cancel')}
                  </Button>
                </Box>
              ) : (
                <Button
                  size="small"
                  startIcon={<PersonAddIcon />}
                  onClick={() => setAddOpenFor(household.id)}
                >
                  {t('household.addMember')}
                </Button>
              )}
            </Box>
          </Paper>
        );
      })}
    </Stack>
  );
}
