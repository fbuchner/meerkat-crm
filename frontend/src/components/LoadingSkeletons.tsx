import { Box, Card, Stack, Skeleton, Paper } from '@mui/material';

/**
 * Skeleton loader for contact cards in the contacts list
 */
export const ContactCardSkeleton = () => {
  return (
    <Card sx={{ display: 'flex', alignItems: 'center', p: 1 }}>
      <Skeleton animation="wave" variant="circular" width={56} height={56} sx={{ mr: 2 }} />
      <Box sx={{ flex: 1 }}>
        <Skeleton animation="wave" variant="text" width="60%" height={28} />
        <Skeleton animation="wave" variant="text" width="40%" height={20} />
      </Box>
    </Card>
  );
};

/**
 * Renders multiple contact card skeletons
 */
export const ContactListSkeleton = ({ count = 5 }: { count?: number }) => {
  return (
    <Stack spacing={2}>
      {Array.from({ length: count }).map((_, index) => (
        <ContactCardSkeleton key={index} />
      ))}
    </Stack>
  );
};

/**
 * Skeleton loader for the contact detail header
 */
export const ContactDetailHeaderSkeleton = () => {
  return (
    <Card>
      <Box sx={{ display: 'flex', p: 3, alignItems: 'flex-start' }}>
        <Skeleton animation="wave" variant="circular" width={120} height={120} sx={{ mr: 3 }} />
        <Box sx={{ flex: 1 }}>
          <Skeleton animation="wave" variant="text" width="50%" height={40} sx={{ mb: 1 }} />
          <Skeleton animation="wave" variant="text" width="30%" height={24} sx={{ mb: 2 }} />
          <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
            <Skeleton animation="wave" variant="rounded" width={80} height={32} />
            <Skeleton animation="wave" variant="rounded" width={80} height={32} />
          </Stack>
          <Skeleton animation="wave" variant="text" width="70%" height={20} />
          <Skeleton animation="wave" variant="text" width="60%" height={20} />
        </Box>
      </Box>
    </Card>
  );
};

/**
 * Skeleton loader for timeline items in contact detail
 */
