package services

import (
	"context"
	"encoding/json"
	mcarddav "meerkat/carddav"
	"meerkat/config"
	"meerkat/models"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Each scenario runs against every backend in contactSyncBackends() — see
// contact_sync_remote_test.go for the harness and how to enable Radicale.

func newContactSyncDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(
		&models.User{}, &models.Contact{}, &models.JobExecution{},
		&models.CardDAVConnection{}, &models.CardDAVContactLink{}, &models.Relationship{},
	))
	return db
}

type contactSyncEnv struct {
	db      *gorm.DB
	cfg     config.Config
	user    models.User
	conn    *models.CardDAVConnection
	remote  *davRemote
	service *ContactSyncService
}

func setupContactSyncEnv(t *testing.T, direction string, remote *davRemote) *contactSyncEnv {
	t.Helper()
	env := &contactSyncEnv{
		db:      newContactSyncDB(t),
		cfg:     config.Config{JWTSecretKey: "test-jwt-secret-key-with-32-chars!!", ProfilePhotoDir: t.TempDir()},
		remote:  remote,
		service: NewContactSyncService(false),
	}
	env.user = models.User{Username: "local", Password: "password123!A", Email: "local@example.com"}
	require.NoError(t, env.db.Create(&env.user).Error)

	env.conn = &models.CardDAVConnection{
		UserID:          env.user.ID,
		BaseURL:         remote.proxyURL,
		AddressBookPath: remote.addressBookPath(),
		Direction:       direction,
		SyncEnabled:     true,
		Username:        remote.user,
	}
	if remote.pass != "" {
		encrypted, err := EncryptCredential(env.cfg.JWTSecretKey, remote.pass)
		require.NoError(t, err)
		env.conn.PasswordEncrypted = encrypted
	}
	require.NoError(t, env.db.Create(env.conn).Error)
	return env
}

func (env *contactSyncEnv) sync(t *testing.T) ContactSyncStats {
	t.Helper()
	stats, err := env.service.SyncConnection(context.Background(), env.db, env.cfg, env.conn)
	require.NoError(t, err)
	return stats
}

func (env *contactSyncEnv) createContact(t *testing.T, firstname, lastname, email string) models.Contact {
	t.Helper()
	contact := models.Contact{UserID: env.user.ID, Firstname: firstname, Lastname: lastname}
	if email != "" {
		contact.Emails = []models.ContactEmail{{Type: "home", Value: email}}
	}
	require.NoError(t, env.db.Create(&contact).Error)
	return contact
}

func (env *contactSyncEnv) localContacts(t *testing.T) []models.Contact {
	t.Helper()
	var contacts []models.Contact
	require.NoError(t, env.db.Where("user_id = ?", env.user.ID).Find(&contacts).Error)
	return contacts
}

func (env *contactSyncEnv) links(t *testing.T) []models.CardDAVContactLink {
	t.Helper()
	var links []models.CardDAVContactLink
	require.NoError(t, env.db.Where("connection_id = ?", env.conn.ID).Find(&links).Error)
	return links
}

func TestContactSyncInitialPullCreatesContacts(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionTwoWay, func(t *testing.T, env *contactSyncEnv) {
		env.remote.putContact(t, "uid-alice", "Alice", "Remote", "alice@example.com")
		env.remote.putContact(t, "uid-bob", "Bob", "Remote", "bob@example.com")

		stats := env.sync(t)

		assert.Equal(t, 2, stats.PulledCreated)
		contacts := env.localContacts(t)
		require.Len(t, contacts, 2)
		uids := map[string]bool{}
		for _, c := range contacts {
			uids[c.VCardUID] = true
		}
		assert.True(t, uids["uid-alice"] && uids["uid-bob"])
		assert.Len(t, env.links(t), 2)
	})
}

func TestContactSyncInitialPushCreatesRemote(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionTwoWay, func(t *testing.T, env *contactSyncEnv) {
		env.createContact(t, "Carol", "Local", "carol@example.com")
		env.createContact(t, "Dave", "Local", "dave@example.com")

		stats := env.sync(t)

		assert.Equal(t, 2, stats.PushedCreated)
		assert.Equal(t, 2, env.remote.count(t))
		assert.Len(t, env.links(t), 2)
	})
}

