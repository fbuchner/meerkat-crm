# T10 — WP-85 graph traversal + multi-hop chains

| | |
|---|---|
| **Rating** | 2 — a great demo and a rare query |
| **Size** | M–L |
| **Depends on** | — |
| **Alpha** | after |
| **Source** | `92.2` |

## Why the rating is low

"Teddy's owner", "John's sister's husband" reads impressively and is genuinely rarely reached for. The
graph *display* (already built, `NetworkPage`/`NetworkGraph.tsx`) delivers most of the practical value;
multi-hop *querying* is the part that will not see daily use.

**It is scheduled ahead of [T11](24-T11-search-fts5.md) (rated 5) purely because T11 depends on it** —
dependency beats rating. If T11's scope can be met without full traversal, revisit whether this is needed
at all.

## What to build

Per `92.2`:

- **Graph traversal + multi-hop chains via recursive CTEs** (SQLite supports `WITH RECURSIVE`) over
  `relationship_edges`.
- **Inferred relations computed, not stored** — a grandparent derived from two `parent_of` edges is
  never persisted. This is explicit in the roadmap and matches the model's existing discipline of never
  storing a reciprocal edge.

## What exists today

- `models.RelationshipEdge` — one direction stored; the inverse derived from
  `models/relationship_type_registry.go`'s `InverseRelationType`. Endpoints are `Contact.VCardUID`.
- `models/relationship_type_registry.go` — canonical tokens, inverses, symmetry, and `Synonyms`
  (currently consumed only by the removed WP-81 tool's matcher; T11 is its next real consumer).
- `controllers/graph_controller.go` → `GetGraph` — already builds a node/edge view filtered to
  `status: confirmed`, resolving `VCardUID` → node id via a map, skipping edges whose endpoints are not
  in the node set.

## Traps

- **Cycles.** Real relationship graphs contain them (mutual `friend_of`, a household loop). A recursive
  CTE without a visited-set or depth cap will not terminate. Cap depth explicitly.
- **Only `status: confirmed`** edges may participate. `GetGraph` already enforces this — match it.
- **Direction is not symmetric.** `parent_of` traversed backwards is `child_of`. Traversal must apply the
  registry's inverse when walking against an edge's stored direction, or every chain longer than one hop
  will be wrong. This is the single highest-risk piece of logic in the ticket — test it in both
  directions for every asymmetric token, the way `api/relationshipEdges.test.ts` does on the frontend.
- Sensitivity (`91.13`) — a `secret` edge should not leak into a traversal result that is displayed or
  exported.

## Done when

- `go build ./... && go vet ./... && gofmt -l . && go test ./...` green.
- Tests cover: a two-hop chain in both directions; an inferred grandparent from two `parent_of` edges; a
  cyclic graph terminating; the depth cap; a `suggested` edge being excluded.
- Hand-verified: remove the inverse application when traversing backwards, confirm the direction test
  fails, restore.
- Query performance sane on a seeded graph of a few thousand edges (recursive CTEs degrade fast without
  the right index — `relationship_edges` already indexes `source_id`, `target_id`, and `status`).
