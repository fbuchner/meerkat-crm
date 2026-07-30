-- Scope API tokens can be restricted to CardDAV-only use (Tier 3a item 1), so a leaked
-- synced-device credential can't be replayed against the general REST API. 'full' (default)
-- preserves today's behavior for every existing token; 'carddav' is new and intentionally
-- narrower -- rejected by AuthMiddleware's bearer-token path, accepted only by
-- carddav/auth.go's Basic-Auth fallback. 'full' still works for CardDAV too, since it already
-- grants broader REST access than CardDAV exposes.
ALTER TABLE api_tokens ADD COLUMN scope TEXT NOT NULL DEFAULT 'full';