func TestContactSyncRelinksByUID(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionTwoWay, func(t *testing.T, env *contactSyncEnv) {
		local := models.Contact{UserID: env.user.ID, Firstname: "Erin", Lastname: "Shared", VCardUID: "uid-shared"}
		require.NoError(t, env.db.Create(&local).Error)
		env.remote.putContact(t, "uid-shared", "Erin", "SharedRemote", "erin@example.com")

		stats := env.sync(t)

		// Same UID on both sides pairs up instead of duplicating; the remote
		// representation wins the initial merge (no REV on the remote card).
		assert.Equal(t, 0, stats.PulledCreated)
		assert.Equal(t, 0, stats.PushedCreated)
		assert.Equal(t, 1, stats.PulledUpdated)
		// A first-time pairing has no merge base to diff against, so it is not
		// a conflict — reporting one would tell the user something went wrong
		// for every contact matched on an initial connect.
		assert.Equal(t, 0, stats.Conflicts)
		contacts := env.localContacts(t)
		require.Len(t, contacts, 1)
		assert.Equal(t, "SharedRemote", contacts[0].Lastname)
		assert.Len(t, env.links(t), 1)
	})
}

func TestContactSyncMatchByNameAndEmail(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionTwoWay, func(t *testing.T, env *contactSyncEnv) {
		env.createContact(t, "Frank", "Match", "frank@example.com")
		env.remote.putContact(t, "uid-remote-frank", "Frank", "Match", "frank@example.com")

		env.sync(t)

		contacts := env.localContacts(t)
		require.Len(t, contacts, 1)
		// The merged contact adopts the remote UID so both sides share one identity.
		assert.Equal(t, "uid-remote-frank", contacts[0].VCardUID)
		links := env.links(t)
		require.Len(t, links, 1)
		assert.Equal(t, "uid-remote-frank", links[0].RemoteUID)
	})
}

// TestContactSyncMatchByIdentityLocalWinsPersistsUID covers a merge-by-identity
// whose conflict resolves in favor of the local edit (push). mergeRemoteCreation
// adopts the remote UID onto the local contact so the link, future pushes, and
// the built-in CardDAV server agree on one identity — but resolveConflict's
// push branch only saves the link row, never the contact row. Without persisting
// the UID at the point it's assigned, the contact row keeps its original,
// locally-generated UID while the link's RemoteUID (and remote path) point at
// the adopted one; the next push then embeds the stale UID in a vCard PUT to a
// path keyed by the correct one, which conforming CardDAV servers (Radicale
// included) reject as a UID/path mismatch.
func TestContactSyncMatchByIdentityLocalWinsPersistsUID(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionTwoWay, func(t *testing.T, env *contactSyncEnv) {
		env.createContact(t, "Frank", "Match", "frank@example.com")
		env.remote.putContact(t, "uid-remote-frank", "Frank", "Match", "frank@example.com")
		env.remote.setRemoteRevision(t, "uid-remote-frank", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))

		stats := env.sync(t)

		// Merged on first contact, so not counted as a conflict; the local edit
		// still wins on REV and is pushed.
		assert.Equal(t, 0, stats.Conflicts)
		assert.Equal(t, 1, stats.PushedUpdated)

		var local models.Contact
		require.NoError(t, env.db.Where("firstname = ?", "Frank").First(&local).Error)
		assert.Equal(t, "uid-remote-frank", local.VCardUID)

		links := env.links(t)
		require.Len(t, links, 1)
		assert.Equal(t, local.VCardUID, links[0].RemoteUID)

		// A no-op second sync must not error or re-conflict: if the UID had
		// been left stale, this push would fail (or, against the in-memory
		// fake, at least drift from the link's remote_uid again).
		stats = env.sync(t)
		assert.Equal(t, ContactSyncStats{}, stats)
	})
}