export const TimelineItemSkeleton = () => {
  return (
    <Box sx={{ display: 'flex', mb: 3 }}>
      <Box sx={{ mr: 2, display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
        <Skeleton animation="wave" variant="circular" width={24} height={24} />
        <Skeleton animation="wave" variant="rectangular" width={2} height={60} sx={{ mt: 1 }} />
      </Box>
      <Box sx={{ flex: 1 }}>
        <Skeleton animation="wave" variant="text" width="20%" height={20} sx={{ mb: 1 }} />
        <Paper sx={{ p: 2 }}>
          <Skeleton animation="wave" variant="text" width="40%" height={24} sx={{ mb: 1 }} />
          <Skeleton animation="wave" variant="text" width="100%" height={20} />
          <Skeleton animation="wave" variant="text" width="90%" height={20} />
        </Paper>
      </Box>
    </Box>
  );
};

/**
 * Renders multiple timeline item skeletons
 */
export const TimelineSkeleton = ({ count = 3 }: { count?: number }) => {
  return (
    <Box>
      {Array.from({ length: count }).map((_, index) => (
        <TimelineItemSkeleton key={index} />
      ))}
    </Box>
  );
};

/**
 * Skeleton loader for list items (activities, notes)
 */
export const ListItemSkeleton = () => {
  return (
    <Card sx={{ p: 2, mb: 2 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
        <Skeleton animation="wave" variant="text" width="40%" height={24} />
        <Skeleton animation="wave" variant="text" width="20%" height={20} />
      </Box>
      <Skeleton animation="wave" variant="text" width="100%" height={20} />
      <Skeleton animation="wave" variant="text" width="80%" height={20} />
      <Box sx={{ mt: 2, display: 'flex', gap: 1 }}>
        <Skeleton animation="wave" variant="rounded" width={60} height={24} />
        <Skeleton animation="wave" variant="rounded" width={60} height={24} />
      </Box>
    </Card>
  );
};

/**
 * Renders multiple list item skeletons
 */
export const ListSkeleton = ({ count = 5 }: { count?: number }) => {
  return (
    <Box>
      {Array.from({ length: count }).map((_, index) => (
        <ListItemSkeleton key={index} />
      ))}
    </Box>
  );
};

/**
 * Skeleton loader for the photo gallery
 */
export const PhotoGallerySkeleton = () => {
  return (
    <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(150px, 1fr))', gap: 2 }}>
      {Array.from({ length: 6 }).map((_, index) => (
        <Skeleton key={index} variant="rectangular" height={150} sx={{ borderRadius: 1 }} />
      ))}
    </Box>
  );
};

/**
 * Skeleton loader for form inputs
 */
export const FormSkeleton = ({ fields = 4 }: { fields?: number }) => {
  return (
    <Stack spacing={3}>
      {Array.from({ length: fields }).map((_, index) => (
        <Box key={index}>
          <Skeleton animation="wave" variant="text" width="30%" height={20} sx={{ mb: 1 }} />
          <Skeleton animation="wave" variant="rectangular" width="100%" height={56} sx={{ borderRadius: 1 }} />
        </Box>
      ))}
      <Skeleton animation="wave" variant="rectangular" width="100%" height={42} sx={{ borderRadius: 1, mt: 2 }} />
    </Stack>
  );
};

/**
 * Skeleton loader for auth forms (login/register)
 */
export const AuthFormSkeleton = () => {
  return (
    <Box sx={{ maxWidth: 400, mx: 'auto', p: 3 }}>
      <Skeleton animation="wave" variant="text" width="60%" height={48} sx={{ mb: 3, mx: 'auto' }} />
      <FormSkeleton fields={3} />
      <Box sx={{ mt: 2, textAlign: 'center' }}>
        <Skeleton animation="wave" variant="text" width="40%" height={20} sx={{ mx: 'auto' }} />
      </Box>
    </Box>
  );
};

/**
 * Skeleton loader for table rows
 */
export const TableRowSkeleton = ({ columns = 4 }: { columns?: number }) => {
  return (
    <Box sx={{ display: 'flex', gap: 2, p: 2, borderBottom: '1px solid', borderColor: 'divider' }}>
      {Array.from({ length: columns }).map((_, index) => (
        <Box key={index} sx={{ flex: index === 0 ? 2 : 1 }}>
          <Skeleton animation="wave" variant="text" width="80%" height={20} />
        </Box>
      ))}
    </Box>
  );
};

/**
 * Skeleton loader for complete tables
 */
export const TableSkeleton = ({ rows = 5, columns = 4 }: { rows?: number; columns?: number }) => {
  return (
    <Paper>
      {/* Table Header */}
      <Box sx={{ display: 'flex', gap: 2, p: 2, bgcolor: 'action.hover' }}>
        {Array.from({ length: columns }).map((_, index) => (
          <Box key={index} sx={{ flex: index === 0 ? 2 : 1 }}>
            <Skeleton animation="wave" variant="text" width="60%" height={24} />
          </Box>
        ))}
      </Box>
      {/* Table Rows */}
      {Array.from({ length: rows }).map((_, index) => (
        <TableRowSkeleton key={index} columns={columns} />
      ))}
    </Paper>
  );
};

/**
 * Skeleton loader for grid items
 */
export const GridItemSkeleton = () => {
  return (
    <Card>
      <Skeleton animation="wave" variant="rectangular" height={200} />
      <Box sx={{ p: 2 }}>
        <Skeleton animation="wave" variant="text" width="80%" height={24} sx={{ mb: 1 }} />
        <Skeleton animation="wave" variant="text" width="100%" height={20} />
        <Skeleton animation="wave" variant="text" width="90%" height={20} />
      </Box>
    </Card>
  );
};

/**
 * Skeleton loader for grid layouts
 */
export const GridSkeleton = ({ items = 6, columns = 3 }: { items?: number; columns?: 2 | 3 | 4 }) => {
  return (
    <Box sx={{ display: 'grid', gridTemplateColumns: `repeat(${columns}, 1fr)`, gap: 3 }}>
      {Array.from({ length: items }).map((_, index) => (
        <GridItemSkeleton key={index} />
      ))}
    </Box>
  );
};

/**
 * Skeleton loader for statistics/dashboard cards
 */
export const StatCardSkeleton = () => {
  return (
    <Card sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 2 }}>
        <Box sx={{ flex: 1 }}>
          <Skeleton animation="wave" variant="text" width="60%" height={20} />
          <Skeleton animation="wave" variant="text" width="40%" height={48} sx={{ mt: 1 }} />
        </Box>
        <Skeleton animation="wave" variant="circular" width={48} height={48} />
      </Box>
      <Skeleton animation="wave" variant="text" width="50%" height={16} />
    </Card>
  );
};

/**
 * Skeleton loader for dashboard with stat cards
 */
export const DashboardSkeleton = () => {
  return (
    <Box>
      <Skeleton animation="wave" variant="text" width="40%" height={48} sx={{ mb: 3 }} />
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))', gap: 3 }}>
        {Array.from({ length: 4 }).map((_, index) => (
          <StatCardSkeleton key={index} />
        ))}
      </Box>
      <Box sx={{ mt: 4 }}>
        <Skeleton animation="wave" variant="text" width="30%" height={32} sx={{ mb: 2 }} />
        <ListSkeleton count={5} />
      </Box>
    </Box>
  );
};

