-- Add language preference column to users table
ALTER TABLE users ADD COLUMN language TEXT DEFAULT 'en';