func TestContactSyncRemoteEditPulled(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionTwoWay, func(t *testing.T, env *contactSyncEnv) {
		env.remote.putContact(t, "uid-grace", "Grace", "Old", "grace@example.com")
		env.sync(t)

		env.remote.putContact(t, "uid-grace", "Grace", "New", "grace@example.com")
		stats := env.sync(t)

		assert.Equal(t, 1, stats.PulledUpdated)
		contacts := env.localContacts(t)
		require.Len(t, contacts, 1)
		assert.Equal(t, "New", contacts[0].Lastname)
	})
}

func TestContactSyncLocalEditPushed(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionTwoWay, func(t *testing.T, env *contactSyncEnv) {
		local := env.createContact(t, "Henry", "Old", "henry@example.com")
		env.sync(t)

		require.NoError(t, env.db.First(&local, local.ID).Error)
		local.Lastname = "New"
		require.NoError(t, env.db.Save(&local).Error)
		stats := env.sync(t)

		assert.Equal(t, 1, stats.PushedUpdated)
		assert.Equal(t, "New", env.remote.lastname(t, local.VCardUID))
	})
}

func TestContactSyncRemoteDeletePropagates(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionTwoWay, func(t *testing.T, env *contactSyncEnv) {
		env.remote.putContact(t, "uid-iris", "Iris", "Gone", "iris@example.com")
		env.sync(t)

		env.remote.deleteContact(t, "uid-iris")
		stats := env.sync(t)

		assert.Equal(t, 1, stats.PulledDeleted)
		assert.Empty(t, env.localContacts(t))
		assert.Empty(t, env.links(t))
	})
}

func TestContactSyncLocalDeletePropagates(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionTwoWay, func(t *testing.T, env *contactSyncEnv) {
		local := env.createContact(t, "Jack", "Gone", "jack@example.com")
		env.sync(t)

		require.NoError(t, env.db.Delete(&models.Contact{}, local.ID).Error)
		stats := env.sync(t)

		assert.Equal(t, 1, stats.PushedDeleted)
		assert.Zero(t, env.remote.count(t))
		assert.Empty(t, env.links(t))
	})
}

func TestContactSyncConflictRemoteWinsWhenNewer(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionTwoWay, func(t *testing.T, env *contactSyncEnv) {
		env.remote.putContact(t, "uid-kim", "Kim", "Base", "kim@example.com")
		env.sync(t)

		var local models.Contact
		require.NoError(t, env.db.Where("vcard_uid = ?", "uid-kim").First(&local).Error)
		local.Lastname = "LocalEdit"
		require.NoError(t, env.db.Save(&local).Error)

		// The remote edit lands after the local one, so it wins on revision.
		env.remote.putContact(t, "uid-kim", "Kim", "RemoteEdit", "kim@example.com")
		env.remote.setRemoteRevision(t, "uid-kim", time.Now().Add(time.Hour))

		stats := env.sync(t)

		assert.Equal(t, 1, stats.Conflicts)
		require.NoError(t, env.db.First(&local, local.ID).Error)
		assert.Equal(t, "RemoteEdit", local.Lastname)
	})
}

// TestContactSyncConflictRemoteWinsWithoutREV covers the fallback for servers
// that report no revision at all: without a basis for comparison the remote
// copy wins. Only servers that store cards verbatim can present this case.
func TestContactSyncConflictRemoteWinsWithoutREV(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionTwoWay, func(t *testing.T, env *contactSyncEnv) {
		if !env.remote.preservesRawCards {
			t.Skipf("%s always derives a REV, so a card without one cannot be served", env.remote.name)
		}
		env.remote.putContact(t, "uid-kim2", "Kim", "Base", "kim2@example.com")
		env.sync(t)

		env.remote.putRaw(t, "uid-kim2", strings.Join([]string{
			"BEGIN:VCARD", "VERSION:3.0", "UID:uid-kim2",
			"FN:Kim RemoteEdit", "N:RemoteEdit;Kim;;;",
			"EMAIL;TYPE=HOME:kim2@example.com",
			"END:VCARD", "",
		}, "\r\n"))
		require.True(t, env.remote.revision(t, "uid-kim2").IsZero(),
			"this scenario requires a card the server reports no revision for")

		var local models.Contact
		require.NoError(t, env.db.Where("vcard_uid = ?", "uid-kim2").First(&local).Error)
		local.Lastname = "LocalEdit"
		require.NoError(t, env.db.Save(&local).Error)

		stats := env.sync(t)

		assert.Equal(t, 1, stats.Conflicts)
		require.NoError(t, env.db.First(&local, local.ID).Error)
		assert.Equal(t, "RemoteEdit", local.Lastname)
	})
}

