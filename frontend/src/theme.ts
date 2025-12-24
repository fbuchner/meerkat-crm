// theme.ts
import { createTheme } from "@mui/material/styles";

export const theme = createTheme({
  palette: {
    mode: "light",

    primary: {
      main: "#2563EB",
      light: "#DBEAFE",
      dark: "#1E40AF",
      contrastText: "#FFFFFF",
    },

    secondary: {
      main: "#14B8A6",
      light: "#99F6E4",
      dark: "#0F766E",
    },

    background: {
      default: "#F8FAFC",
      paper: "#FFFFFF",
    },

    text: {
      primary: "#0F172A",
      secondary: "#475569",
    },

    divider: "#E2E8F0",

    success: {
      main: "#16A34A",
    },
    warning: {
      main: "#F59E0B",
    },
    error: {
      main: "#DC2626",
    },
  },

  shape: {
    borderRadius: 10,
  },

  typography: {
    fontFamily: `"Inter", "Roboto", "Helvetica", "Arial", sans-serif`,

    h5: {
      fontWeight: 600,
    },
    h6: {
      fontWeight: 600,
    },
    subtitle1: {
      fontWeight: 500,
    },
    body2: {
      color: "#475569",
    },
  },

  components: {
    MuiCard: {
      styleOverrides: {
        root: {
          boxShadow:
            "0px 1px 2px rgba(15, 23, 42, 0.06), 0px 2px 8px rgba(15, 23, 42, 0.04)",
        },
      },
    },

    MuiChip: {
      styleOverrides: {
        root: {
          fontWeight: 500,
        },
      },
    },

    MuiAppBar: {
      styleOverrides: {
        root: {
          boxShadow: "none",
        },
      },
    },
  },
});
