// theme.ts
//
// Palette values come from assets/colors/tokens.json (OKLCH-derived; see
// assets/colors/README.md for the conversion math and the hover/active/
// contrastText derivation rules). `light`/`dark` on primary/secondary are
// deliberately omitted where we don't have a bespoke tint -- MUI's own
// augmentColor() fills them in from `main`.
import { createTheme } from "@mui/material/styles";

export const lightTheme = createTheme({
  palette: {
    mode: "light",

    primary: {
      main: "#3E543E", // mycelium
      dark: "#314631", // mycelium hover (also used by MUI for contained-button hover/emphasis)
      contrastText: "#FFFFFF",
    },

    secondary: {
      main: "#97A390", // lichen
      dark: "#879481", // lichen hover
      contrastText: "#30271F", // bark -- verified 5.54:1 (AA); white fails at 2.64:1
    },

    background: {
      default: "#FAF5EA", // bone
      paper: "#EFE7D9", // parchment -- cards/sidebars. Dialogs/menus/popovers use the
      // lighter "paper" token (#FFFFFF) instead, see MuiDialog/MuiPopover/MuiMenu below.
    },

    text: {
      primary: "#30271F", // bark
      secondary: "#595148", // soil
    },

    divider: "#CBC7BF", // hypha

    // Mushroom-themed status colors -- see assets/colors/README.md for the
    // two corrections made before wiring these in (moss/success was too
    // close in hue to mycelium/lichen; inkyCap/info's light value didn't
    // clear 4.5:1 with any label color).
    success: {
      main: "#0D844C", // moss
      dark: "#0B7643", // moss hover
      contrastText: "#FFFFFF",
    },
    warning: {
      main: "#D3A563", // chanterelle
      dark: "#C39553", // chanterelle hover
      contrastText: "#30271F", // bark -- chanterelle is light in both modes, needs a dark label in both
    },
    error: {
      main: "#AD5349", // russula
      dark: "#9C443B", // russula hover
      contrastText: "#FFFFFF",
    },
    info: {
      main: "#7B6B98", // inky-cap (corrected from the original oklch(0.60 0.070 300))
      dark: "#6C5D89", // inky-cap hover
      contrastText: "#FFFFFF",
    },
  },

  shape: {
    borderRadius: 10,
  },

  typography: {
    fontFamily: `"Helvetica", "Helvetica Neue", "Roboto", "Arial", sans-serif`,

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
      color: "#595148", // soil
    },
  },

  components: {
    MuiCard: {
      styleOverrides: {
        root: {
          boxShadow:
            "0px 1px 2px rgba(48, 39, 31, 0.06), 0px 2px 8px rgba(48, 39, 31, 0.04)",
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
          // No explicit color override -- AppBar defaults to color="primary"
          // (mycelium), matching its "brand primary, navigation" role.
          boxShadow: "none",
        },
      },
    },

    // "paper" token (elevated dialogs/floating elements) is one step lighter
    // than background.paper (parchment) -- applied here rather than as the
    // base background.paper, to keep the 3-tier bone/parchment/paper
    // elevation the tokens define instead of collapsing it to MUI's default
    // 2-tier default/paper model.
    MuiDialog: {
      styleOverrides: {
        paper: {
          backgroundColor: "#FFFFFF", // paper
        },
      },
    },
    MuiPopover: {
      styleOverrides: {
        paper: {
          backgroundColor: "#FFFFFF", // paper
        },
      },
    },
    MuiMenu: {
      styleOverrides: {
        paper: {
          backgroundColor: "#FFFFFF", // paper
        },
      },
    },
  },
});

export const darkTheme = createTheme({
  palette: {
    mode: "dark",

    primary: {
      main: "#9EB698", // mycelium (dark mode: bright, not dark -- see contrastText note)
      dark: "#8EA789", // mycelium hover
      // mycelium is bright in dark mode (L 0.75), so a light label fails
      // contrast (1.73:1) -- verified a dark label works instead (7.92:1, AAA).
      contrastText: "#1E1A13", // bone
    },

    secondary: {
      main: "#9BAA94", // lichen
      dark: "#8C9A85", // lichen hover
      contrastText: "#1E1A13", // bone -- verified 7.07:1 (AAA); bark fails at 1.94:1 here
    },

    background: {
      default: "#1E1A13", // bone
      paper: "#2B261B", // parchment
    },

    text: {
      primary: "#EAE4DA", // bark
      secondary: "#B5ADA2", // soil
    },

    divider: "#45423B", // hypha

    success: {
      main: "#349D62", // moss
      dark: "#318F5A", // moss hover
      contrastText: "#1E1A13", // bone -- moss is bright in dark mode, same reasoning as mycelium
    },
    warning: {
      main: "#DDAE6C", // chanterelle
      dark: "#CD9F5D", // chanterelle hover
      contrastText: "#1E1A13", // bone
    },
    error: {
      main: "#C4675D", // russula
      dark: "#B3584E", // russula hover
      contrastText: "#1E1A13", // bone
    },
    info: {
      main: "#9F8FBE", // inky-cap
      dark: "#8F80AE", // inky-cap hover
      contrastText: "#1E1A13", // bone
    },
  },

  shape: {
    borderRadius: 10,
  },

  typography: {
    fontFamily: `"Helvetica", "Helvetica Neue", "Roboto", "Arial", sans-serif`,

    h5: {
      fontWeight: 600,
    },
    h6: {
      fontWeight: 600,
    },
    body2: {
      color: "#B5ADA2", // soil
    },
  },

  components: {
    MuiCard: {
      styleOverrides: {
        root: {
          backgroundImage: "none",
          border: "1px solid #45423B", // hypha
        },
      },
    },

    MuiAppBar: {
      styleOverrides: {
        root: {
          // Deliberately not primary.main here (mycelium is bright in dark
          // mode -- a full-bleed bright light-green top bar would fight the
          // rest of the dark chrome, and tried-and-checked-in dark parchment
          // instead just blended into the cards below it with no brand
          // presence left). Uses light mode's mycelium value directly instead
          // -- a fixed dark-green brand anchor that doesn't flip between
          // modes, same idea as a fixed-color sidebar/header in apps like
          // Slack. Already verified AAA with white text (8.26:1) since it's
          // the exact value light mode's AppBar already uses.
          backgroundColor: "#3E543E", // mycelium (light-mode value, used as-is here)
          color: "#FFFFFF",
          boxShadow: "none",
        },
      },
    },

    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundImage: "none",
        },
      },
    },

    MuiChip: {
      styleOverrides: {
        root: {
          backgroundColor: "#45423B", // hypha
          color: "#EAE4DA", // bark
        },
      },
    },

    MuiDialog: {
      styleOverrides: {
        paper: {
          backgroundColor: "#393226", // paper
        },
      },
    },
    MuiPopover: {
      styleOverrides: {
        paper: {
          backgroundColor: "#393226", // paper
        },
      },
    },
    MuiMenu: {
      styleOverrides: {
        paper: {
          backgroundColor: "#393226", // paper
        },
      },
    },
  },
});
