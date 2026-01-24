-- Add custom field names to users (JSON array of field names in display order)
ALTER TABLE users ADD COLUMN custom_field_names TEXT DEFAULT '[]';

-- Add custom fields to contacts (JSON object mapping field name to value)
ALTER TABLE contacts ADD COLUMN custom_fields TEXT DEFAULT '{}';
