import {
  Box,
  Typography,
  IconButton,
  Stack,
  Paper,
  Link,
  Divider,
  Chip,
} from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import CheckIcon from '@mui/icons-material/Check';
import CloseIcon from '@mui/icons-material/Close';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { RelationshipEdge, getDisplayLabel, getOtherPartyId } from '../api/relationshipEdges';
import { Contact } from '../api/contacts';

interface RelationshipEdgeListProps {
  confirmedEdges: RelationshipEdge[];
  suggestedEdges: RelationshipEdge[];
  contactsByUid: Map<string, Contact>;
  viewedContactUid: string;
  onEdit: (edge: RelationshipEdge) => void;
  onDelete: (edgeId: string) => void;
  onAccept: (edgeId: string) => void;
  onReject: (edgeId: string) => void;
}

export default function RelationshipEdgeList({
  confirmedEdges,
  suggestedEdges,
  contactsByUid,
  viewedContactUid,
  onEdit,
  onDelete,
  onAccept,
  onReject,
}: RelationshipEdgeListProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();

  const handleDeleteClick = (edgeId: string) => {
    if (window.confirm(t('relationships.deleteMessage'))) {
      onDelete(edgeId);
    }
  };

  const handleRejectClick = (edgeId: string) => {
    if (window.confirm(t('relationships.rejectMessage'))) {
      onReject(edgeId);
    }
  };

  const handleContactClick = (contactId: number) => {
    navigate(`/contacts/${contactId}`);
  };

  const otherPartyName = (edge: RelationshipEdge): { name: string; contact?: Contact } => {
    const contact = contactsByUid.get(getOtherPartyId(edge, viewedContactUid));
    return { name: contact ? `${contact.firstname} ${contact.lastname}` : t('relationships.unknownContact'), contact };
  };

  const hasNoRelationships = confirmedEdges.length === 0 && suggestedEdges.length === 0;

  if (hasNoRelationships) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ py: 2, textAlign: 'center' }}>
        {t('relationships.noRelationships')}
      </Typography>
    );
  }

  return (
    <Stack spacing={1.5}>
      {confirmedEdges.map((edge) => {
        const { name, contact } = otherPartyName(edge);
        return (
          <Paper
            key={edge.id}
            variant="outlined"
            sx={{
              p: 2,
              '&:hover .action-buttons': {
                opacity: 1,
              },
            }}
          >
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
              <Box sx={{ flex: 1 }}>
                {contact ? (
                  <Link
                    component="button"
                    variant="subtitle1"
                    onClick={() => handleContactClick(contact.ID)}
                    sx={{ fontWeight: 500, textAlign: 'left', p: 0 }}
                  >
                    {name}
                  </Link>
                ) : (
                  <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
                    {name}
                  </Typography>
                )}
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Typography variant="body2" color="text.secondary">
                    {t(getDisplayLabel(edge, viewedContactUid))}
                  </Typography>
                  {edge.sensitivity !== 'normal' && (
                    <Chip
                      label={t(`relationships.sensitivity${edge.sensitivity.charAt(0).toUpperCase()}${edge.sensitivity.slice(1)}`)}
                      size="small"
                      sx={{ height: 18 }}
                    />
                  )}
                </Box>
              </Box>
              <Box
                className="action-buttons"
                sx={{
                  display: 'flex',
                  gap: 0.5,
                  opacity: 0,
                  transition: 'opacity 0.2s ease-in-out',
                }}
              >
                <IconButton size="small" onClick={() => onEdit(edge)} aria-label={t('common.edit')}>
                  <EditIcon fontSize="small" />
                </IconButton>
                <IconButton
                  size="small"
                  onClick={() => handleDeleteClick(edge.id)}
                  aria-label={t('common.delete')}
                  color="error"
                >
                  <DeleteIcon fontSize="small" />
                </IconButton>
              </Box>
            </Box>
          </Paper>
        );
      })}

      {suggestedEdges.length > 0 && (
        <>
          {confirmedEdges.length > 0 && <Divider sx={{ my: 2 }} />}
          <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1 }}>
            {t('relationships.suggestedRelationships')}
          </Typography>
          {suggestedEdges.map((edge) => {
            const { name, contact } = otherPartyName(edge);
            return (
              <Paper key={edge.id} variant="outlined" sx={{ p: 2, bgcolor: 'action.hover' }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                  <Box sx={{ flex: 1 }}>
                    {contact ? (
                      <Link
                        component="button"
                        variant="subtitle1"
                        onClick={() => handleContactClick(contact.ID)}
                        sx={{ fontWeight: 500, textAlign: 'left', p: 0 }}
                      >
                        {name}
                      </Link>
                    ) : (
                      <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
                        {name}
                      </Typography>
                    )}
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <Typography variant="body2" color="text.secondary">
                        {t(getDisplayLabel(edge, viewedContactUid))}
                      </Typography>
                      <Chip label={t('relationships.suggested')} color="warning" size="small" sx={{ height: 18 }} />
                    </Box>
                  </Box>
                  {/* Always visible, not hover-gated -- a review inbox needs a
                      discoverable action. */}
                  <Box sx={{ display: 'flex', gap: 0.5 }}>
                    <IconButton
                      size="small"
                      onClick={() => onAccept(edge.id)}
                      aria-label={t('relationships.accept')}
                      color="success"
                    >
                      <CheckIcon fontSize="small" />
                    </IconButton>
                    <IconButton
                      size="small"
                      onClick={() => handleRejectClick(edge.id)}
                      aria-label={t('relationships.reject')}
                      color="error"
                    >
                      <CloseIcon fontSize="small" />
                    </IconButton>
                  </Box>
                </Box>
              </Paper>
            );
          })}
        </>
      )}
    </Stack>
  );
}
