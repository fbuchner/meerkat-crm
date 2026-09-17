import { Reminder, ReminderFormData } from '../api/reminders';

// Extracts month (0-indexed) and day from a stored birthday, which is either
// a full ISO date ("YYYY-MM-DD") or a year-less date ("--MM-DD").
export function parseBirthdayMonthDay(birthdayISO: string): { month: number; day: number } | undefined {
  if (!birthdayISO) return undefined;

  let month: number;
  let day: number;

  if (birthdayISO.startsWith('--')) {
    month = parseInt(birthdayISO.substring(2, 4), 10) - 1;
    day = parseInt(birthdayISO.substring(5, 7), 10);
  } else {
    const parts = birthdayISO.split('-');
    if (parts.length !== 3) return undefined;
    month = parseInt(parts[1], 10) - 1;
    day = parseInt(parts[2], 10);
  }

  if (isNaN(month) || isNaN(day)) return undefined;
  return { month, day };
}

// Next occurrence (this year or next) of the given birthday, at 09:00 local time.
export function computeNextBirthdayOccurrence(birthdayISO: string): Date | undefined {
  const monthDay = parseBirthdayMonthDay(birthdayISO);
  if (!monthDay) return undefined;

  const today = new Date();
  const nextBirthday = new Date(today.getFullYear(), monthDay.month, monthDay.day);

  if (nextBirthday < today) {
    nextBirthday.setFullYear(today.getFullYear() + 1);
  }

  nextBirthday.setHours(9, 0, 0, 0);
  return nextBirthday;
}

export function buildBirthdayReminderPayload(
  contactId: number,
  birthdayISO: string,
  message: string
): ReminderFormData | undefined {
  const nextOccurrence = computeNextBirthdayOccurrence(birthdayISO);
  if (!nextOccurrence) return undefined;

  return {
    message,
    by_mail: true,
    remind_at: nextOccurrence.toISOString(),
    recurrence: 'yearly',
    reoccur_from_completion: false,
    contact_id: contactId
  };
}

// Heuristic match: a birthday reminder is a yearly, calendar-pinned reminder
// whose month/day matches the contact's birthday. There is no dedicated
// "type" field on Reminder, so this convention is relied on across the app
// (see backend/services/monica_import_service.go's isBirthdayReminder).
export function findBirthdayReminder(reminders: Reminder[], birthdayISO: string): Reminder | undefined {
  const monthDay = parseBirthdayMonthDay(birthdayISO);
  if (!monthDay) return undefined;

  const matches = reminders.filter((reminder) => {
    if (reminder.recurrence !== 'yearly' || reminder.reoccur_from_completion) return false;
    const remindAt = new Date(reminder.remind_at);
    return remindAt.getMonth() === monthDay.month && remindAt.getDate() === monthDay.day;
  });

  if (matches.length === 0) return undefined;

  return matches.reduce((earliest, current) =>
    new Date(current.remind_at) < new Date(earliest.remind_at) ? current : earliest
  );
}
