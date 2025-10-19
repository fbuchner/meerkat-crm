import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box,
  Typography,
  Paper,
  TextField,
  Button,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  CircularProgress,
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
import AddIcon from '@mui/icons-material/Add';
import SaveIcon from '@mui/icons-material/Save';
import CancelIcon from '@mui/icons-material/Cancel';
import { API_BASE_URL } from './api';
import AddNoteDialog from './components/AddNoteDialog';

interface Note {
  ID: number;
  content: string;
  date: string;
  CreatedAt: string;
  UpdatedAt: string;
}

interface NotesPageProps {
  token: string;
}

const NotesPage: React.FC<NotesPageProps> = ({ token }) => {
  const { t } = useTranslation();
  const [notes, setNotes] = useState<Note[]>([]);
  const [filteredNotes, setFilteredNotes] = useState<Note[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [editingNoteId, setEditingNoteId] = useState<number | null>(null);
  const [editValues, setEditValues] = useState<{ content: string; date: string }>({
    content: '',
    date: '',
  });
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [noteToDelete, setNoteToDelete] = useState<number | null>(null);

  const fetchNotes = useCallback(async () => {
    setLoading(true);
    try {
      const response = await fetch(`${API_BASE_URL}/notes`, {
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });

      if (response.ok) {
        const data = await response.json();
        // Backend returns notes directly as an array
        const notesArray = Array.isArray(data) ? data : (data.notes || []);
        // Sort by date descending (newest first)
        const sorted = notesArray.sort((a: Note, b: Note) => {
          return new Date(b.date).getTime() - new Date(a.date).getTime();
        });
        setNotes(sorted);
        setFilteredNotes(sorted);
      }
    } catch (err) {
      console.error('Failed to fetch notes:', err);
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    fetchNotes();
  }, [fetchNotes]);

  useEffect(() => {
    if (searchQuery.trim() === '') {
      setFilteredNotes(notes);
    } else {
      const query = searchQuery.toLowerCase();
      const filtered = notes.filter((note) => {
        return note.content?.toLowerCase().includes(query);
      });
      setFilteredNotes(filtered);
    }
  }, [searchQuery, notes]);

  const handleAddNote = () => {
    setAddDialogOpen(true);
  };

  const handleNoteSave = async (content: string, date: string) => {
    try {
      const response = await fetch(`${API_BASE_URL}/notes`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ content, date }),
      });

      if (response.ok) {
        setAddDialogOpen(false);
        fetchNotes();
      }
    } catch (err) {
      console.error('Failed to create note:', err);
      throw err;
    }
  };

  const handleEditClick = (note: Note) => {
    setEditingNoteId(note.ID);
    setEditValues({
      content: note.content || '',
      date: note.date ? new Date(note.date).toISOString().split('T')[0] : '',
    });
  };

  const handleSaveEdit = async (noteId: number) => {
    try {
      const response = await fetch(`${API_BASE_URL}/notes/${noteId}`, {
        method: 'PUT',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          content: editValues.content,
          date: editValues.date,
        }),
      });

      if (response.ok) {
        setEditingNoteId(null);
        fetchNotes();
      }
    } catch (err) {
      console.error('Failed to update note:', err);
    }
  };

  const handleCancelEdit = () => {
    setEditingNoteId(null);
    setEditValues({ content: '', date: '' });
  };

  const handleDeleteClick = (noteId: number) => {
    setNoteToDelete(noteId);
    setDeleteConfirmOpen(true);
  };

  const handleConfirmDelete = async () => {
    if (!noteToDelete) return;

    try {
      const response = await fetch(`${API_BASE_URL}/notes/${noteToDelete}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });

      if (response.ok) {
        setDeleteConfirmOpen(false);
        setNoteToDelete(null);
        fetchNotes();
      }
    } catch (err) {
      console.error('Failed to delete note:', err);
    }
  };

  const handleCancelDelete = () => {
    setDeleteConfirmOpen(false);
    setNoteToDelete(null);
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
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4">{t('notes.title')}</Typography>
        <Button variant="contained" startIcon={<AddIcon />} onClick={handleAddNote}>
          {t('notes.addNote')}
        </Button>
      </Box>

      <Paper sx={{ p: 2, mb: 3 }}>
        <TextField
          fullWidth
          label={t('notes.search')}
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          variant="outlined"
        />
      </Paper>

      {filteredNotes.length === 0 ? (
        <Paper sx={{ p: 4, textAlign: 'center' }}>
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
                  {editingNoteId === note.ID ? (
                    <Box>
                      <TextField
                        fullWidth
                        multiline
                        rows={4}
                        value={editValues.content}
                        onChange={(e) => setEditValues({ ...editValues, content: e.target.value })}
                        sx={{ mb: 2 }}
                      />
                      <TextField
                        fullWidth
                        type="date"
                        label={t('notes.date')}
                        value={editValues.date}
                        onChange={(e) => setEditValues({ ...editValues, date: e.target.value })}
                        InputLabelProps={{ shrink: true }}
                        sx={{ mb: 2 }}
                      />
                      <Box display="flex" gap={1}>
                        <Button
                          size="small"
                          variant="contained"
                          startIcon={<SaveIcon />}
                          onClick={() => handleSaveEdit(note.ID)}
                        >
                          {t('common.save')}
                        </Button>
                        <Button
                          size="small"
                          variant="outlined"
                          startIcon={<CancelIcon />}
                          onClick={handleCancelEdit}
                        >
                          {t('common.cancel')}
                        </Button>
                      </Box>
                    </Box>
                  ) : (
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
                            onClick={() => handleDeleteClick(note.ID)}
                          >
                            <DeleteIcon fontSize="small" />
                          </IconButton>
                        </Box>
                      </Box>
                    </Box>
                  )}
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

      <Dialog open={deleteConfirmOpen} onClose={handleCancelDelete}>
        <DialogTitle>{t('notes.deleteConfirm')}</DialogTitle>
        <DialogContent>
          <Typography>{t('notes.deleteMessage')}</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCancelDelete}>{t('common.cancel')}</Button>
          <Button onClick={handleConfirmDelete} color="error" variant="contained">
            {t('common.delete')}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default NotesPage;