func TestContactSyncConflictLocalWinsWithNewerEdit(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionTwoWay, func(t *testing.T, env *contactSyncEnv) {
		env.remote.putContact(t, "uid-lena", "Lena", "Base", "lena@example.com")
		env.sync(t)

		// A remote edit that predates the local one below must lose.
		env.remote.putContact(t, "uid-lena", "Lena", "RemoteEdit", "lena@example.com")
		env.remote.setRemoteRevision(t, "uid-lena", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))

		var local models.Contact
		require.NoError(t, env.db.Where("vcard_uid = ?", "uid-lena").First(&local).Error)
		local.Lastname = "LocalEdit"
		require.NoError(t, env.db.Save(&local).Error)

		stats := env.sync(t)

		assert.Equal(t, 1, stats.Conflicts)
		assert.Equal(t, 1, stats.PushedUpdated)
		assert.Equal(t, "LocalEdit", env.remote.lastname(t, "uid-lena"))
	})
}

// TestContactSyncConflictHonorsExtendedFormatREV covers the encoding Apple
// writes on every platform, and that Nextcloud's web UI writes with fractional
// seconds. Reading REV with go-vcard's strict basic-format parser rejected
// these outright, which silently collapsed conflict resolution to "remote
// always wins" against the most common servers Meerkat is pointed at.
func TestContactSyncConflictHonorsExtendedFormatREV(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionTwoWay, func(t *testing.T, env *contactSyncEnv) {
		if !env.remote.preservesRawCards {
			t.Skipf("%s rewrites stored cards, so a specific REV encoding cannot be staged", env.remote.name)
		}
		env.remote.putContact(t, "uid-apple", "Ada", "Base", "ada@example.com")
		env.sync(t)

		// An Apple-style remote edit that predates the local one below.
		env.remote.putRaw(t, "uid-apple", strings.Join([]string{
			"BEGIN:VCARD", "VERSION:3.0",
			"PRODID:-//Apple Inc.//iOS 17.0.2//EN",
			"UID:uid-apple",
			"FN:Ada RemoteEdit", "N:RemoteEdit;Ada;;;",
			"EMAIL;TYPE=HOME:ada@example.com",
			"REV:2020-01-01T00:00:00Z",
			"END:VCARD", "",
		}, "\r\n"))
		require.False(t, env.remote.revision(t, "uid-apple").IsZero(),
			"an Apple-encoded REV must be readable at all")

		var local models.Contact
		require.NoError(t, env.db.Where("vcard_uid = ?", "uid-apple").First(&local).Error)
		local.Lastname = "LocalEdit"
		require.NoError(t, env.db.Save(&local).Error)

		stats := env.sync(t)

		assert.Equal(t, 1, stats.Conflicts)
		assert.Equal(t, 1, stats.PushedUpdated, "the newer local edit must win over the 2020 remote revision")
		assert.Equal(t, "LocalEdit", env.remote.lastname(t, "uid-apple"))
	})
}

