import { useEffect, useMemo, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  Card as CardModel,
  CRMEnvelope,
  NameComponent,
  ContactRecordResponse,
  getContactRecord,
  updateContactRecord,
  nameComponentValue,
  withAnniversary,
  getOrganizationFields,
  withOrganization,
  getTitleField,
  withTitles,
  getContactProfilePicture,
  deleteContact,
  uploadProfilePicture,
  archiveContact,
  unarchiveContact
} from './api/contacts';
import { getCurrentUser } from './api/admin';
import { resolveEnabledFields, ContactFieldKey } from './contactFields';
import { 
  getContactNotes, 
  Note 
} from './api/notes';
import {
  getContactActivities,
  Activity
} from './api/activities';
import {
  ReminderCompletion,
  getCompletionsForContact,
  deleteCompletion
} from './api/reminders';
import {
  Box,
  Card,
  CardContent,
  Divider,
  Button,
  Tabs,
  Tab,
  Typography
} from '@mui/material';
import { ContactDetailHeaderSkeleton, TimelineSkeleton } from './components/LoadingSkeletons';
import NoteIcon from '@mui/icons-material/Note';
import EventIcon from '@mui/icons-material/Event';
import NotificationsActiveIcon from '@mui/icons-material/NotificationsActive';
import AddNoteDialog from './components/AddNoteDialog';
import AddActivityDialog from './components/AddActivityDialog';
import ReminderDialog from './components/ReminderDialog';
import ReminderList from './components/ReminderList';
import EditTimelineItemDialog from './components/EditTimelineItemDialog';
import ContactHeader from './components/ContactHeader';
import MergeContactsDialog from './components/MergeContactsDialog';
import ContactInformation from './components/ContactInformation';
import ContactTimeline from './components/ContactTimeline';
import ProfilePictureUploadDialog from './components/ProfilePictureUploadDialog';
import { useContactDialogs } from './hooks/useContactDialogs';
import { useTimelineEditing } from './hooks/useTimelineEditing';
import { useReminderManagement } from './hooks/useReminderManagement';
import { useRelationshipEdges } from './hooks/useRelationshipEdges';
import { useLifeEvents } from './hooks/useLifeEvents';
import { usePreferences } from './hooks/usePreferences';
import { useCircles } from './hooks/useCircles';
import { useTags } from './hooks/useTags';
import { addCircleMember, removeCircleMember } from './api/circles';
import { addContactTag, removeContactTag } from './api/tags';
import { Circle } from './api/circles';
import { Tag } from './api/tags';
import RelationshipEdgeDialog from './components/RelationshipEdgeDialog';
import LifeEventDialog from './components/LifeEventDialog';
import PreferenceDialog, { toPreferenceInput, PreferenceFormData } from './components/PreferenceDialog';
import { Preference } from './api/preferences';
import { LifeEventFormData } from './components/LifeEventDialog';
import { getOtherPartyId } from './api/relationshipEdges';
import { LifeEvent } from './api/lifeEvents';
import { PartialDate } from './api/lifeEvents';
import { useSnackbar } from './context/SnackbarContext';
import { ApiError } from './api/client';
import { handleFetchError } from './utils/errorHandler';
import { useDateFormat } from './DateFormatProvider';

function fullDateFromPartial(d: PartialDate): string | undefined {
  if (d.year != null && d.month != null && d.day != null) {
    return `${d.year}-${String(d.month).padStart(2, '0')}-${String(d.day).padStart(2, '0')}`;
  }
  if (d.month != null && d.day != null) {
    const y = new Date().getFullYear();
    return `${y}-${String(d.month).padStart(2, '0')}-${String(d.day).padStart(2, '0')}`;
  }
  if (d.year != null) {
    return `${d.year}-01-01`;
  }
  return undefined;
}

