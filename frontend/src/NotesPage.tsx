import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box,
  Typography,
  Paper,
  TextField,
  Button,
  IconButton,
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
import DeleteIcon from '@mui/icons-material/Delete';
import { ListSkeleton } from './components/LoadingSkeletons';
import { useNotes } from './hooks/useNotes';
import { createUnassignedNote, updateNote, deleteNote, Note } from './api/notes';
import AddNoteDialog from './components/AddNoteDialog';
import EditTimelineItemDialog from './components/EditTimelineItemDialog';

interface NotesPageProps {
  token: string;
}

const NotesPage: React.FC<NotesPageProps> = ({ token }) => {
  const { t } = useTranslation();
  const { notes: allNotes, loading, refetch } = useNotes();
  const [filteredNotes, setFilteredNotes] = useState<Note[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [editingNote, setEditingNote] = useState<Note | null>(null);
  const [editValues, setEditValues] = useState<{ noteContent?: string; noteDate?: string }>({});

  // Sort notes by date descending (newest first) and filter
  useEffect(() => {
    const sorted = [...allNotes].sort((a, b) => {
      return new Date(b.date).getTime() - new Date(a.date).getTime();
    });

    if (searchQuery.trim() === '') {
      setFilteredNotes(sorted);
    } else {
      const query = searchQuery.toLowerCase();
      const filtered = sorted.filter((note) => {
        return note.content?.toLowerCase().includes(query);
      });
      setFilteredNotes(filtered);
    }
  }, [searchQuery, allNotes]);

  const handleAddNote = () => {
    setAddDialogOpen(true);
  };

  const handleNoteSave = async (content: string, date: string) => {
    try {
      await createUnassignedNote({ content, date: new Date(date).toISOString() }, token);
      setAddDialogOpen(false);
      refetch();
    } catch (err) {
      console.error('Failed to create note:', err);
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
      console.error('Failed to update note:', err);
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
      console.error('Failed to delete note:', err);
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

  if (loading) {
    return (
      <Box>
        <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
          <Typography variant="h4">{t('notes.title')}</Typography>
        </Box>
        <ListSkeleton count={8} />
      </Box>
    );
  }

  return (
    <Box>
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
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          variant="outlined"
        />
      </Paper>

      {filteredNotes.length === 0 ? (
        <Paper sx={{ p: 3, textAlign: 'center' }}>
          <Typography variant="body1" color="text.secondary">
            {searchQuery ? t('notes.noResults') : t('notes.noNotes')}
          </Typography>
        </Paper>
      ) : (
        <Timeline position="right">
          {filteredNotes.map((note, index) => (
            <TimelineItem key={note.ID}>
              <TimelineOppositeContent color="text.secondary" sx={{ flex: 0.2 }}>
                {formatDate(note.date)}
              </TimelineOppositeContent>
              <TimelineSeparator>
                <TimelineDot color="primary">
                  <NoteIcon />
                </TimelineDot>
                {index < filteredNotes.length - 1 && <TimelineConnector />}
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
                        <IconButton
                          size="small"
                          color="error"
                          onClick={() => handleEditClick(note)}
                        >
                          <DeleteIcon fontSize="small" />
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
