import { ReactNode, createContext, useContext, useEffect, useMemo, useState, useCallback } from "react";

export type DateFormat = "eu" | "us";

interface DateFormatContextValue {
  dateFormat: DateFormat;
  setDateFormat: (format: DateFormat) => void;
  formatDate: (dateString: string) => string;
  formatBirthday: (birthday: string, includeAge?: boolean) => string;
  formatBirthdayForInput: (birthday: string) => string;
  parseBirthdayInput: (input: string) => string | null;
  getBirthdayPlaceholder: () => string;
  getBirthdayFormatHint: () => string;
}

const DateFormatContext = createContext<DateFormatContextValue | undefined>(undefined);

const DATE_FORMAT_STORAGE_KEY = "dateFormat";

const getStoredFormat = (): DateFormat => {
  if (typeof window === "undefined") {
    return "eu";
  }

  const storedValue = window.localStorage.getItem(DATE_FORMAT_STORAGE_KEY);
  if (storedValue === "eu" || storedValue === "us") {
    return storedValue;
  }

  return "eu";
};

/**
 * Format a standard date (ISO format) to the user's preferred display format
 */
function formatDateWithFormat(dateString: string, format: DateFormat): string {
  if (!dateString) return '';
  
  const date = new Date(dateString);
  if (isNaN(date.getTime())) return dateString;
  
  const day = String(date.getDate()).padStart(2, '0');
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const year = date.getFullYear();
  
  if (format === "eu") {
    return `${day}.${month}.${year}`;
  } else {
    return `${month}/${day}/${year}`;
  }
}

/**
 * Format a birthday (YYYY-MM-DD or --MM-DD) to the user's preferred display format
 * Optionally includes age calculation
 */
function formatBirthdayWithFormat(birthday: string, format: DateFormat, includeAge: boolean = false): string {
  if (!birthday) return '';
  
  // Check if it's a year-less birthday (starts with --)
  if (birthday.startsWith('--')) {
    // --MM-DD format
    const month = birthday.substring(2, 4);
    const day = birthday.substring(5, 7);
    
    if (format === "eu") {
      return `${day}.${month}.`;
    } else {
      return `${month}/${day}`;
    }
  }

  // YYYY-MM-DD format
  const parts = birthday.split('-');
  if (parts.length === 3) {
    const year = parts[0];
    const month = parts[1];
    const day = parts[2];
    
    let dateStr: string;
    if (format === "eu") {
      dateStr = `${day}.${month}.${year}`;
    } else {
      dateStr = `${month}/${day}/${year}`;
    }
    
    // Calculate age if requested and year is valid
    if (includeAge && year && year.length === 4) {
      const birthYear = parseInt(year, 10);
      if (!isNaN(birthYear)) {
        const today = new Date();
        const birthDate = new Date(birthYear, parseInt(month, 10) - 1, parseInt(day, 10));
        let age = today.getFullYear() - birthYear;
        
        // Adjust if birthday hasn't occurred yet this year
        if (today < new Date(today.getFullYear(), birthDate.getMonth(), birthDate.getDate())) {
          age--;
        }
        
        if (age >= 0) {
          return `${dateStr} (${age})`;
        }
      }
    }
    
    return dateStr;
  }

  return birthday; // Return as-is if format doesn't match
}

/**
 * Format a birthday for editing (convert ISO to display format)
 */
function formatBirthdayForInputWithFormat(birthday: string, format: DateFormat): string {
  if (!birthday) return '';
  
  // Check if it's a year-less birthday (starts with --)
  if (birthday.startsWith('--')) {
    const month = birthday.substring(2, 4);
    const day = birthday.substring(5, 7);
    
    if (format === "eu") {
      return `${day}.${month}.`;
    } else {
      return `${month}/${day}`;
    }
  }

  // YYYY-MM-DD format
  const parts = birthday.split('-');
  if (parts.length === 3) {
    const year = parts[0];
    const month = parts[1];
    const day = parts[2];
    
    if (format === "eu") {
      return `${day}.${month}.${year}`;
    } else {
      return `${month}/${day}/${year}`;
    }
  }

  return birthday;
}

/**
 * Parse user input in display format back to ISO format for storage
 * Returns null if input is invalid
 */
