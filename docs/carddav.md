---
title: CardDAV Sync
nav_order: 4
has_children: false
---

# Contact Sync (CardDAV)

Meerkat supports CardDAV in two independent ways:

1. **Built-in CardDAV server** — phones and computers sync their contacts directly against Meerkat (this page, below).
2. **CardDAV client sync** — Meerkat syncs its contacts with an *existing* CardDAV server such as Nextcloud, Radicale, or sabre/dav (see [Syncing with an external CardDAV server](#syncing-with-an-external-carddav-server)).

## Built-in CardDAV server

The built-in CardDAV server allows you to synchronize your contacts with your mobile device or computer (e.g. Apple Contacts in iPhone macOS Contacts or on Android with a third-party CardDAV client like DAVx⁵).

Enable CardDAV by setting the `CARDDAV_ENABLED` environment variable to `true`.  Once enabled, the CardDAV server runs alongside the web interface. A standard discovery endpoint is available at `/.well-known/carddav` for automatic configuration.

## Connecting Your Phone

### iOS

1. Open **Settings** > **Contacts** > **Accounts** > **Add Account** > **Other**.
2. Select **Add CardDAV Account**.
3. Enter the following:
   - **Server**: Your Meerkat CRM URL (e.g., `meerkat.example.com`)
   - **User Name**: Your Meerkat CRM username or email
   - **Password**: Your Meerkat CRM password
4. Tap **Next**. iOS will automatically discover the CardDAV endpoint.
5. Your contacts will begin syncing.

### Android

Android does not include a native CardDAV client. You will need a third-party app such as [DAVx5](https://www.davx5.com/) (open source):

1. Install **DAVx5** from F-Droid or Google Play.
2. Open DAVx5 and add a new account.
3. Select **Login with URL and user name**.
4. Enter:
   - **Base URL**: Your Meerkat CRM URL followed by `/carddav/` (e.g., `https://meerkat.example.com/carddav/`)
   - **User name**: Your Meerkat CRM username or email
   - **Password**: Your Meerkat CRM password
5. DAVx5 will detect the address book. Select it and sync.
6. Your Meerkat CRM contacts will appear in your phone's Contacts app.

## Sync Behavior

- **Two-way sync**: Changes made in Meerkat CRM appear on your phone, and changes made on your phone are synced back to Meerkat CRM. This also applies to profile pictures.
- **Conflict detection**: Meerkat CRM uses ETags to detect conflicts. If a contact has been modified on both the server and the client since the last sync, the client will be notified and can resolve the conflict.
- **Supported fields**: Meerkat CRM syncs all fields though now all fields might be visible in your client. In case you add additional fields on your client (like a secondary address) the fields will be preserved in the Meerkat database but will not show in the Meerkat CRM frontend.

## Troubleshooting

- **Contacts not syncing**: Verify that `CARDDAV_ENABLED=true` is set in your server environment and restart the application.
- **Discovery not working**: Some clients require the full CardDAV URL instead of relying on auto-discovery. Try entering `https://your-server.com/carddav/` directly as the server URL.
- **Locked out**: After multiple failed login attempts, your account may be temporarily locked. Wait a few minutes and try again, or reset your password via the web interface.

# Limitations

- **Sync-token** is not implemented since not yet supported by go-webdav. Clients therefore have to fall back to propfind with depth 1 and compare locally.

## Syncing with an external CardDAV server

Instead of (or in addition to) running its own CardDAV server, Meerkat can connect to an existing CardDAV server — for example Nextcloud, Radicale, Baïkal, or any sabre/dav based server — and keep your contacts in sync with one of its address books.

### Setup

1. Open **Settings → Data → Contact Sync (CardDAV)** and click **Connect server**.
2. Enter the server's base URL and your credentials. If your server supports app passwords (Nextcloud does), use one instead of your account password. Examples:
   - Nextcloud: `https://cloud.example.com/remote.php/dav` (the bare hostname also works via `/.well-known/carddav` discovery)
   - Radicale: `https://radicale.example.com`
   - You can also paste the full URL of a specific address book.
3. Pick the address book to sync with and choose a sync direction. Address books the server manages itself are hidden from the list.
   - **Two-way sync** (default): changes flow in both directions — edits, new contacts, and deletions.
   - **Import only**: Meerkat pulls changes from the server but never writes to it. Contacts you delete in Meerkat stay deleted (they are not re-imported).
   - **Export only**: Meerkat pushes its contacts to the server and treats itself as the source of truth; remote edits and deletions are overwritten on the next sync.

Contacts sync automatically (default every 6 hours) and can be synced manually from the settings page. A sync runs in the background — the settings page shows *Syncing…* while it works and reports the result when it finishes, so you can navigate away and come back. Credentials are stored encrypted (AES-256-GCM).

### Conflict resolution

If the same contact was edited on both sides between syncs, the newer edit wins (compared via the vCard `REV` timestamp against Meerkat's modification time). If the server doesn't provide a `REV` timestamp, the server's version wins. Unknown vCard fields are always preserved in both directions, so no data is destroyed either way. If a contact is edited on one side and deleted on the other, the edit wins and the contact is restored.

Meerkat stamps every contact it uploads with a `REV` reflecting its own last modification (written as `2026-07-22T10:30:00Z`, the encoding RFC 2426 specifies for vCard 3.0), so other clients can apply the same rule to Meerkat's edits. A `REV` that arrives from another client is deliberately not carried forward on later uploads — it describes that client's edit, and re-sending it would understate any changes made in Meerkat since.

Incoming `REV` timestamps are accepted in any of the encodings clients use in practice: the extended form Apple writes on iOS and macOS (`2023-10-04T12:07:13Z`), the same with fractional seconds as Nextcloud's web interface emits, the compact vCard 4.0 form (`20231004T120713Z`), either with or without a UTC offset, and the date-only forms permitted by RFC 2426.

### Environment variables

- `CARDDAV_SYNC_INTERVAL_HOURS` — scheduled sync interval (default `6`, minimum `1`).
- `CARDDAV_BLOCK_PRIVATE_URLS` — refuse connections to private/loopback addresses (SSRF protection for cloud deployments). Defaults to the value of `CALDAV_BLOCK_PRIVATE_URLS`.

### Notes and caveats

- **One connection per user.** All of your contacts sync with the selected address book.
- **Disconnecting keeps your contacts.** Deleting the connection only removes the sync state. If you reconnect to the same address book later, contacts are re-paired by their vCard UID without creating duplicates.
- **Avoid sync loops.** Do not point a phone at both Meerkat's built-in CardDAV server *and* the same external address book that Meerkat syncs with — every change would travel in a circle and may show up with a delay or, in unlucky timing, as duplicates.
- **First sync merging.** Existing contacts are matched by UID first, then by name + primary email. Ambiguous matches are not merged automatically; a duplicate is created instead (safer than merging the wrong people).
- **Limits.** Address books with more than 5000 contacts are not synced.
- **Partial syncs retry themselves.** If individual contacts fail (for example the server rejects a write), the connection is marked *Partially synced* and the failed items are retried on the next run — Meerkat deliberately does not record progress past them.
- **One sync at a time.** A manual sync and the scheduled one never run concurrently for the same connection; the second waits for the first, and pressing *Sync now* while a run is in flight is refused rather than queued.
- **A restart cancels an in-flight sync.** Work already committed is kept and the next run picks up where it left off, since progress is only recorded for items that succeeded.
