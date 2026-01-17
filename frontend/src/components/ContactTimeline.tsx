import { Fragment } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  Box,
  Typography,
  Paper,
  IconButton,
  Link
} from '@mui/material';
import {
  Timeline,
  TimelineItem,
  TimelineSeparator,
  TimelineConnector,
  TimelineContent,
  TimelineDot,
  TimelineOppositeContent
} from '@mui/lab';
import NoteIcon from '@mui/icons-material/Note';
import EventIcon from '@mui/icons-material/Event';
import EditIcon from '@mui/icons-material/Edit';
import { Note } from '../api/notes';
import { Activity } from '../api/activities';
import { useDateFormat } from '../DateFormatProvider';

interface ContactTimelineProps {
  timelineItems: Array<{ type: 'note' | 'activity'; data: Note | Activity; date: string }>;
  onEditItem: (type: 'note' | 'activity', item: Note | Activity) => void;
}

export default function ContactTimeline({ timelineItems, onEditItem }: ContactTimelineProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { formatDate } = useDateFormat();

  if (timelineItems.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary">
        {t('contactDetail.noActivity')}
      </Typography>
    );
  }

  return (
    <Timeline position="right">
      {timelineItems.map((item, index) => {
        const itemDate = new Date(item.date);
        const isValidDate = !isNaN(itemDate.getTime());

        return (
          <TimelineItem key={`${item.type}-${item.data.ID}`}>
            <TimelineOppositeContent color="text.secondary" sx={{ flex: 0.3 }}>
              <Typography variant="caption">
                {isValidDate ? formatDate(item.date) : (item.date || 'N/A')}
              </Typography>
            </TimelineOppositeContent>
            <TimelineSeparator>
              <TimelineDot color={item.type === 'note' ? 'primary' : 'secondary'}>
                {item.type === 'note' ? <NoteIcon fontSize="small" /> : <EventIcon fontSize="small" />}
              </TimelineDot>
              {index < timelineItems.length - 1 && <TimelineConnector />}
            </TimelineSeparator>
            <TimelineContent>
              <Paper
                elevation={1}
                sx={{
                  p: 1.5,
                  position: 'relative',
                  '&:hover .edit-icon': {
                    opacity: 1
                  }
                }}
              >
                <Typography variant="subtitle2" sx={{ fontWeight: 500 }}>
                  {item.type === 'note' ? t('contactDetail.note') : (item.data as Activity).title}
                </Typography>
                <Typography variant="body2" sx={{ mt: 1 }}>
                  {item.type === 'note'
                    ? (item.data as Note).content
                    : (item.data as Activity).description || t('contactDetail.noDescription')}
                </Typography>
                {item.type === 'activity' && (item.data as Activity).location && (
                  <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
                    📍 {(item.data as Activity).location}
                  </Typography>
                )}
                {item.type === 'activity' && (item.data as Activity).contacts && (item.data as Activity).contacts!.length > 0 && (
                  <Box sx={{ mt: 1.5, display: 'flex', flexWrap: 'wrap', alignItems: 'center' }}>
                    <Typography variant="caption" color="text.secondary" sx={{ mr: 0.5 }}>
                      👥
                    </Typography>
                    {(item.data as Activity).contacts!.map((activityContact, idx) => (
                      <Fragment key={activityContact.ID}>
                        <Link
                          component="button"
                          variant="caption"
                          onClick={(e) => {
                            e.preventDefault();
                            navigate(`/contacts/${activityContact.ID}`);
                          }}
                          sx={{
                            cursor: 'pointer',
                            textDecoration: 'none',
                            '&:hover': { textDecoration: 'underline' }
                          }}
                        >
                          {activityContact.firstname}{' '}
                          {activityContact.nickname ? `"${activityContact.nickname}" ` : ''}
                          {activityContact.lastname}
                        </Link>
                        {idx < (item.data as Activity).contacts!.length - 1 && (
                          <Typography variant="caption" color="text.secondary">,&nbsp;</Typography>
                        )}
                      </Fragment>
                    ))}
                  </Box>
                )}

                <IconButton
                  className="edit-icon"
                  size="small"
                  onClick={() => onEditItem(item.type, item.data)}
                  sx={{
                    position: 'absolute',
                    top: 8,
                    right: 8,
                    opacity: 0,
                    transition: 'opacity 0.2s'
                  }}
                >
                  <EditIcon fontSize="small" />
                </IconButton>
              </Paper>
            </TimelineContent>
          </TimelineItem>
        );
      })}
    </Timeline>
  );
}