// TestContactSyncPushedCardCarriesFreshRevision is the end-to-end guard for the
// contract the conflict rules rest on: whatever Meerkat uploads must advertise
// Meerkat's own modification time. Re-emitting a REV that arrived with an
// import would tell peers the contact is older than it is, inviting them to
// discard local edits.
func TestContactSyncPushedCardCarriesFreshRevision(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionTwoWay, func(t *testing.T, env *contactSyncEnv) {
		if !env.remote.preservesRawCards {
			t.Skipf("%s rewrites stored cards, so the pushed REV cannot be observed", env.remote.name)
		}

		// A contact that arrives carrying somebody else's stale revision.
		env.remote.putRaw(t, "uid-rev", strings.Join([]string{
			"BEGIN:VCARD", "VERSION:3.0", "UID:uid-rev",
			"FN:Rev Person", "N:Person;Rev;;;",
			"EMAIL;TYPE=HOME:rev@example.com",
			"REV:20200101T000000Z",
			"END:VCARD", "",
		}, "\r\n"))
		env.sync(t)

		var local models.Contact
		require.NoError(t, env.db.Where("vcard_uid = ?", "uid-rev").First(&local).Error)
		local.Lastname = "EditedLocally"
		require.NoError(t, env.db.Save(&local).Error)
		env.sync(t)

		rev := env.remote.revision(t, "uid-rev")
		require.False(t, rev.IsZero(), "pushed card must carry a REV")
		require.NoError(t, env.db.First(&local, local.ID).Error)
		assert.WithinDuration(t, local.UpdatedAt.UTC(), rev, time.Second,
			"pushed REV must track the local modification time, not the imported one")
	})
}

func TestContactSyncEditBeatsDeleteRestoresLocal(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionTwoWay, func(t *testing.T, env *contactSyncEnv) {
		env.remote.putContact(t, "uid-mia", "Mia", "Base", "mia@example.com")
		env.sync(t)

		var local models.Contact
		require.NoError(t, env.db.Where("vcard_uid = ?", "uid-mia").First(&local).Error)
		require.NoError(t, env.db.Delete(&local).Error)
		env.remote.putContact(t, "uid-mia", "Mia", "Edited", "mia@example.com")

		env.sync(t)

		require.NoError(t, env.db.First(&local, local.ID).Error) // not soft-deleted anymore
		assert.Equal(t, "Edited", local.Lastname)
	})
}

func TestContactSyncPullOnlyNeverWritesToServer(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionPull, func(t *testing.T, env *contactSyncEnv) {
		env.remote.putContact(t, "uid-nora", "Nora", "Remote", "nora@example.com")
		env.createContact(t, "Owen", "LocalOnly", "owen@example.com")

		stats := env.sync(t)

		assert.Equal(t, 1, stats.PulledCreated)
		assert.Zero(t, stats.PushedCreated)
		assert.Len(t, env.localContacts(t), 2)
		assert.Zero(t, env.remote.methodCount(http.MethodPut), "pull-only must not PUT")
		assert.Zero(t, env.remote.methodCount(http.MethodDelete), "pull-only must not DELETE")
	})
}

func TestContactSyncPullOnlyKeepsLocalTombstone(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionPull, func(t *testing.T, env *contactSyncEnv) {
		env.remote.putContact(t, "uid-pia", "Pia", "Remote", "pia@example.com")
		env.sync(t)

		var local models.Contact
		require.NoError(t, env.db.Where("vcard_uid = ?", "uid-pia").First(&local).Error)
		require.NoError(t, env.db.Delete(&local).Error)
		env.remote.putContact(t, "uid-pia", "Pia", "Edited", "pia@example.com")

		env.sync(t)

		// The locally deleted contact must not be resurrected by the remote edit.
		err := env.db.Where("vcard_uid = ?", "uid-pia").First(&local).Error
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Zero(t, env.remote.methodCount(http.MethodDelete))
	})
}

func TestContactSyncPushOnlyIgnoresRemoteAndReassertsDeletes(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionPush, func(t *testing.T, env *contactSyncEnv) {
		env.remote.putContact(t, "uid-quinn", "Quinn", "Foreign", "quinn@example.com")
		local := env.createContact(t, "Rita", "Local", "rita@example.com")

		stats := env.sync(t)
		assert.Equal(t, 1, stats.PushedCreated)
		assert.Zero(t, stats.PulledCreated, "push-only must not import remote contacts")
		assert.Len(t, env.localContacts(t), 1)

		// Deleting the pushed contact on the server: push-only treats Meerkat as
		// authoritative and re-creates it.
		env.remote.deleteContact(t, local.VCardUID)

		env.sync(t)
		card, ok := env.remote.card(t, local.VCardUID)
		require.True(t, ok, "push-only must re-create the deleted remote object")
		require.NotNil(t, card.Name())
		assert.Equal(t, "Rita", card.Name().GivenName)
	})
}

