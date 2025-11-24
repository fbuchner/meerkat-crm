import { useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import Cropper, { Area } from 'react-easy-crop';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  Slider,
  Typography,
  CircularProgress,
  Alert
} from '@mui/material';
import ZoomInIcon from '@mui/icons-material/ZoomIn';
import CloudUploadIcon from '@mui/icons-material/CloudUpload';

interface ProfilePictureUploadDialogProps {
  open: boolean;
  onClose: () => void;
  onUpload: (croppedImageBlob: Blob) => Promise<void>;
}

// Helper function to create a cropped image blob
async function getCroppedImg(
  imageSrc: string,
  pixelCrop: Area,
  outputSize: number = 400
): Promise<Blob> {
  const image = await createImage(imageSrc);
  const canvas = document.createElement('canvas');
  const ctx = canvas.getContext('2d');

  if (!ctx) {
    throw new Error('Could not get canvas context');
  }

  // Set canvas size to desired output size
  canvas.width = outputSize;
  canvas.height = outputSize;

  // Draw the cropped image
  ctx.drawImage(
    image,
    pixelCrop.x,
    pixelCrop.y,
    pixelCrop.width,
    pixelCrop.height,
    0,
    0,
    outputSize,
    outputSize
  );

  // Return as blob
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => {
        if (blob) {
          resolve(blob);
        } else {
          reject(new Error('Canvas is empty'));
        }
      },
      'image/jpeg',
      0.9
    );
  });
}

// Helper function to load an image
function createImage(url: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.addEventListener('load', () => resolve(image));
    image.addEventListener('error', (error) => reject(error));
    image.src = url;
  });
}

export default function ProfilePictureUploadDialog({
  open,
  onClose,
  onUpload
}: ProfilePictureUploadDialogProps) {
  const { t } = useTranslation();
  const [imageSrc, setImageSrc] = useState<string | null>(null);
  const [crop, setCrop] = useState({ x: 0, y: 0 });
  const [zoom, setZoom] = useState(1);
  const [croppedAreaPixels, setCroppedAreaPixels] = useState<Area | null>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB

  const onCropComplete = useCallback((_croppedArea: Area, croppedAreaPixels: Area) => {
    setCroppedAreaPixels(croppedAreaPixels);
  }, []);

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    setError(null);

    // Validate file type
    if (!file.type.startsWith('image/')) {
      setError(t('profilePicture.invalidFileType'));
      return;
    }

    // Validate file size
    if (file.size > MAX_FILE_SIZE) {
      setError(t('profilePicture.fileTooLarge'));
      return;
    }

    const reader = new FileReader();
    reader.addEventListener('load', () => {
      setImageSrc(reader.result as string);
      setCrop({ x: 0, y: 0 });
      setZoom(1);
    });
    reader.readAsDataURL(file);
  };

  const handleUpload = async () => {
    if (!imageSrc || !croppedAreaPixels) return;

    setUploading(true);
    setError(null);

    try {
      const croppedImage = await getCroppedImg(imageSrc, croppedAreaPixels);
      await onUpload(croppedImage);
      handleClose();
    } catch (err) {
      console.error('Error uploading profile picture:', err);
      setError(t('profilePicture.uploadError'));
    } finally {
      setUploading(false);
    }
  };

  const handleClose = () => {
    setImageSrc(null);
    setCrop({ x: 0, y: 0 });
    setZoom(1);
    setCroppedAreaPixels(null);
    setError(null);
    onClose();
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>{t('profilePicture.title')}</DialogTitle>
      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {!imageSrc ? (
          // File selection view
          <Box
            sx={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              minHeight: 300,
              border: '2px dashed',
              borderColor: 'divider',
              borderRadius: 2,
              p: 4,
              cursor: 'pointer',
              '&:hover': {
                borderColor: 'primary.main',
                bgcolor: 'action.hover'
              }
            }}
            component="label"
          >
            <input
              type="file"
              accept="image/*"
              onChange={handleFileSelect}
              style={{ display: 'none' }}
            />
            <CloudUploadIcon sx={{ fontSize: 64, color: 'text.secondary', mb: 2 }} />
            <Typography variant="h6" color="text.secondary">
              {t('profilePicture.selectImage')}
            </Typography>
            <Typography variant="body2" color="text.secondary">
              {t('profilePicture.dragOrClick')}
            </Typography>
          </Box>
        ) : (
          // Cropping view
          <Box>
            <Box
              sx={{
                position: 'relative',
                width: '100%',
                height: 350,
                bgcolor: 'black',
                borderRadius: 1,
                overflow: 'hidden'
              }}
            >
              <Cropper
                image={imageSrc}
                crop={crop}
                zoom={zoom}
                aspect={1}
                onCropChange={setCrop}
                onZoomChange={setZoom}
                onCropComplete={onCropComplete}
                cropShape="round"
                showGrid={false}
              />
            </Box>

            {/* Zoom slider */}
            <Box sx={{ display: 'flex', alignItems: 'center', mt: 2, px: 2 }}>
              <ZoomInIcon sx={{ mr: 2, color: 'text.secondary' }} />
              <Slider
                value={zoom}
                min={1}
                max={3}
                step={0.1}
                onChange={(_, value) => setZoom(value as number)}
                aria-label={t('profilePicture.zoom')}
              />
            </Box>

            {/* Change image button */}
            <Box sx={{ display: 'flex', justifyContent: 'center', mt: 1 }}>
              <Button
                component="label"
                size="small"
                variant="text"
              >
                {t('profilePicture.changeImage')}
                <input
                  type="file"
                  accept="image/*"
                  onChange={handleFileSelect}
                  style={{ display: 'none' }}
                />
              </Button>
            </Box>
          </Box>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} disabled={uploading}>
          {t('common.cancel')}
        </Button>
        <Button
          onClick={handleUpload}
          variant="contained"
          disabled={!imageSrc || uploading}
          startIcon={uploading ? <CircularProgress size={20} /> : undefined}
        >
          {uploading ? t('profilePicture.uploading') : t('common.save')}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
