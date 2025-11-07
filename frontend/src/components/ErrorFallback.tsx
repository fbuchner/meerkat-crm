import React from 'react';
import {
  Box,
  Typography,
  Button,
  Alert,
  AlertTitle,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';

/**
 * Props for ErrorFallback component
 */
interface ErrorFallbackProps {
  /** Error object */
  error?: Error;
  /** Reset function to try again */
  onReset?: () => void;
  /** Title for the error message */
  title?: string;
  /** Custom error message */
  message?: string;
  /** Size variant */
  size?: 'small' | 'medium' | 'large';
  /** Whether to show technical details */
  showDetails?: boolean;
}

/**
 * ErrorFallback component
 * 
 * A simple, inline error fallback UI for use with ErrorBoundary
 * or as a standalone error display component.
 * 
 * @example
 * ```tsx
 * <ErrorBoundary fallback={<ErrorFallback title="Failed to load contacts" />}>
 *   <ContactsList />
 * </ErrorBoundary>
 * ```
 */
export const ErrorFallback: React.FC<ErrorFallbackProps> = ({
  error,
  onReset,
  title = 'Something went wrong',
  message = 'An unexpected error occurred. Please try again.',
  size = 'medium',
  showDetails = false,
}) => {
  const iconSize = size === 'small' ? 32 : size === 'large' ? 64 : 48;
  const titleVariant = size === 'small' ? 'h6' : size === 'large' ? 'h4' : 'h5';
  const bodyVariant = size === 'small' ? 'body2' : 'body1';

  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        p: size === 'small' ? 2 : size === 'large' ? 4 : 3,
        textAlign: 'center',
      }}
    >
      <ErrorOutlineIcon
        sx={{
          fontSize: iconSize,
          color: 'error.main',
          mb: 2,
        }}
      />
      
      <Typography variant={titleVariant} gutterBottom>
        {title}
      </Typography>
      
      <Typography variant={bodyVariant} color="text.secondary" paragraph>
        {message}
      </Typography>

      {onReset && (
        <Button
          variant="contained"
          startIcon={<RefreshIcon />}
          onClick={onReset}
          size={size}
        >
          Try Again
        </Button>
      )}

      {showDetails && error && (
        <Alert severity="error" sx={{ mt: 2, width: '100%', textAlign: 'left' }}>
          <AlertTitle>Technical Details</AlertTitle>
          <Typography variant="caption" component="pre" sx={{ whiteSpace: 'pre-wrap' }}>
            {error.toString()}
            {error.stack && `\n\n${error.stack}`}
          </Typography>
        </Alert>
      )}
    </Box>
  );
};

/**
 * SectionErrorFallback component
 * 
 * A minimal error fallback for small sections or cards.
 * Uses an Alert component for a less intrusive error display.
 */
export const SectionErrorFallback: React.FC<ErrorFallbackProps> = ({
  error,
  onReset,
  title = 'Error loading section',
  message,
}) => {
  return (
    <Alert
      severity="error"
      action={
        onReset && (
          <Button color="inherit" size="small" onClick={onReset}>
            Retry
          </Button>
        )
      }
    >
      <AlertTitle>{title}</AlertTitle>
      {message || error?.message || 'An unexpected error occurred.'}
    </Alert>
  );
};

export default ErrorFallback;