func TestContactSyncSecondRunIsQuiet(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionTwoWay, func(t *testing.T, env *contactSyncEnv) {
		env.remote.putContact(t, "uid-sam", "Sam", "Remote", "sam@example.com")
		env.createContact(t, "Tess", "Local", "tess@example.com")
		env.sync(t)

		// Nothing changed: the second run must not echo anything back.
		stats := env.sync(t)
		assert.Equal(t, ContactSyncStats{}, stats)
	})
}

// TestContactSyncTokenModeIsUsed pins which remote-state strategy each server
// gets. Without it a regression could silently drop every sync-collection
// server onto the full-listing fallback and every other scenario would still
// pass, hiding the loss of incremental sync.
func TestContactSyncTokenModeIsUsed(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionTwoWay, func(t *testing.T, env *contactSyncEnv) {
		env.remote.putContact(t, "uid-token-1", "Tim", "Token", "tim@example.com")
		env.sync(t)

		var stored models.CardDAVConnection
		require.NoError(t, env.db.First(&stored, env.conn.ID).Error)

		if !env.remote.syncCollection {
			assert.Empty(t, stored.SyncToken, "a server without sync-collection must not persist a token")
			return
		}
		require.NotEmpty(t, stored.SyncToken, "sync-collection token must be persisted for incremental runs")
		firstToken := stored.SyncToken

		// Incremental run: only the new object should come back, and the token
		// must advance so the next run starts from here.
		env.remote.putContact(t, "uid-token-2", "Tina", "Token", "tina@example.com")
		stats := env.sync(t)

		assert.Equal(t, 1, stats.PulledCreated)
		require.NoError(t, env.db.First(&stored, env.conn.ID).Error)
		assert.NotEqual(t, firstToken, stored.SyncToken, "sync token must advance after a change")

		// A deletion in token mode arrives as an explicit tombstone rather than
		// as absence from a full listing.
		env.remote.deleteContact(t, "uid-token-2")
		stats = env.sync(t)
		assert.Equal(t, 1, stats.PulledDeleted)
		assert.Len(t, env.localContacts(t), 1)
	})
}

// TestContactSyncItemErrorHoldsSyncToken pins that a run with per-item
// failures reports itself as partial and does not advance the sync token.
// Advancing it would mean the next incremental REPORT never offers the failed
// work again, quietly turning a transient server error into permanent loss.
func TestContactSyncItemErrorHoldsSyncToken(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionTwoWay, func(t *testing.T, env *contactSyncEnv) {
		env.remote.putContact(t, "uid-anchor", "Anna", "Anchor", "anna@example.com")
		env.sync(t)

		var stored models.CardDAVConnection
		require.NoError(t, env.db.First(&stored, env.conn.ID).Error)
		tokenBefore := stored.SyncToken

		// Work on both sides: the remote object imports fine, the local contact
		// needs a PUT — and the server rejects every PUT.
		env.remote.putContact(t, "uid-later", "Lena", "Later", "lena@example.com")
		env.createContact(t, "Peg", "Push", "peg@example.com")
		env.remote.failRequests(func(req *http.Request) bool { return req.Method == http.MethodPut })

		stats := env.sync(t)
		assert.Equal(t, 1, stats.Errors)
		assert.Equal(t, 0, stats.PushedCreated)
		assert.Equal(t, 1, stats.PulledCreated)

		require.NoError(t, env.db.First(&stored, env.conn.ID).Error)
		assert.Equal(t, models.CardDAVSyncStatusPartial, stored.LastSyncStatus)
		assert.Contains(t, stored.LastSyncError, "1 contacts failed to sync")
		assert.Equal(t, tokenBefore, stored.SyncToken,
			"a run with item errors must not advance the sync token past the failed work")

		// With the server healthy again the push lands, without the user having
		// to touch anything.
		env.remote.failRequests(nil)
		stats = env.sync(t)
		assert.Equal(t, 0, stats.Errors)
		assert.Equal(t, 1, stats.PushedCreated)

		require.NoError(t, env.db.First(&stored, env.conn.ID).Error)
		assert.Equal(t, models.CardDAVSyncStatusSuccess, stored.LastSyncStatus)
		assert.Empty(t, stored.LastSyncError)
	})
}

