---
title: Data Model
nav_order: 3
has_children: false
---

# Data Model

## Contacts

Contacts are the core of Meerkat CRM and represent people from your network. Use **circles** to group your contacts (e.g. Business, Friends, Bowling Group). 

When creating a contact you can optionally create a birthday reminder. This simply adds a reminder for this contact's birthday. Indepenently of that you will receive emails (if set up) for all birthdays anyway. 

You can add **custom fields** through the settings menu. These will appear on every contact and can hold any text value. When deleting a custom field the information will still be stored in the database but not be visible. It will reappear when you add the same custom field again.

**Profile pictures** are stored downsized to 400x400px as files as well as as 48x48px thumbnails in the database (base64). You upload pictures directly or enter an image URL and have the backend pull it in from the web. Currently JPEG, PNG and HEIC up to 10 MB are supported file formats.

You can **archive** contacts to hide them from default search results and the dashboard. Archiving permanently deletes all active reminders for that contact. Notes, activities and relationships are preserved. Archived contacts are still synced via CardDAV (as often you might have phone contacts that you want to keep but do not want them to show up in Meerkat CRM).


## Activities

Use Activities to document events or interactions with one or more contacts (eg.dinner, phone call or meeting). Each activity has a title, date, and can optionally include a description, location, and linked contacts. If an activity has at least two linked contacts it appears in the network graph.

Activities appear on each linked contact's detail page in the **Timeline** tab, most recent activities show first.

The global **Activities** page lists every activity across all contacts and offers searching and a date range filter.


## Notes

Notes let you capture free-text information.
All notes attached to a specific contact appear in that contact's timeline. Use them to record things you've learned about a person, conversation highlights or anything you want to remember.
Unassigned notes are not attached to any contact. They are accessible from the Notes page in the main navigation and can be used like a personal journal.


## Reminders

Reminders help you stay on top of important dates and follow-ups. 
Reminders can be one-time or recur on a schedule. Reminders are tied to a specific contact and appear on both the contact's detail page and the dashboard. You can decide wether the reminder should reschule from the completion date (e.g. for a catch-up) or from the original date (e.g. for an anniversary).
The **Stay in Touch** button on a contact's detail page opens a prefilled reminder creation dialog for a quarterly catch-up.

If you enable **Send email notification** on a reminder, you will receive an email when the reminder is due. This requires a valid email address on your account and a configured email service (Resend API key) on the server.

When a reminder is due, you can complete or skip it. The difference is that selecting **Complete** creates a completion entry on the related contact's timeline while the **Skip** option directly schedules the next reminder occurence (if there is one) without creating a timeline entry. Overdue reminders remain visible until they are completed or skipped though they will not show up in the reminder emails  again.


## Relationships

Relationships let you map connections between contacts. You can select a suggested relationship or enter your own title. 
Relationships can either link to an existing contact or you can enter details manually without creating a dedicated contact. Relationships are shown both on the outgoing contact ("Sister: Eva") as well as on the incoming contact detail page ("Sister of Jon").
