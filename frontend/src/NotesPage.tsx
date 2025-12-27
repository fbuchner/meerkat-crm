import { useState, useMemo, ChangeEvent } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box,
  Typography,
  Paper,
  TextField,
  Button,
  IconButton,
  Pagination,
} from '@mui/material';
import {
  Timeline,
  TimelineItem,
  TimelineSeparator,
  TimelineConnector,
  TimelineContent,
  TimelineDot,
  TimelineOppositeContent,
} from '@mui/lab';
import NoteIcon from '@mui/icons-material/Note';
import EditIcon from '@mui/icons-material/Edit';
import { ListSkeleton } from './components/LoadingSkeletons';
import { useNotes } from './hooks/useNotes';
import { useDebouncedValue } from './hooks/useDebounce';
import { createUnassignedNote, updateNote, deleteNote, Note } from './api/notes';
import AddNoteDialog from './components/AddNoteDialog';
import EditTimelineItemDialog from './components/EditTimelineItemDialog';
import { handleError } from './utils/errorHandler';

interface NotesPageProps {
  token: string;
}

const NotesPage: React.FC<NotesPageProps> = ({ token }) => {
  const { t } = useTranslation();
  const [searchInput, setSearchInput] = useState('');
  const debouncedSearch = useDebouncedValue(searchInput, 400);
  const [page, setPage] = useState(1);
  const NOTES_PER_PAGE = 25;

  const notesParams = useMemo(
    () => ({
      page,
      limit: NOTES_PER_PAGE,
      search: debouncedSearch.trim() || undefined,
    }),
    [page, debouncedSearch]
  );

  const {
    notes,
    total,
    page: serverPage,
    limit,
    loading,
    refetch,
  } = useNotes(undefined, notesParams);
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [editingNote, setEditingNote] = useState<Note | null>(null);
  const [editValues, setEditValues] = useState<{ noteContent?: string; noteDate?: string }>({});

  const handleSearchChange = (value: string) => {
    setSearchInput(value);
    setPage(1);
  };

  const handlePageChange = (_: ChangeEvent<unknown>, value: number) => {
    setPage(value);
  };

  const currentPage = serverPage || page;
  const pageSize = limit || NOTES_PER_PAGE;
  const totalPages = Math.max(1, Math.ceil((total || 0) / pageSize));

  const handleAddNote = () => {
    setAddDialogOpen(true);
  };

  const handleNoteSave = async (content: string, date: string) => {
    try {
      await createUnassignedNote({ content, date: new Date(date).toISOString() }, token);
      setAddDialogOpen(false);
      refetch();
    } catch (err) {
      handleError(err, { operation: 'creating note' });
      throw err;
    }
  };

  const handleEditClick = (note: Note) => {
    setEditingNote(note);
    setEditValues({
      noteContent: note.content || '',
      noteDate: note.date ? new Date(note.date).toISOString().split('T')[0] : '',
    });
  };

  const handleSaveEdit = async () => {
    if (!editingNote || !editValues.noteContent?.trim()) return;

    try {
      await updateNote(editingNote.ID, {
        content: editValues.noteContent,
        date: editValues.noteDate ? new Date(editValues.noteDate).toISOString() : new Date().toISOString(),
      }, token);
      setEditingNote(null);
      setEditValues({});
      refetch();
    } catch (err) {
      handleError(err, { operation: 'updating note' });
    }
  };

  const handleCancelEdit = () => {
    setEditingNote(null);
    setEditValues({});
  };

  const handleDeleteNote = async () => {
    if (!editingNote) return;

    try {
      await deleteNote(editingNote.ID, token);
      setEditingNote(null);
      setEditValues({});
      refetch();
    } catch (err) {
      handleError(err, { operation: 'deleting note' });
    }
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  };

  const isInitialLoading = loading && notes.length === 0;
  const hasSearchQuery = searchInput.trim().length > 0;

  return (
    <Box sx={{ maxWidth: 1200, mx: 'auto', mt: 2, p: 2 }}>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Typography variant="h5">{t('notes.title')}</Typography>
        <Button variant="outlined" startIcon={<NoteIcon />} onClick={handleAddNote}>
          {t('notes.addNote')}
        </Button>
      </Box>

      <Paper sx={{ p: 1.5, mb: 2 }}>
        <TextField
          fullWidth
          size="small"
          label={t('notes.search')}
          value={searchInput}
          onChange={(e) => handleSearchChange(e.target.value)}
          variant="outlined"
        />
      </Paper>

      {isInitialLoading ? (
        <ListSkeleton count={8} />
      ) : notes.length === 0 ? (
        <Paper sx={{ p: 3, textAlign: 'center' }}>
          <Typography variant="body1" color="text.secondary">
            {hasSearchQuery ? t('notes.noResults') : t('notes.noNotes')}
          </Typography>
        </Paper>
      ) : (
        <Timeline position="right">
          {notes.map((note, index) => (
            <TimelineItem key={note.ID}>
              <TimelineOppositeContent color="text.secondary" sx={{ flex: 0.2 }}>
                {formatDate(note.date)}
              </TimelineOppositeContent>
              <TimelineSeparator>
                <TimelineDot color="primary">
                  <NoteIcon />
                </TimelineDot>
                {index < notes.length - 1 && <TimelineConnector />}
              </TimelineSeparator>
              <TimelineContent sx={{ flex: 0.8 }}>
                <Paper
                  elevation={2}
                  sx={{
                    p: 2,
                    '&:hover .edit-actions': {
                      opacity: 1,
                    },
                  }}
                >
                  <Box>
                    <Box display="flex" justifyContent="space-between" alignItems="flex-start">
                      <Typography variant="body1" sx={{ whiteSpace: 'pre-wrap', flex: 1 }}>
                        {note.content}
                      </Typography>
                      <Box
                        className="edit-actions"
                        sx={{ opacity: 0, transition: 'opacity 0.2s', display: 'flex', gap: 1 }}
                      >
                        <IconButton size="small" onClick={() => handleEditClick(note)}>
                          <EditIcon fontSize="small" />
                        </IconButton>
                      </Box>
                    </Box>
                  </Box>
                </Paper>
              </TimelineContent>
            </TimelineItem>
          ))}
        </Timeline>
      )}

      {totalPages > 1 && (
        <Box display="flex" justifyContent="center" mt={3}>
          <Pagination
            color="primary"
            count={totalPages}
            page={currentPage}
            onChange={handlePageChange}
            showFirstButton
            showLastButton
          />
        </Box>
      )}

      <AddNoteDialog
        open={addDialogOpen}
        onClose={() => setAddDialogOpen(false)}
        onSave={handleNoteSave}
      />

      {editingNote && (
        <EditTimelineItemDialog
          open={!!editingNote}
          onClose={handleCancelEdit}
          onSave={handleSaveEdit}
          onDelete={handleDeleteNote}
          type="note"
          values={editValues}
          onChange={setEditValues}
          allContacts={[]}
        />
      )}
    </Box>
  );
};

export default NotesPage;