// TestContactSyncPushOnlyDeletesRemoteDespiteRemoteEdit covers a local deletion
// racing a remote edit under push-only sync. Meerkat is authoritative there, so
// the remote change is irrelevant — but the object must still be removed. The
// remote pass sees a changed etag and defers to the local pass, which must not
// then mistake the object for one the remote pass already handled.
func TestContactSyncPushOnlyDeletesRemoteDespiteRemoteEdit(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionPush, func(t *testing.T, env *contactSyncEnv) {
		local := env.createContact(t, "Dana", "Drop", "dana@example.com")
		env.sync(t)
		_, ok := env.remote.card(t, local.VCardUID)
		require.True(t, ok, "the contact should have been pushed on the first run")

		// Edit remotely (which moves its etag) and delete locally in the same
		// window, so both passes have a claim on the object.
		env.remote.putContact(t, local.VCardUID, "Dana", "RemoteEdit", "dana@example.com")
		require.NoError(t, env.db.Delete(&models.Contact{}, local.ID).Error)

		stats := env.sync(t)
		assert.Equal(t, 0, stats.Errors)
		assert.Equal(t, 1, stats.PushedDeleted)

		// count rather than a GET: Meerkat's own CardDAV server answers 500,
		// not 404, for an object that is gone.
		assert.Zero(t, env.remote.count(t),
			"push-only must remove the object even when the remote copy changed")
		assert.Empty(t, env.links(t), "the link must be dropped along with the object")
	})
}

// TestStartContactSyncRunsInBackground covers the entry point the API uses:
// it returns immediately, and the outcome the UI polls for is persisted rather
// than returned, because the run outlives the request that started it.
func TestStartContactSyncRunsInBackground(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionTwoWay, func(t *testing.T, env *contactSyncEnv) {
		env.remote.putContact(t, "uid-async", "Ana", "Sync", "ana@example.com")

		require.True(t, StartContactSync(env.db, env.cfg, *env.conn))
		require.Eventually(t, func() bool { return !IsContactSyncRunning(env.conn.ID) },
			30*time.Second, 10*time.Millisecond, "the background run should finish")

		var stored models.CardDAVConnection
		require.NoError(t, env.db.First(&stored, env.conn.ID).Error)
		assert.Equal(t, models.CardDAVSyncStatusSuccess, stored.LastSyncStatus)

		// The counts have to survive the request, so they live on the row.
		var stats ContactSyncStats
		require.NoError(t, json.Unmarshal([]byte(stored.LastSyncStats), &stats))
		assert.Equal(t, 1, stats.PulledCreated)
		assert.Len(t, env.localContacts(t), 1)
	})
}

// TestStartContactSyncRefusesConcurrentRun pins that pressing "Sync now" twice
// cannot start two runs against the same connection.
func TestStartContactSyncRefusesConcurrentRun(t *testing.T) {
	runContactSync(t, models.CardDAVDirectionTwoWay, func(t *testing.T, env *contactSyncEnv) {
		// Hold the run open on its first request so the second start is
		// guaranteed to land while it is still in flight.
		release := make(chan struct{})
		env.remote.failRequests(func(req *http.Request) bool {
			<-release
			return false
		})

		require.True(t, StartContactSync(env.db, env.cfg, *env.conn))
		assert.True(t, IsContactSyncRunning(env.conn.ID))
		assert.False(t, StartContactSync(env.db, env.cfg, *env.conn),
			"a second run must be refused while one is already in flight")

		close(release)
		require.Eventually(t, func() bool { return !IsContactSyncRunning(env.conn.ID) },
			30*time.Second, 10*time.Millisecond)

		// Once it is done, a fresh run is allowed again.
		require.True(t, StartContactSync(env.db, env.cfg, *env.conn))
		require.Eventually(t, func() bool { return !IsContactSyncRunning(env.conn.ID) },
			30*time.Second, 10*time.Millisecond)
	})
}

