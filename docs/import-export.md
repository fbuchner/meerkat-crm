---
title: Import & Export
parent: User Guide
nav_order: 7
---

# Import & Export

## Importing Contacts

Meerkat CRM supports importing contacts from CSV and VCF (vCard) files. The import process includes a preview step with duplicate detection so you can review changes before they are applied.

To start an import, go to the **Contacts** page and click **Import**.

### CSV Import

The CSV import follows a four-step process:

1. **Upload**: Select a CSV file from your device (max 5 MB, up to 1,000 rows). The file should have column headers in the first row. You can also drag and drop a file into the upload area.

2. **Map Columns**: Match your CSV columns to Meerkat CRM contact fields. The importer shows each column header along with sample data from your file and suggests mappings based on the column names. Set columns you don't want to import to "Ignore". At least one column must be mapped.

   Supported fields for mapping: First Name, Last Name, Nickname, Gender, Email, Phone, Birthday, Address, How We Met, Food Preferences, Work Information, Additional Contact Information, and Circles.

3. **Review**: Preview the contacts that will be imported. Each row shows a status:
   - **Valid**: The contact will be created as a new entry.
   - **Duplicate**: A matching contact was found (by name or email). You can choose to **Add as New**, **Update Existing**, or **Skip** for each duplicate.
   - **Error**: The row has validation issues and will be skipped.

   A summary shows how many contacts will be created, updated, and skipped.

4. **Done**: The import results are displayed, showing how many contacts were created, updated, and skipped. If any existing contacts were updated, a merge note is automatically added documenting what changed.

### VCF Import

VCF (vCard) import works similarly but skips the column mapping step, since vCard files have a standardized format:

1. **Upload**: Select a VCF file (max 10 MB, up to 1,000 contacts).
2. **Review**: Preview the contacts with duplicate detection, just like CSV import.
3. **Done**: Results including any photos that were processed.

VCF import also handles embedded photos and photo URLs. Photos from the vCard are automatically saved and thumbnails are generated.

### Import Preview and Confirmation

During the review step, the importer detects potential duplicates by matching on:

- **Email address**: If an imported contact has the same email as an existing contact.
- **Name**: If an imported contact has the same first and last name as an existing contact.

For each duplicate, you can choose:
- **Skip**: Don't import this contact.
- **Add as New**: Import as a separate contact, even though a similar one exists.
- **Update Existing**: Merge the imported data into the existing contact. Non-empty imported fields will overwrite existing values.

## Exporting Data

### Full Data Export

The full data export downloads all your data as a single CSV file. This includes:

- **Contacts**: All contact fields plus custom field values.
- **Relationships**: All relationship definitions.
- **Activities**: All activities with participant names.
- **Notes**: All notes (both contact-specific and unassigned).
- **Reminders**: All reminders with recurrence settings.

Profile photos are **not** included in the CSV export.

To export, go to **Settings** and click **Download CSV** under the Export Data section.

### VCF Export

The VCF export downloads all your contacts as a vCard file. This format is compatible with most contact management apps (Apple Contacts, Google Contacts, Outlook, etc.) and **includes profile photos**.

To export, go to **Settings** and click **Download VCF** under the Export Contacts as VCF section.
