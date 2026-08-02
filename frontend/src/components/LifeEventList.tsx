import { Box, Typography, Chip, IconButton, Paper, Tooltip } from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import CakeIcon from '@mui/icons-material/Cake';
import { useTranslation } from 'react-i18next';
import { LifeEvent, partialDateDisplay } from '../api/lifeEvents';
import { Contact } from '../api/contacts';

interface LifeEventListProps {
  events: LifeEvent[];
  contactsByUid: Map<string, Contact>;
  onEdit: (event: LifeEvent) => void;
  onDelete: (id: string) => void;
}

export default function LifeEventList({
  events,
  contactsByUid,
  onEdit,
  onDelete,
}: LifeEventListProps) {
  const { t } = useTranslation();

  if (events.length === 0) {
    return (
      <Paper sx={{ p: 3, textAlign: 'center' }}>
        <Typography variant="body1" color="text.secondary">
          {t('lifeEvent.empty')}
        </Typography>
      </Paper>
    );
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      {events.map((event) => (
        <Paper key={event.id} sx={{ p: 2, '&:hover .life-event-actions': { opacity: 1 } }}>
          <Box display="flex" justifyContent="space-between" alignItems="flex-start">
            <Box>
              <Box display="flex" alignItems="center" gap={1} mb={0.5}>
                <CakeIcon fontSize="small" color="action" />
                <Typography variant="subtitle2">
                  {t(`lifeEvent.types.${event.type}`, event.type)}
                </Typography>
                <Chip
                  label={partialDateDisplay(event.date)}
                  size="small"
                  variant="outlined"
                />
                {event.remind && (
                  <Tooltip title={t('lifeEvent.remindTooltip')}>
                    <Chip label={t('lifeEvent.remind')} size="small" color="primary" />
                  </Tooltip>
                )}
              </Box>
              {event.description && (
                <Typography variant="body2" color="text.secondary" sx={{ whiteSpace: 'pre-wrap', mb: 0.5 }}>
                  {event.description}
                </Typography>
              )}
              {event.related_entity_ids && event.related_entity_ids.length > 0 && (
                <Box display="flex" flexWrap="wrap" gap={0.5}>
                  {event.related_entity_ids.map((uid) => {
                    const c = contactsByUid.get(uid);
                    return (
                      <Chip
                        key={uid}
                        label={c ? `${c.firstname} ${c.lastname}` : uid}
                        size="small"
                        variant="outlined"
                      />
                    );
                  })}
                </Box>
              )}
            </Box>
            <Box
              className="life-event-actions"
              sx={{ opacity: 0, transition: 'opacity 0.2s', display: 'flex', gap: 1 }}
            >
              <IconButton size="small" onClick={() => onEdit(event)}>
                <EditIcon fontSize="small" />
              </IconButton>
              <IconButton size="small" onClick={() => onDelete(event.id)}>
                <DeleteIcon fontSize="small" />
              </IconButton>
            </Box>
          </Box>
        </Paper>
      ))}
    </Box>
  );
}
