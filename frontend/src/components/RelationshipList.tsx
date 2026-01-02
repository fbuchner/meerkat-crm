import {
  Box,
  Typography,
  IconButton,
  Stack,
  Paper,
  Link,
} from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { Relationship } from '../api/relationships';

interface RelationshipListProps {
  relationships: Relationship[];
  onEdit: (relationship: Relationship) => void;
  onDelete: (relationshipId: number) => void;
}

export default function RelationshipList({
  relationships,
  onEdit,
  onDelete,
}: RelationshipListProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();

  const handleDeleteClick = (relationshipId: number) => {
    if (window.confirm(t('relationships.deleteMessage'))) {
      onDelete(relationshipId);
    }
  };

  const handleLinkedContactClick = (contactId: number) => {
    navigate(`/contacts/${contactId}`);
  };

  const formatGender = (gender?: string) => {
    if (!gender) return null;
    const genderKey = gender.toLowerCase();
    const translationKey = `contacts.${genderKey}`;
    const translated = t(translationKey);
    // If translation returns the key itself, no translation exists - use original
    return translated === translationKey ? gender : translated;
  };


  const formatRelationshipType = (type: string) => {
    const typeKey = type.toLowerCase().replace(/\s+/g, '_');
    const translationKey = `relationships.types.${typeKey}`;
    const translated = t(translationKey);
    // If translation returns the key itself, no translation exists - use original
    return translated === translationKey ? type : translated;
  };

  if (relationships.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ py: 2, textAlign: 'center' }}>
        {t('relationships.noRelationships')}
      </Typography>
    );
  }

  return (
    <Stack spacing={1.5}>
      {relationships.map((relationship) => {
        // For linked contacts, use their data; otherwise use the stored relationship data
        const linkedContact = relationship.related_contact;
        const displayName = linkedContact 
          ? `${linkedContact.firstname} ${linkedContact.lastname}` 
          : relationship.name;
        const displayGender = linkedContact?.gender || relationship.gender;
        const displayBirthday = linkedContact?.birthday || relationship.birthday;

        return (
          <Paper
            key={relationship.ID}
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
                {linkedContact ? (
                  <Link
                    component="button"
                    variant="subtitle1"
                    onClick={() => handleLinkedContactClick(linkedContact.ID)}
                    sx={{ fontWeight: 500, textAlign: 'left', p: 0 }}
                  >
                    {displayName}
                  </Link>
                ) : (
                  <Typography variant="subtitle1" sx={{ fontWeight: 500 }}>
                    {displayName}
                  </Typography>
                )}
                <Typography variant="body2" color="text.secondary">
                  {formatRelationshipType(relationship.type)}
                  {displayGender && ` · ${formatGender(displayGender)}`}
                  {displayBirthday && ` · ${t('relationships.birthday')}: ${displayBirthday}`}
                </Typography>
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
                <IconButton
                  size="small"
                  onClick={() => onEdit(relationship)}
                  aria-label={t('common.edit')}
                >
                  <EditIcon fontSize="small" />
                </IconButton>
                <IconButton
                  size="small"
                  onClick={() => handleDeleteClick(relationship.ID)}
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
    </Stack>
  );
}
