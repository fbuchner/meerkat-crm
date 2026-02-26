---
title: CardDAV Sync
nav_order: 4
has_children: false
---

# Contact Sync (CardDAV)

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
