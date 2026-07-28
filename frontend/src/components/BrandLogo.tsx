import { Box } from '@mui/material';
import { useThemePreference } from '../AppThemeProvider';

interface BrandLogoProps {
  height: number;
}

// mycelium is dark in light mode and bright in dark mode (same asymmetry as
// palette.primary.contrastText throughout the theme) -- the transparent logo
// PNG has to flip with it, or it reads as a near-invisible dark mark on a
// dark background in dark mode.
export default function BrandLogo({ height }: BrandLogoProps) {
  const { mode } = useThemePreference();
  const src = mode === 'dark' ? '/mycorrhizal-logo-dark_512.png' : '/mycorrhizal-logo-light_512.png';

  return (
    <Box
      component="img"
      src={src}
      alt="Mycorrhizal CRM"
      sx={{ height, width: 'auto', flexShrink: 0 }}
    />
  );
}