function parseBirthdayInputWithFormat(input: string, format: DateFormat): string | null {
  if (!input || input.trim() === '') return '';
  
  const trimmed = input.trim();
  
  // Also accept ISO format directly (YYYY-MM-DD or --MM-DD)
  const isoFullDateRegex = /^\d{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$/;
  const isoYearlessRegex = /^--(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$/;
  if (isoFullDateRegex.test(trimmed) || isoYearlessRegex.test(trimmed)) {
    return trimmed;
  }
  
  if (format === "eu") {
    // EU format: DD.MM.YYYY or DD.MM.
    // Full date with year
    const euFullMatch = trimmed.match(/^(\d{1,2})\.(\d{1,2})\.(\d{4})$/);
    if (euFullMatch) {
      const day = euFullMatch[1].padStart(2, '0');
      const month = euFullMatch[2].padStart(2, '0');
      const year = euFullMatch[3];
      
      // Validate date components
      const dayNum = parseInt(day, 10);
      const monthNum = parseInt(month, 10);
      if (monthNum < 1 || monthNum > 12 || dayNum < 1 || dayNum > 31) {
        return null;
      }
      
      return `${year}-${month}-${day}`;
    }
    
    // Year-less format: DD.MM. or DD.MM
    const euYearlessMatch = trimmed.match(/^(\d{1,2})\.(\d{1,2})\.?$/);
    if (euYearlessMatch) {
      const day = euYearlessMatch[1].padStart(2, '0');
      const month = euYearlessMatch[2].padStart(2, '0');
      
      // Validate date components
      const dayNum = parseInt(day, 10);
      const monthNum = parseInt(month, 10);
      if (monthNum < 1 || monthNum > 12 || dayNum < 1 || dayNum > 31) {
        return null;
      }
      
      return `--${month}-${day}`;
    }
  } else {
    // US format: MM/DD/YYYY or MM/DD
    // Full date with year
    const usFullMatch = trimmed.match(/^(\d{1,2})\/(\d{1,2})\/(\d{4})$/);
    if (usFullMatch) {
      const month = usFullMatch[1].padStart(2, '0');
      const day = usFullMatch[2].padStart(2, '0');
      const year = usFullMatch[3];
      
      // Validate date components
      const dayNum = parseInt(day, 10);
      const monthNum = parseInt(month, 10);
      if (monthNum < 1 || monthNum > 12 || dayNum < 1 || dayNum > 31) {
        return null;
      }
      
      return `${year}-${month}-${day}`;
    }
    
    // Year-less format: MM/DD
    const usYearlessMatch = trimmed.match(/^(\d{1,2})\/(\d{1,2})$/);
    if (usYearlessMatch) {
      const month = usYearlessMatch[1].padStart(2, '0');
      const day = usYearlessMatch[2].padStart(2, '0');
      
      // Validate date components
      const dayNum = parseInt(day, 10);
      const monthNum = parseInt(month, 10);
      if (monthNum < 1 || monthNum > 12 || dayNum < 1 || dayNum > 31) {
        return null;
      }
      
      return `--${month}-${day}`;
    }
  }
  
  return null;
}

export function DateFormatProvider({ children }: { children: ReactNode }) {
  const [dateFormat, setDateFormat] = useState<DateFormat>(() => getStoredFormat());

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    window.localStorage.setItem(DATE_FORMAT_STORAGE_KEY, dateFormat);
  }, [dateFormat]);

  const formatDate = useCallback(
    (dateString: string) => formatDateWithFormat(dateString, dateFormat),
    [dateFormat]
  );

  const formatBirthday = useCallback(
    (birthday: string, includeAge: boolean = false) => formatBirthdayWithFormat(birthday, dateFormat, includeAge),
    [dateFormat]
  );

  const formatBirthdayForInput = useCallback(
    (birthday: string) => formatBirthdayForInputWithFormat(birthday, dateFormat),
    [dateFormat]
  );

  const parseBirthdayInput = useCallback(
    (input: string) => parseBirthdayInputWithFormat(input, dateFormat),
    [dateFormat]
  );

  const getBirthdayPlaceholder = useCallback(
    () => dateFormat === "eu" ? "DD.MM.YYYY" : "MM/DD/YYYY",
    [dateFormat]
  );

  const getBirthdayFormatHint = useCallback(
    () => dateFormat === "eu" 
      ? "DD.MM.YYYY (year optional, e.g., 30.04.1990 or 30.04.)" 
      : "MM/DD/YYYY (year optional, e.g., 04/30/1990 or 04/30)",
    [dateFormat]
  );

  const contextValue = useMemo(
    () => ({
      dateFormat,
      setDateFormat,
      formatDate,
      formatBirthday,
      formatBirthdayForInput,
      parseBirthdayInput,
      getBirthdayPlaceholder,
      getBirthdayFormatHint,
    }),
    [dateFormat, formatDate, formatBirthday, formatBirthdayForInput, parseBirthdayInput, getBirthdayPlaceholder, getBirthdayFormatHint]
  );

  return (
    <DateFormatContext.Provider value={contextValue}>
      {children}
    </DateFormatContext.Provider>
  );
}

export const useDateFormat = () => {
  const context = useContext(DateFormatContext);

  if (!context) {
    throw new Error("useDateFormat must be used within DateFormatProvider");
  }

  return context;
};
