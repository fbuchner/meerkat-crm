import { useState, useEffect, FormEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  IconButton,
  Chip,
  TablePagination,
  CircularProgress,
  Alert,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  FormControlLabel,
  Switch,
  Stack,
} from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import { getUsers, updateUser, deleteUser } from './api/admin';
import { isAdmin } from './auth';
import type { User, UserUpdateInput } from './types';

export default function UsersPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);
  const [total, setTotal] = useState(0);

  // Edit dialog state
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [editForm, setEditForm] = useState<UserUpdateInput>({});
  const [editLoading, setEditLoading] = useState(false);
  const [editError, setEditError] = useState('');

  // Delete dialog state
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deletingUser, setDeletingUser] = useState<User | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [deleteError, setDeleteError] = useState('');

  // Check admin access
  useEffect(() => {
    if (!isAdmin()) {
      navigate('/');
    }
  }, [navigate]);

  // Fetch users
  useEffect(() => {
    const fetchUsers = async () => {
      setLoading(true);
      setError('');

      try {
        const response = await getUsers(page + 1, rowsPerPage);
        setUsers(response.users);
        setTotal(response.total);
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : t('users.loadError');
        setError(errorMessage);
      } finally {
        setLoading(false);
      }
    };

    fetchUsers();
  }, [page, rowsPerPage, t]);

  const handleChangePage = (_: unknown, newPage: number) => {
    setPage(newPage);
  };

  const handleChangeRowsPerPage = (event: React.ChangeEvent<HTMLInputElement>) => {
    setRowsPerPage(parseInt(event.target.value, 10));
    setPage(0);
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString();
  };

  // Edit handlers
  const handleEditClick = (user: User) => {
    setEditingUser(user);
    setEditForm({
      username: user.username,
      email: user.email,
      is_admin: user.is_admin,
    });
    setEditError('');
    setEditDialogOpen(true);
  };

  const handleEditClose = () => {
    setEditDialogOpen(false);
    setEditingUser(null);
    setEditForm({});
    setEditError('');
  };

  const handleEditSubmit = async (event: FormEvent) => {
    event.preventDefault();
    if (!editingUser) return;

    setEditLoading(true);
    setEditError('');

    try {
      // Only include fields that have changed
      const updateData: UserUpdateInput = {};
      if (editForm.username !== editingUser.username) {
        updateData.username = editForm.username;
      }
      if (editForm.email !== editingUser.email) {
        updateData.email = editForm.email;
      }
      if (editForm.password) {
        updateData.password = editForm.password;
      }
      if (editForm.is_admin !== editingUser.is_admin) {
        updateData.is_admin = editForm.is_admin;
      }

      const updatedUser = await updateUser(editingUser.id, updateData);
      setUsers(users.map(u => u.id === updatedUser.id ? updatedUser : u));
      handleEditClose();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : t('users.updateError');
      setEditError(errorMessage);
    } finally {
      setEditLoading(false);
    }
  };

  // Delete handlers
  const handleDeleteClick = (user: User) => {
    setDeletingUser(user);
    setDeleteError('');
    setDeleteDialogOpen(true);
  };

  const handleDeleteClose = () => {
    setDeleteDialogOpen(false);
    setDeletingUser(null);
    setDeleteError('');
  };

  const handleDeleteConfirm = async () => {
    if (!deletingUser) return;

    setDeleteLoading(true);
    setDeleteError('');

    try {
      await deleteUser(deletingUser.id);
      setUsers(users.filter(u => u.id !== deletingUser.id));
      setTotal(total - 1);
      handleDeleteClose();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : t('users.deleteError');
      setDeleteError(errorMessage);
    } finally {
      setDeleteLoading(false);
    }
  };

  if (loading && users.length === 0) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ maxWidth: 1200, mx: 'auto', mt: 2, p: 2 }}>
      <Typography variant="h5" gutterBottom sx={{ mb: 1.5 }}>
        {t('users.title')}
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>{t('users.columns.id')}</TableCell>
              <TableCell>{t('users.columns.username')}</TableCell>
              <TableCell>{t('users.columns.email')}</TableCell>
              <TableCell>{t('users.columns.role')}</TableCell>
              <TableCell>{t('users.columns.created')}</TableCell>
              <TableCell align="right">{t('users.columns.actions')}</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {users.map((user) => (
              <TableRow key={user.id}>
                <TableCell>{user.id}</TableCell>
                <TableCell>{user.username}</TableCell>
                <TableCell>{user.email}</TableCell>
                <TableCell>
                  <Chip
                    label={user.is_admin ? t('users.roles.admin') : t('users.roles.user')}
                    color={user.is_admin ? 'primary' : 'default'}
                    size="small"
                  />
                </TableCell>
                <TableCell>{formatDate(user.created_at)}</TableCell>
                <TableCell align="right">
                  <IconButton
                    size="small"
                    onClick={() => handleEditClick(user)}
                    title={t('common.edit')}
                  >
                    <EditIcon fontSize="small" />
                  </IconButton>
                  <IconButton
                    size="small"
                    onClick={() => handleDeleteClick(user)}
                    title={t('common.delete')}
                  >
                    <DeleteIcon fontSize="small" />
                  </IconButton>
                </TableCell>
              </TableRow>
            ))}
            {users.length === 0 && !loading && (
              <TableRow>
                <TableCell colSpan={6} align="center">
                  {t('users.noUsers')}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
        <TablePagination
          component="div"
          count={total}
          page={page}
          onPageChange={handleChangePage}
          rowsPerPage={rowsPerPage}
          onRowsPerPageChange={handleChangeRowsPerPage}
          rowsPerPageOptions={[10, 25, 50]}
        />
      </TableContainer>

      {/* Edit User Dialog */}
      <Dialog open={editDialogOpen} onClose={handleEditClose} maxWidth="sm" fullWidth>
        <form onSubmit={handleEditSubmit}>
          <DialogTitle>{t('users.editDialog.title')}</DialogTitle>
          <DialogContent>
            <Stack spacing={2} sx={{ mt: 1 }}>
              {editError && <Alert severity="error">{editError}</Alert>}
              <TextField
                label={t('users.editDialog.username')}
                value={editForm.username || ''}
                onChange={(e) => setEditForm({ ...editForm, username: e.target.value })}
                fullWidth
                size="small"
              />
              <TextField
                label={t('users.editDialog.email')}
                type="email"
                value={editForm.email || ''}
                onChange={(e) => setEditForm({ ...editForm, email: e.target.value })}
                fullWidth
                size="small"
              />
              <TextField
                label={t('users.editDialog.password')}
                type="password"
                value={editForm.password || ''}
                onChange={(e) => setEditForm({ ...editForm, password: e.target.value })}
                fullWidth
                size="small"
                helperText={t('users.editDialog.passwordHint')}
              />
              <FormControlLabel
                control={
                  <Switch
                    checked={editForm.is_admin || false}
                    onChange={(e) => setEditForm({ ...editForm, is_admin: e.target.checked })}
                  />
                }
                label={t('users.editDialog.isAdmin')}
              />
            </Stack>
          </DialogContent>
          <DialogActions>
            <Button onClick={handleEditClose} disabled={editLoading}>
              {t('common.cancel')}
            </Button>
            <Button type="submit" variant="contained" disabled={editLoading}>
              {editLoading ? t('common.saving') : t('common.save')}
            </Button>
          </DialogActions>
        </form>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onClose={handleDeleteClose}>
        <DialogTitle>{t('users.deleteDialog.title')}</DialogTitle>
        <DialogContent>
          {deleteError && <Alert severity="error" sx={{ mb: 2 }}>{deleteError}</Alert>}
          <Typography>
            {t('users.deleteDialog.message', { username: deletingUser?.username })}
          </Typography>
          <Alert severity="warning" sx={{ mt: 2 }}>
            {t('users.deleteDialog.warning')}
          </Alert>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleDeleteClose} disabled={deleteLoading}>
            {t('common.cancel')}
          </Button>
          <Button
            onClick={handleDeleteConfirm}
            color="error"
            variant="contained"
            disabled={deleteLoading}
          >
            {deleteLoading ? t('common.delete') + '...' : t('common.delete')}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