func TestCanonicalVCardHashDeterministicAndIgnoresREV(t *testing.T) {
	contact := &models.Contact{
		Firstname: "Uma", Lastname: "Hash", VCardUID: "uid-uma",
		Emails: []models.ContactEmail{{Type: "home", Value: "uma@example.com"}},
		Phones: []models.ContactPhone{{Type: "cell", Value: "+123456789"}},
	}

	first := canonicalVCardHash(mcarddav.ContactToVCard(contact, ""))
	require.NotEmpty(t, first)
	for i := 0; i < 5; i++ {
		assert.Equal(t, first, canonicalVCardHash(mcarddav.ContactToVCard(contact, "")))
	}

	withRev := mcarddav.ContactToVCard(contact, "")
	withRev.Add(vcard.FieldRevision, &vcard.Field{Value: "20240101T000000Z"})
	assert.Equal(t, first, canonicalVCardHash(withRev))
}

func TestIsGeneratedAddressBook(t *testing.T) {
	cases := []struct {
		name string
		path string
		want bool
	}{
		{"nextcloud system book", "/remote.php/dav/addressbooks/users/admin/z-server-generated--system/", true},
		{"nextcloud recently contacted", "/remote.php/dav/addressbooks/users/admin/z-app-generated--contactsinteraction--recent/", true},
		{"any other app book", "/remote.php/dav/addressbooks/users/admin/z-app-generated--something/", true},
		{"uppercase", "/remote.php/dav/addressbooks/users/admin/Z-SERVER-GENERATED--SYSTEM/", true},
		{"no trailing slash", "/remote.php/dav/addressbooks/users/admin/z-app-generated--x", true},

		{"nextcloud default book", "/remote.php/dav/addressbooks/users/admin/contacts/", false},
		{"prefix without the separator", "/remote.php/dav/addressbooks/users/admin/z-server-generated/", false},
		{"prefix not at the start", "/remote.php/dav/addressbooks/users/admin/my-z-app-generated--book/", false},
		{"radicale book", "/testuser/book-1/", false},
		{"meerkat's own book", "/carddav/addressbooks/remote/contacts/", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isGeneratedAddressBook(tc.path); got != tc.want {
				t.Errorf("isGeneratedAddressBook(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

// TestDiscoverFiltersGeneratedAddressBooks drives the real discovery path
// against a server that actually holds a server-managed collection. Nextcloud
// creates these itself; here one is staged by name so the behaviour is covered
// without needing a Nextcloud instance.
func TestDiscoverFiltersGeneratedAddressBooks(t *testing.T) {
	for _, backend := range contactSyncBackends() {
		if backend.name == "meerkat" {
			continue // serves a single fixed address book, so nothing to filter
		}
		t.Run(backend.name, func(t *testing.T) {
			remote := backend.create(t)

			generatedPath := radicaleBookRoot + "z-app-generated--contactsinteraction--recent/"
			makeAddressBook(t, remote, generatedPath, "Recently contacted", "staging a generated book")

			books, err := NewContactSyncService(false).Discover(
				context.Background(), remote.proxyURL, remote.user, remote.pass)
			require.NoError(t, err)

			var paths []string
			for _, book := range books {
				paths = append(paths, book.Path)
			}
			assert.NotContains(t, paths, generatedPath,
				"server-managed address books must not be offered as sync targets")
			assert.Contains(t, paths, remote.addressBookPath(),
				"ordinary address books must still be discoverable")
		})
	}
}