/**
 * Skeleton loader for profile/avatar with details
 */
export const ProfileSkeleton = () => {
  return (
    <Card sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', mb: 3 }}>
        <Skeleton animation="wave" variant="circular" width={80} height={80} sx={{ mr: 3 }} />
        <Box sx={{ flex: 1 }}>
          <Skeleton animation="wave" variant="text" width="50%" height={32} sx={{ mb: 1 }} />
          <Skeleton animation="wave" variant="text" width="70%" height={20} />
        </Box>
      </Box>
      <Stack spacing={2}>
        <Skeleton animation="wave" variant="text" width="100%" height={20} />
        <Skeleton animation="wave" variant="text" width="90%" height={20} />
        <Skeleton animation="wave" variant="text" width="80%" height={20} />
      </Stack>
    </Card>
  );
};

/**
 * Skeleton loader for search bars
 */
export const SearchBarSkeleton = () => {
  return (
    <Box sx={{ display: 'flex', gap: 2, mb: 3 }}>
      <Skeleton animation="wave" variant="rectangular" sx={{ flex: 1, height: 56, borderRadius: 1 }} />
      <Skeleton animation="wave" variant="rectangular" width={120} height={56} sx={{ borderRadius: 1 }} />
    </Box>
  );
};

/**
 * Skeleton loader for page header with actions
 */
export const PageHeaderSkeleton = () => {
  return (
    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
      <Skeleton animation="wave" variant="text" width="30%" height={48} />
      <Box sx={{ display: 'flex', gap: 2 }}>
        <Skeleton animation="wave" variant="rectangular" width={100} height={40} sx={{ borderRadius: 1 }} />
        <Skeleton animation="wave" variant="rectangular" width={120} height={40} sx={{ borderRadius: 1 }} />
      </Box>
    </Box>
  );
};

/**
 * Full page skeleton with header and content
 */
export const PageSkeleton = ({ contentType = 'list' }: { contentType?: 'list' | 'grid' | 'table' | 'form' }) => {
  return (
    <Box>
      <PageHeaderSkeleton />
      {contentType === 'list' && <ListSkeleton count={6} />}
      {contentType === 'grid' && <GridSkeleton items={6} columns={3} />}
      {contentType === 'table' && <TableSkeleton rows={8} columns={5} />}
      {contentType === 'form' && <FormSkeleton fields={6} />}
    </Box>
  );
};
