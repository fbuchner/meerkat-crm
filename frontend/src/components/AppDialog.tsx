import { useState, useCallback } from 'react';
import { Dialog } from '@mui/material';
import { keyframes } from '@mui/system';
import type { DialogProps } from '@mui/material';

const shake = keyframes`
  0%, 100% { transform: translateX(0); }
  20% { transform: translateX(-8px); }
  40% { transform: translateX(8px); }
  60% { transform: translateX(-4px); }
  80% { transform: translateX(4px); }
`;

type AppDialogProps = Omit<DialogProps, 'onClose'> & {
  onClose?: () => void;
};

export default function AppDialog({ onClose, slotProps, ...props }: AppDialogProps) {
  const [shaking, setShaking] = useState(false);

  const handleClose = useCallback(
    (_event: object, reason: 'backdropClick' | 'escapeKeyDown') => {
      if (reason === 'backdropClick') {
        if (!shaking) {
          setShaking(true);
          setTimeout(() => setShaking(false), 400);
        }
        return;
      }
      onClose?.();
    },
    [onClose, shaking]
  );

  return (
    <Dialog
      {...props}
      onClose={handleClose}
      slotProps={{
        ...slotProps,
        paper: {
          ...(slotProps?.paper as object | undefined),
          sx: {
            ...((slotProps?.paper as { sx?: object } | undefined)?.sx),
            animation: shaking ? `${shake} 0.4s ease-in-out` : 'none',
          },
        },
      }}
    />
  );
}
