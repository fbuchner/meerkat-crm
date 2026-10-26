import { test, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, within, cleanup } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import i18n from '../i18n/config';
import ContactInformation from './ContactInformation';
import { resolveEnabledFields } from '../contactFields';
import { DateFormatProvider } from '../DateFormatProvider';

const baseProps = {
  editingField: null,
  validationError: '',
  editValue: '',
  onEditStart: vi.fn(),
  onEditCancel: vi.fn(),
  onEditSave: vi.fn(),
  onEditValueChange: vi.fn(),
  enabledFields: resolveEnabledFields(['is_deceased', 'birthday']),
};

afterEach(cleanup);

function renderWithI18n(ui: React.ReactElement) {
  return render(
    <I18nextProvider i18n={i18n}>
      <DateFormatProvider>{ui}</DateFormatProvider>
    </I18nextProvider>
  );
}

test('shows Deceased label and date when contact is deceased', () => {
  renderWithI18n(
    <ContactInformation
      {...baseProps}
      contact={{ firstname: 'Jane', birthday: '1990-05-01', is_deceased: true, deceased_date: '2020-01-15' }}
      onUpdateContact={vi.fn()}
    />
  );
  expect(screen.getByText(/15\.01\.2020/)).toBeInTheDocument();
  // Age-at-death (29) is shown on both the Deceased row and the Birthday row
  // (which shows age-at-death, not age-as-of-today, for deceased contacts).
  expect(screen.getAllByText(/\(29 years old\)/)).toHaveLength(2);
});

test('birthday row shows age-at-death (not age-as-of-today) for a deceased contact', () => {
  renderWithI18n(
    <ContactInformation
      {...baseProps}
      contact={{ firstname: 'Jane', birthday: '1990-05-01', is_deceased: true, deceased_date: '2020-01-15' }}
      onUpdateContact={vi.fn()}
    />
  );
  // Should not show today's age (would be much higher than 29).
  expect(screen.queryByText(/\(36 years old\)/)).not.toBeInTheDocument();
});

test('checking the Deceased checkbox and saving calls onUpdateContact with is_deceased true', async () => {
  const onUpdateContact = vi.fn().mockResolvedValue(undefined);
  renderWithI18n(
    <ContactInformation
      {...baseProps}
      contact={{ firstname: 'Jane', is_deceased: false, deceased_date: '' }}
      onUpdateContact={onUpdateContact}
    />
  );

  const heartIcon = screen.getByTestId('HeartBrokenIcon');
  const deceasedField = heartIcon.closest('.MuiBox-root')?.parentElement as HTMLElement;
  fireEvent.click(within(deceasedField as HTMLElement).getByLabelText('Edit'));
  fireEvent.click(screen.getByRole('checkbox', { name: /deceased/i }));
  fireEvent.click(screen.getByText('Save'));

  expect(onUpdateContact).toHaveBeenCalled();
  const [[partial]] = onUpdateContact.mock.calls;
  expect(partial.is_deceased).toBe(true);
});

test('hideEmptyFields hides empty fields until "Show all fields" is clicked', () => {
  renderWithI18n(
    <ContactInformation
      {...baseProps}
      enabledFields={resolveEnabledFields(['birthday', 'organization', 'job_title'])}
      hideEmptyFields
      contact={{ ID: 1, firstname: 'Jane', organization: 'Acme' }}
      onUpdateContact={vi.fn()}
    />
  );
  expect(screen.getByText('Acme')).toBeInTheDocument();
  expect(screen.queryByTestId('CakeIcon')).not.toBeInTheDocument();
  expect(screen.queryByTestId('BadgeIcon')).not.toBeInTheDocument();

  fireEvent.click(screen.getByRole('button', { name: /show all fields/i }));
  expect(screen.getByTestId('CakeIcon')).toBeInTheDocument();
  expect(screen.getByTestId('BadgeIcon')).toBeInTheDocument();

  fireEvent.click(screen.getByRole('button', { name: /hide empty fields/i }));
  expect(screen.queryByTestId('CakeIcon')).not.toBeInTheDocument();
});

test('without hideEmptyFields every enabled field shows and there is no toggle', () => {
  renderWithI18n(
    <ContactInformation
      {...baseProps}
      enabledFields={resolveEnabledFields(['birthday', 'organization'])}
      contact={{ ID: 1, firstname: 'Jane' }}
      onUpdateContact={vi.fn()}
    />
  );
  expect(screen.getByTestId('CakeIcon')).toBeInTheDocument();
  expect(screen.queryByRole('button', { name: /show all fields/i })).not.toBeInTheDocument();
});

test('a field being edited stays visible while collapsed', () => {
  renderWithI18n(
    <ContactInformation
      {...baseProps}
      enabledFields={resolveEnabledFields(['birthday'])}
      hideEmptyFields
      editingField="birthday"
      contact={{ ID: 1, firstname: 'Jane' }}
      onUpdateContact={vi.fn()}
    />
  );
  expect(screen.getByTestId('CakeIcon')).toBeInTheDocument();
});
