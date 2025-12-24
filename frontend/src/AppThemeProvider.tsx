import { ReactNode, createContext, useContext, useEffect, useMemo, useState } from "react";
import { ThemeProvider } from "@mui/material/styles";
import CssBaseline from "@mui/material/CssBaseline";
import { darkTheme, lightTheme } from "./theme";

export type ThemePreference = "system" | "light" | "dark";

interface ThemePreferenceContextValue {
  preference: ThemePreference;
  setPreference: (preference: ThemePreference) => void;
  mode: "light" | "dark";
}

const ThemePreferenceContext = createContext<ThemePreferenceContextValue | undefined>(undefined);

const THEME_PREFERENCE_STORAGE_KEY = "themePreference";

const getStoredPreference = (): ThemePreference => {
  if (typeof window === "undefined") {
    return "system";
  }

  const storedValue = window.localStorage.getItem(THEME_PREFERENCE_STORAGE_KEY);
  if (storedValue === "light" || storedValue === "dark" || storedValue === "system") {
    return storedValue;
  }

  return "system";
};

const getSystemPrefersDark = () => {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
    return false;
  }

  return window.matchMedia("(prefers-color-scheme: dark)").matches;
};

export function AppThemeProvider({ children }: { children: ReactNode }) {
  const [preference, setPreference] = useState<ThemePreference>(() => getStoredPreference());
  const [systemPrefersDark, setSystemPrefersDark] = useState(() => getSystemPrefersDark());

  useEffect(() => {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
      return;
    }

    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    const handleChange = (event: MediaQueryListEvent) => setSystemPrefersDark(event.matches);

    if (typeof mediaQuery.addEventListener === "function") {
      mediaQuery.addEventListener("change", handleChange);
    } else {
      mediaQuery.addListener(handleChange);
    }

    return () => {
      if (typeof mediaQuery.removeEventListener === "function") {
        mediaQuery.removeEventListener("change", handleChange);
      } else {
        mediaQuery.removeListener(handleChange);
      }
    };
  }, []);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    window.localStorage.setItem(THEME_PREFERENCE_STORAGE_KEY, preference);
  }, [preference]);

  const mode: "light" | "dark" = preference === "system" ? (systemPrefersDark ? "dark" : "light") : preference;
  const theme = useMemo(() => (mode === "dark" ? darkTheme : lightTheme), [mode]);
  const contextValue = useMemo(
    () => ({
      preference,
      setPreference,
      mode,
    }),
    [preference, mode]
  );

  return (
    <ThemePreferenceContext.Provider value={contextValue}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        {children}
      </ThemeProvider>
    </ThemePreferenceContext.Provider>
  );
}

export const useThemePreference = () => {
  const context = useContext(ThemePreferenceContext);

  if (!context) {
    throw new Error("useThemePreference must be used within AppThemeProvider");
  }

  return context;
};
