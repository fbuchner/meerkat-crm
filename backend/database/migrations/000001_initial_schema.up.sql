-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    username TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Create contacts table
CREATE TABLE IF NOT EXISTS contacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    firstname TEXT NOT NULL COLLATE NOCASE,
    lastname TEXT COLLATE NOCASE,
    nickname TEXT COLLATE NOCASE,
    gender TEXT,
    email TEXT COLLATE NOCASE,
    phone TEXT,
    birthday TEXT,
    photo TEXT,
    photo_thumbnail TEXT,
    address TEXT,
    how_we_met TEXT,
    food_preference TEXT,
    work_information TEXT,
    contact_information TEXT,
    circles TEXT
);

CREATE INDEX IF NOT EXISTS idx_contacts_deleted_at ON contacts(deleted_at);
CREATE INDEX IF NOT EXISTS idx_contacts_firstname ON contacts(firstname COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_contacts_lastname ON contacts(lastname COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_contacts_email ON contacts(email COLLATE NOCASE);

-- Create activities table
CREATE TABLE IF NOT EXISTS activities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    title TEXT NOT NULL,
    description TEXT,
    location TEXT,
    date DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_activities_deleted_at ON activities(deleted_at);
CREATE INDEX IF NOT EXISTS idx_activities_date ON activities(date);

-- Create activity_contacts join table
CREATE TABLE IF NOT EXISTS activity_contacts (
    activity_id INTEGER NOT NULL,
    contact_id INTEGER NOT NULL,
    PRIMARY KEY (activity_id, contact_id),
    FOREIGN KEY (activity_id) REFERENCES activities(id) ON DELETE CASCADE,
    FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_activity_contacts_activity_id ON activity_contacts(activity_id);
CREATE INDEX IF NOT EXISTS idx_activity_contacts_contact_id ON activity_contacts(contact_id);

-- Create notes table
CREATE TABLE IF NOT EXISTS notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    content TEXT NOT NULL,
    date DATETIME NOT NULL,
    contact_id INTEGER,
    FOREIGN KEY (contact_id) REFERENCES contacts(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_notes_deleted_at ON notes(deleted_at);
CREATE INDEX IF NOT EXISTS idx_notes_contact_id ON notes(contact_id);
CREATE INDEX IF NOT EXISTS idx_notes_date ON notes(date);

-- Create relationships table
CREATE TABLE IF NOT EXISTS relationships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    gender TEXT,
    birthday TEXT,
    contact_id INTEGER NOT NULL,
    related_contact_id INTEGER,
    FOREIGN KEY (contact_id) REFERENCES contacts(id),
    FOREIGN KEY (related_contact_id) REFERENCES contacts(id)
);

CREATE INDEX IF NOT EXISTS idx_relationships_deleted_at ON relationships(deleted_at);
CREATE INDEX IF NOT EXISTS idx_relationships_contact_id ON relationships(contact_id);
CREATE INDEX IF NOT EXISTS idx_relationships_related_contact_id ON relationships(related_contact_id);

-- Create reminders table
CREATE TABLE IF NOT EXISTS reminders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    message TEXT NOT NULL,
    by_mail INTEGER DEFAULT 0,
    remind_at DATETIME NOT NULL,
    recurrence TEXT NOT NULL,
    reoccur_from_completion INTEGER DEFAULT 1,
    last_sent DATETIME,
    contact_id INTEGER NOT NULL,
    FOREIGN KEY (contact_id) REFERENCES contacts(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_reminders_deleted_at ON reminders(deleted_at);
CREATE INDEX IF NOT EXISTS idx_reminders_contact_id ON reminders(contact_id);
CREATE INDEX IF NOT EXISTS idx_reminders_remind_at ON reminders(remind_at);