export default function ContactDetailPage() {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { showError } = useSnackbar();
  const { formatBirthdayForInput, parseBirthdayInput, autoFormatBirthdayInput } = useDateFormat();
  // record is the single source of truth, fetched/written directly against
  // the nested Card/CRM wire shape -- see docs/fork-plan/95.
  const [record, setRecord] = useState<ContactRecordResponse | null>(null);
  const firstname = record ? nameComponentValue(record.card?.name?.components, 'given') || '' : '';
  const lastname = record ? nameComponentValue(record.card?.name?.components, 'surname') || '' : '';
  const [profilePic, setProfilePic] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [editingField, setEditingField] = useState<string | null>(null);
  const [editValue, setEditValue] = useState<string>('');
  const [validationError, setValidationError] = useState<string>('');
  const [notes, setNotes] = useState<Note[]>([]);
  const [activities, setActivities] = useState<Activity[]>([]);
  const [completions, setCompletions] = useState<ReminderCompletion[]>([]);
  
  // Profile editing state
  const [editingProfile, setEditingProfile] = useState(false);
  const [profileValues, setProfileValues] = useState({
    prefix: '',
    firstname: '',
    middle_name: '',
    lastname: '',
    suffix: '',
    nickname: '',
    gender: ''
  });

  // Circle/Tag state (T4 — real entities instead of flat strings)
  const {
    circles: allCircles,
    circleNamesByUid,
    refresh: refreshCircles,
    handleCreate: handleCreateCircle,
  } = useCircles({ showError });

  const {
    tags: allTags,
    tagNamesByUid,
    refresh: refreshTags,
    handleCreate: handleCreateTag,
  } = useTags({ showError });

  const contactCircles = useMemo(() => {
    if (!record?.uid) return [];
    const names = circleNamesByUid.get(record.uid) || [];
    return allCircles.filter((c) => names.includes(c.name));
  }, [record?.uid, circleNamesByUid, allCircles]);

  const contactTags = useMemo(() => {
    if (!record?.uid) return [];
    const names = tagNamesByUid.get(record.uid) || [];
    return allTags.filter((t) => names.includes(t.name));
  }, [record?.uid, tagNamesByUid, allTags]);

  // Tab state
  const [activeTab, setActiveTab] = useState(0);

  // Profile picture upload state
  const [profilePictureDialogOpen, setProfilePictureDialogOpen] = useState(false);

  // Contact merge dialog state (ticket N1)
  const [mergeDialogOpen, setMergeDialogOpen] = useState(false);

  // Custom field names
  const [customFieldNames, setCustomFieldNames] = useState<string[]>([]);

  // Enabled extended contact fields (UI visibility)
  const [enabledFields, setEnabledFields] = useState<Set<ContactFieldKey>>(() => resolveEnabledFields(null));

  // Unified refresh function for notes, activities, and completions
  const refreshNotesAndActivities = async () => {
    if (!id) return;

    try {
      const [notesData, activitiesData, completionsData] = await Promise.all([
        getContactNotes(id),
        getContactActivities(id),
        getCompletionsForContact(parseInt(id))
      ]);
      setNotes(notesData.notes || []);
      setActivities(activitiesData.activities || []);
      setCompletions(completionsData || []);
    } catch (err) {
      handleFetchError(err, 'refreshing notes and activities');
    }
  };

  // Custom hooks
  const {
    noteDialogOpen,
    activityDialogOpen,
    setNoteDialogOpen,
    setActivityDialogOpen,
    handleSaveNote,
    handleSaveActivity
  } = useContactDialogs(id, refreshNotesAndActivities, { showError });

  const {
    editingTimelineItem,
    editTimelineValues,
    allContacts,
    handleStartEditTimelineItem,
    handleCancelEditTimelineItem,
    handleUpdateNote,
    handleUpdateActivity,
    handleDeleteNote,
    handleDeleteActivity,
    setEditTimelineValues
  } = useTimelineEditing(record?.id, refreshNotesAndActivities, { showError });

  const {
    reminders,
    reminderDialogOpen,
    editingReminder,
    refreshReminders,
    handleSaveReminder,
    handleCompleteReminder,
    handleEditReminder,
    handleDeleteReminder,
    handleAddReminder,
    setReminderDialogOpen,
    setEditingReminder
  } = useReminderManagement(id, { showError });

  // State for pre-filled reminder values (used by Stay in Touch)
  const [reminderInitialValues, setReminderInitialValues] = useState<{
    message?: string;
    recurrence?: 'once' | 'weekly' | 'monthly' | 'quarterly' | 'six-months' | 'yearly';
  } | undefined>(undefined);

  const {
    confirmedEdges,
    suggestedEdges,
    contactsByUid,
    relationshipDialogOpen,
    editingEdge,
    refreshRelationshipEdges,
    handleSaveRelationshipEdge,
    handleEditRelationshipEdge,
    handleDeleteRelationshipEdge,
    handleAcceptSuggestion,
    handleRejectSuggestion,
    handleAddRelationshipEdge,
    setRelationshipDialogOpen,
    setEditingEdge,
  } = useRelationshipEdges(record?.uid, { showError });

  const {
    events: lifeEvents,
    contactsByUid: lifeEventsContactsByUid,
    refresh: refreshLifeEvents,
    handleCreate: handleCreateLifeEvent,
    handleUpdate: handleUpdateLifeEvent,
    handleDelete: handleDeleteLifeEvent,
  } = useLifeEvents(record?.uid);

  const {
    preferences,
    handleSave: handleSavePreference,
    handleDelete: handleDeletePreference,
  } = usePreferences(record?.uid, { showError });

  const [preferenceDialogOpen, setPreferenceDialogOpen] = useState(false);
  const [editingPreference, setEditingPreference] = useState<Preference | null>(null);

  const handleAddPreference = () => {
    setEditingPreference(null);
    setPreferenceDialogOpen(true);
  };

  const handleEditPreference = (pref: Preference) => {
    setEditingPreference(pref);
    setPreferenceDialogOpen(true);
  };

  const handleSavePreferenceSubmit = async (data: PreferenceFormData) => {
    if (!record?.uid) return;
    await handleSavePreference(editingPreference, toPreferenceInput(record.uid, data));
  };

  const handlePreferenceDelete = async (id: string) => {
    if (!window.confirm(t('preference.deleteMessage'))) return;
    await handleDeletePreference(id);
  };

  const [lifeEventDialogOpen, setLifeEventDialogOpen] = useState(false);
  const [editingLifeEvent, setEditingLifeEvent] = useState<LifeEvent | null>(null);

  const handleAddLifeEvent = () => {
    setEditingLifeEvent(null);
    setLifeEventDialogOpen(true);
  };

  const handleEditLifeEvent = (event: LifeEvent) => {
    setEditingLifeEvent(event);
    setLifeEventDialogOpen(true);
  };

  const handleSaveLifeEvent = async (data: LifeEventFormData) => {
    if (!record?.uid) return;
    if (editingLifeEvent) {
      await handleUpdateLifeEvent(editingLifeEvent.id, { ...data, entity_id: record.uid });
    } else {
      await handleCreateLifeEvent({ ...data, entity_id: record.uid, source: 'user' });
    }
  };

  const handleLifeEventDelete = async (id: string) => {
    if (!window.confirm(t('lifeEvent.confirmDelete'))) return;
    await handleDeleteLifeEvent(id);
  };

  const editingEdgeOtherParty = useMemo(() => {
    if (!editingEdge || !record) return undefined;
    return contactsByUid.get(getOtherPartyId(editingEdge, record.uid));
  }, [editingEdge, record, contactsByUid]);

  // Fetch available circles
  const handleCircleAdd = async (circle: Circle) => {
    if (!record?.uid) return;
    try {
      if (circle.id) {
        await addCircleMember(circle.id, record.uid);
      } else {
        const created = await handleCreateCircle(circle.name);
        if (created?.id) await addCircleMember(created.id, record.uid);
      }
      await refreshCircles();
    } catch {
      // Error already reported by hook's handleCreateCircle or
      // addCircleMember — just refresh to reconcile state.
      await refreshCircles();
    }
  };

  const handleCircleRemove = async (circle: Circle) => {
    if (!record?.uid) return;
    try {
      await removeCircleMember(circle.id, record.uid);
      await refreshCircles();
    } catch {
      await refreshCircles();
    }
  };

  const handleTagAdd = async (tag: Tag) => {
    if (!record?.uid) return;
    try {
      if (tag.id) {
        await addContactTag(tag.id, record.uid);
      } else {
        const created = await handleCreateTag(tag.name);
        if (created?.id) await addContactTag(created.id, record.uid);
      }
      await refreshTags();
    } catch {
      await refreshTags();
    }
  };

  const handleTagRemove = async (tag: Tag) => {
    if (!record?.uid) return;
    try {
      await removeContactTag(tag.id, record.uid);
      await refreshTags();
    } catch {
      await refreshTags();
    }
  };


  // Fetch contact details, notes, and activities
  useEffect(() => {
    if (!id) return;

    let currentBlobUrl: string | null = null;

    const fetchData = async () => {
      try {
        // First batch: parallel fetch of core data
        const [recordData, notesData, activitiesData, completionsData, user] = await Promise.all([
          getContactRecord(id),
          getContactNotes(id),
          getContactActivities(id),
          getCompletionsForContact(parseInt(id)),
          getCurrentUser().catch(err => {
            console.error('Error fetching current user preferences:', err);
            return null;
          })
        ]);

        setRecord(recordData);
        setNotes(notesData.notes || []);
        setActivities(activitiesData.activities || []);
        setCompletions(completionsData || []);
        setCustomFieldNames(user?.custom_field_names ?? []);
        setEnabledFields(resolveEnabledFields(user?.enabled_contact_fields ?? null));

        // Second batch: refresh reminders and relationship edges in
        // parallel. refreshRelationshipEdges is passed recordData.uid
        // directly rather than relying on the `record` state var -- that
        // state hasn't re-rendered yet at this point in the effect, so
        // relying on it would silently fetch zero edges on every fresh
        // page load.
        await Promise.all([
          refreshReminders(),
          refreshRelationshipEdges(recordData.uid),
          refreshLifeEvents(recordData.uid),
        ]);

        // Only fetch profile picture if contact has one (avoid unnecessary 404)
        if (recordData.photo) {
          try {
            const blob = await getContactProfilePicture(id);
            if (blob) {
              currentBlobUrl = URL.createObjectURL(blob);
              setProfilePic(currentBlobUrl);
            } else {
              setProfilePic('');
            }
          } catch (err) {
            console.error('Error fetching profile picture:', err);
          }
        } else {
          setProfilePic('');
        }

        setLoading(false);
      } catch (err) {
        console.error('Error fetching data:', err);
        setLoading(false);
      }
    };

    fetchData();

    return () => {
      if (currentBlobUrl) {
        URL.revokeObjectURL(currentBlobUrl);
      }
    };
  }, [id, refreshReminders, refreshRelationshipEdges, refreshLifeEvents]);

  // Combine and sort notes, activities, completions, and life events for timeline
  const timelineItems: Array<{ type: 'note' | 'activity' | 'completion' | 'life_event'; data: Note | Activity | ReminderCompletion | LifeEvent; date: string }> = [
    ...notes.map(note => ({
      type: 'note' as const,
      data: note,
      date: note.date || note.CreatedAt
    })),
    ...activities.map(activity => ({
      type: 'activity' as const,
      data: activity,
      date: activity.date || activity.CreatedAt
    })),
    ...completions.map(completion => ({
      type: 'completion' as const,
      data: completion,
      date: completion.completed_at
    })),
    ...lifeEvents.filter(e => e.date != null).map(event => ({
      type: 'life_event' as const,
      data: event,
      date: fullDateFromPartial(event.date!) || event.created_at
    }))
  ].sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime());

  const handleDeleteCompletion = async (completionId: number) => {
    if (!window.confirm(t('timeline.deleteCompletionConfirm'))) {
      return;
    }
    try {
      await deleteCompletion(completionId);
      await refreshNotesAndActivities();
    } catch (err) {
      handleFetchError(err, 'deleting completion');
    }
  };

  const validateBirthday = (value: string): boolean => {
    if (!value || value.trim() === '') return true;
    // Try to parse the birthday input - if it returns null, it's invalid
    const parsed = parseBirthdayInput(value);
    return parsed !== null;
  };

  const handleEditStart = (field: string, currentValue: string) => {
    setEditingField(field);
    // For date fields, convert from ISO to display format
    if ((field === 'birthday' || field === 'anniversary') && currentValue) {
      setEditValue(formatBirthdayForInput(currentValue));
    } else {
      setEditValue(currentValue || '');
    }
    setValidationError('');
  };

  const handleEditCancel = () => {
    setEditingField(null);
    setEditValue('');
    setValidationError('');
  };

  // Maps one of ContactInformation's scalar field names to the Card/CRM
  // patch it corresponds to. Lives here (not in ContactInformation) because
  // building an organization/title patch needs to know the *other* half of
  // the pair (department when editing organization, and vice versa) --
  // which only the current `record` has.
  const buildRecordPatch = (field: string, value: string): { card?: Partial<CardModel>; crm?: Partial<CRMEnvelope> } => {
    const card = record?.card || {};
    const crm = record?.crm || {};
    switch (field) {
      case 'birthday':
        return { card: { anniversaries: withAnniversary(card.anniversaries, 'birth', value) } };
      case 'anniversary':
        return { card: { anniversaries: withAnniversary(card.anniversaries, 'wedding', value) } };
      case 'organization': {
        const { department } = getOrganizationFields(card.organizations);
        return { card: { organizations: withOrganization(value, department || '') } };
      }
      case 'department': {
        const { organization } = getOrganizationFields(card.organizations);
        return { card: { organizations: withOrganization(organization || '', value) } };
      }
      case 'job_title': {
        const role = getTitleField(card.titles, 'role');
        return { card: { titles: withTitles(value, role || '') } };
      }
      case 'role': {
        const jobTitle = getTitleField(card.titles, 'title');
        return { card: { titles: withTitles(jobTitle || '', value) } };
      }
      case 'work_information':
        return { crm: { work_information: value } };
      case 'how_we_met':
        return { crm: { how_we_met: value } };
      case 'contact_information':
        return { crm: { contact_information: value } };
      default:
        if (field.startsWith('custom_field_')) {
          const name = field.replace('custom_field_', '');
          return { crm: { custom_fields: { ...crm.custom_fields, [name]: value } } };
        }
        return {};
    }
  };

  const handleEditSave = async (field: string) => {
    if (!record) return;

    let valueToSave = editValue;

    if (field === 'birthday' || field === 'anniversary') {
      if (!validateBirthday(editValue)) {
        setValidationError(t('contactDetail.birthdayError'));
        return;
      }
      // Convert from display format to ISO format for storage
      const parsed = parseBirthdayInput(editValue);
      valueToSave = parsed || '';
    }

    const patch = buildRecordPatch(field, valueToSave);

    try {
      const updated = await updateContactRecord(id!, {
        gender: record.gender,
        card: { ...record.card, ...patch.card },
        crm: { ...record.crm, ...patch.crm },
      });
      setRecord(updated);
      setEditingField(null);
      setEditValue('');
      setValidationError('');
    } catch (err) {
      console.error('Error updating contact:', err);
      if (err instanceof ApiError) {
        const errorMessage = err.getDisplayMessage();
        setValidationError(errorMessage);
        showError(errorMessage);
      } else {
        showError(t('contactDetail.updateError'));
      }
    }
  };

  // Persist multi-valued / structured field updates (emails, phones, addresses, links, imppAddresses)
  const handleUpdateCard = async (patch: Partial<CardModel>) => {
    if (!record) return;
    try {
      const updated = await updateContactRecord(id!, {
        gender: record.gender,
        card: { ...record.card, ...patch },
        crm: record.crm,
      });
      setRecord(updated);
    } catch (err) {
      console.error('Error updating contact:', err);
      if (err instanceof ApiError) {
        showError(err.getDisplayMessage());
      } else {
        showError(t('contactDetail.updateError'));
      }
      throw err;
    }
  };


  const handleStartEditProfile = () => {
    if (!record) return;
    const components = record.card?.name?.components;
    setProfileValues({
      prefix: nameComponentValue(components, 'title') || '',
      firstname: nameComponentValue(components, 'given') || '',
      middle_name: nameComponentValue(components, 'given2') || '',
      lastname: nameComponentValue(components, 'surname') || '',
      suffix: nameComponentValue(components, 'generation') || '',
      nickname: record.card?.nicknames?.[0]?.name || '',
      gender: record.gender ? record.gender.toLowerCase() : ''
    });
    setEditingProfile(true);
  };

  const handleCancelEditProfile = () => {
    setEditingProfile(false);
    setProfileValues({ prefix: '', firstname: '', middle_name: '', lastname: '', suffix: '', nickname: '', gender: '' });
  };

  const handleSaveProfile = async () => {
    if (!record || !profileValues.firstname.trim()) {
      alert(t('contactDetail.firstNameRequired'));
      return;
    }

    const nameComponents: NameComponent[] = [];
    if (profileValues.prefix.trim()) nameComponents.push({ kind: 'title', value: profileValues.prefix.trim() });
    nameComponents.push({ kind: 'given', value: profileValues.firstname.trim() });
    if (profileValues.middle_name.trim()) nameComponents.push({ kind: 'given2', value: profileValues.middle_name.trim() });
    nameComponents.push({ kind: 'surname', value: profileValues.lastname.trim() });
    if (profileValues.suffix.trim()) nameComponents.push({ kind: 'generation', value: profileValues.suffix.trim() });

    try {
      const updated = await updateContactRecord(id!, {
        gender: profileValues.gender,
        card: {
          ...record.card,
          name: { components: nameComponents },
          nicknames: profileValues.nickname.trim() ? [{ name: profileValues.nickname.trim() }] : undefined,
        },
        crm: record.crm,
      });
      setRecord(updated);
      setEditingProfile(false);
    } catch (err) {
      console.error('Error updating profile:', err);
      if (err instanceof ApiError) {
        showError(err.getDisplayMessage());
      } else {
        showError(t('contactDetail.updateError'));
      }
    }
  };

  const handleDeleteContact = async () => {
    if (!record || !id) return;

    const confirmMessage = t('contactDetail.confirmDeleteContact', {
      name: `${firstname} ${lastname}`
    });

    if (!window.confirm(confirmMessage)) {
      return;
    }

    try {
      await deleteContact(id);
      navigate('/contacts');
    } catch (err) {
      console.error('Error deleting contact:', err);
      alert(t('contactDetail.deleteContactError'));
    }
  };

  const handleArchiveContact = async () => {
    if (!record || !id) return;

    const confirmMessage = t('contactDetail.archiveConfirmation');
    if (!window.confirm(confirmMessage)) {
      return;
    }

    try {
      const updatedContact = await archiveContact(id);
      setRecord({ ...record, archived: updatedContact.archived });
    } catch (err) {
      console.error('Error archiving contact:', err);
      if (err instanceof ApiError) {
        showError(err.getDisplayMessage());
      } else {
        showError(t('contactDetail.updateError'));
      }
    }
  };

  const handleUnarchiveContact = async () => {
    if (!record || !id) return;

    try {
      const updatedContact = await unarchiveContact(id);
      setRecord({ ...record, archived: updatedContact.archived });
    } catch (err) {
      console.error('Error unarchiving contact:', err);
      if (err instanceof ApiError) {
        showError(err.getDisplayMessage());
      } else {
        showError(t('contactDetail.updateError'));
      }
    }
  };

  const handleStayInTouch = () => {
    if (!record) return;
    const contactName = `${firstname}${lastname ? ' ' + lastname : ''}`;
    setReminderInitialValues({
      message: t('contactDetail.catchUpWith', { name: contactName }),
      recurrence: 'quarterly'
    });
    setEditingReminder(null);
    setReminderDialogOpen(true);
  };

  const handleUploadProfilePicture = async (croppedImageBlob: Blob) => {
    if (!id) return;

    await uploadProfilePicture(id, croppedImageBlob);

    // Refresh the profile picture
    const blob = await getContactProfilePicture(id);
    if (blob) {
      // Revoke old URL to prevent memory leaks
      if (profilePic) {
        URL.revokeObjectURL(profilePic);
      }
      setProfilePic(URL.createObjectURL(blob));
    }
  };


  if (loading) {
    return (
      <Box sx={{ maxWidth: 1200, mx: 'auto', mt: 1, px: 2, pb: 2 }}>
        <ContactDetailHeaderSkeleton />
        <Box sx={{ mt: 3 }}>
          <TimelineSkeleton count={5} />
        </Box>
      </Box>
    );
  }

  if (!record) {
    return (
      <Box sx={{ maxWidth: 800, mx: 'auto', mt: 2, p: 2 }}>
        <Typography variant="h6">{t('contactDetail.notFound')}</Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ maxWidth: 1200, mx: 'auto', mt: 1, px: 2, pb: 2 }}>

      {/* Contact Header Card */}
      <ContactHeader
        record={record}
        profilePic={profilePic}
        editingProfile={editingProfile}
        profileValues={profileValues}
        enabledFields={enabledFields}
        contactCircles={contactCircles}
        contactTags={contactTags}
        allCircles={allCircles}
        allTags={allTags}
        onStartEditProfile={handleStartEditProfile}
        onCancelEditProfile={handleCancelEditProfile}
        onSaveProfile={handleSaveProfile}
        onDeleteContact={handleDeleteContact}
        onProfileValueChange={setProfileValues}
        onAddCircle={handleCircleAdd}
        onRemoveCircle={handleCircleRemove}
        onAddTag={handleTagAdd}
        onRemoveTag={handleTagRemove}
        onUploadProfilePicture={() => setProfilePictureDialogOpen(true)}
        onStayInTouch={record.archived ? undefined : handleStayInTouch}
        onArchiveContact={record.archived ? undefined : handleArchiveContact}
        onUnarchiveContact={record.archived ? handleUnarchiveContact : undefined}
        onMergeContact={() => setMergeDialogOpen(true)}
      />

      {record && (
        <MergeContactsDialog
          open={mergeDialogOpen}
          onClose={() => setMergeDialogOpen(false)}
          onMerged={(keeperId) => navigate(`/contacts/${keeperId}`)}
          currentContactId={record.id}
          currentContactUid={record.uid}
          currentContactName={`${firstname} ${lastname}`.trim()}
        />
      )}

      {/* General Information and Timeline - Two Column Layout */}
      <Box sx={{ 
        display: 'flex', 
        flexDirection: { xs: 'column', md: 'row' }, 
        gap: 2 
      }}>
        {/* General Information */}
        <ContactInformation
          card={record.card}
          crm={record.crm}
          editingField={editingField}
          editValue={editValue}
          validationError={validationError}
          onEditStart={handleEditStart}
          onEditCancel={handleEditCancel}
          onEditSave={handleEditSave}
          onEditValueChange={(value) => {
            setEditValue(
              editingField === 'birthday' || editingField === 'anniversary'
                ? autoFormatBirthdayInput(value, editValue)
                : value
            );
            setValidationError('');
          }}
          onUpdateCard={handleUpdateCard}
          enabledFields={enabledFields}
          confirmedEdges={confirmedEdges}
          suggestedEdges={suggestedEdges}
          contactsByUid={contactsByUid}
          viewedContactUid={record?.uid}
          onAddRelationshipEdge={handleAddRelationshipEdge}
          onEditRelationshipEdge={handleEditRelationshipEdge}
          onDeleteRelationshipEdge={handleDeleteRelationshipEdge}
          onAcceptSuggestion={handleAcceptSuggestion}
          onRejectSuggestion={handleRejectSuggestion}
          lifeEvents={lifeEvents}
          lifeEventsContactsByUid={lifeEventsContactsByUid}
          onAddLifeEvent={handleAddLifeEvent}
          onEditLifeEvent={handleEditLifeEvent}
          onDeleteLifeEvent={handleLifeEventDelete}
          preferences={preferences}
          onAddPreference={handleAddPreference}
          onEditPreference={handleEditPreference}
          onDeletePreference={handlePreferenceDelete}
          customFieldNames={customFieldNames}
        />

        {/* Timeline and Reminders Tabs */}
        <Card sx={{ flex: 1 }}>
          <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
            <Tabs value={activeTab} onChange={(_, newValue) => setActiveTab(newValue)} aria-label="timeline and reminders tabs">
              <Tab label={t('contactDetail.timeline')} />
              <Tab label={t('reminders.title')} />
            </Tabs>
          </Box>

          {/* Tab Panel 0: Timeline - Notes and Activities */}
          {activeTab === 0 && (
            <CardContent sx={{ py: 2 }}>
              <Box sx={{ display: 'flex', justifyContent: 'flex-end', alignItems: 'center', mb: 1.5, gap: 0.5 }}>
                <Button 
                  startIcon={<NoteIcon />} 
                  onClick={() => setNoteDialogOpen(true)}
                  variant="outlined"
                  size="small"
                >
                  {t('contactDetail.addNote')}
                </Button>
                <Button 
                  startIcon={<EventIcon />} 
                  onClick={() => setActivityDialogOpen(true)}
                  variant="outlined"
                  size="small"
                >
                  {t('contactDetail.addActivity')}
                </Button>
              </Box>
              <Divider sx={{ mb: 2 }} />
              
              <ContactTimeline
                timelineItems={timelineItems}
                onEditItem={handleStartEditTimelineItem}
                onDeleteCompletion={handleDeleteCompletion}
              />
            </CardContent>
          )}

          {/* Tab Panel 1: Reminders */}
          {activeTab === 1 && (
            <CardContent sx={{ py: 2 }}>
              <Box sx={{ display: 'flex', justifyContent: 'flex-end', alignItems: 'center', mb: 1.5 }}>
                <Button 
                  startIcon={<NotificationsActiveIcon />} 
                  onClick={handleAddReminder}
                  variant="outlined"
                  size="small"
                >
                  {t('reminders.add')}
                </Button>
              </Box>
              <Divider sx={{ mb: 1.5 }} />
              <ReminderList
                reminders={reminders}
                onComplete={handleCompleteReminder}
                onEdit={handleEditReminder}
                onDelete={handleDeleteReminder}
              />
            </CardContent>
          )}
        </Card>
      </Box>

      {/* Dialogs */}
      <AddNoteDialog
        open={noteDialogOpen}
        onClose={() => setNoteDialogOpen(false)}
        onSave={handleSaveNote}
      />
      
      <AddActivityDialog
        open={activityDialogOpen}
        onClose={() => setActivityDialogOpen(false)}
        onSave={handleSaveActivity}
        preselectedContactId={record?.id}
      />

      <ReminderDialog
        open={reminderDialogOpen}
        onClose={() => {
          setReminderDialogOpen(false);
          setEditingReminder(null);
          setReminderInitialValues(undefined);
        }}
        onSave={handleSaveReminder}
        reminder={editingReminder}
        contactId={record?.id || 0}
        initialValues={reminderInitialValues}
      />

      {editingTimelineItem && (
        <EditTimelineItemDialog
          open={!!editingTimelineItem}
          onClose={handleCancelEditTimelineItem}
          onSave={() => {
            if (editingTimelineItem.type === 'note') {
              handleUpdateNote(editingTimelineItem.id);
            } else {
              handleUpdateActivity(editingTimelineItem.id);
            }
          }}
          onDelete={() => {
            if (editingTimelineItem.type === 'note') {
              handleDeleteNote(editingTimelineItem.id);
            } else {
              handleDeleteActivity(editingTimelineItem.id);
            }
          }}
          type={editingTimelineItem.type}
          values={editTimelineValues}
          onChange={setEditTimelineValues}
          allContacts={allContacts}
        />
      )}

      <ProfilePictureUploadDialog
        open={profilePictureDialogOpen}
        onClose={() => setProfilePictureDialogOpen(false)}
        onUpload={handleUploadProfilePicture}
      />

      <RelationshipEdgeDialog
        open={relationshipDialogOpen}
        onClose={() => {
          setRelationshipDialogOpen(false);
          setEditingEdge(null);
        }}
        onSave={handleSaveRelationshipEdge}
        edge={editingEdge}
        viewedContactUid={record?.uid || ''}
        otherPartyContact={editingEdgeOtherParty}
      />

      <LifeEventDialog
        open={lifeEventDialogOpen}
        onClose={() => {
          setLifeEventDialogOpen(false);
          setEditingLifeEvent(null);
        }}
        onSave={handleSaveLifeEvent}
        initial={
          editingLifeEvent
            ? {
                type: editingLifeEvent.type,
                date: editingLifeEvent.date,
                description: editingLifeEvent.description,
                relatedEntityIds: editingLifeEvent.related_entity_ids,
                remind: editingLifeEvent.remind,
              }
            : undefined
        }
        excludeContactUid={record?.uid}
      />

      <PreferenceDialog
        open={preferenceDialogOpen}
        onClose={() => {
          setPreferenceDialogOpen(false);
          setEditingPreference(null);
        }}
        onSave={handleSavePreferenceSubmit}
        preference={editingPreference}
      />
    </Box>
  );
}
