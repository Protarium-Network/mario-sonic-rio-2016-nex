# Protocol coverage

What this server implements for Mario & Sonic at the Rio 2016 Olympic Games,
and how each was confirmed.

## Authentication endpoint

| Protocol         | Methods                | Notes |
|------------------|------------------------|-------|
| Ticket Granting  | Login, LoginEx, RequestTicket | `ValidateLoginData` accepts the account-server token (or anything, in local mode). Secure StationURL points at `PN_RIO2016_SECURE_HOST:PN_RIO2016_SECURE_PORT`. |

`ByteStreamSettings.UseStructureHeader = true` — the client writes Structure
headers (version + length) for both the Data and AuthenticationInfo layers.
Confirmed against a 112-byte `LoginEx` RMC-parameter capture; without it
`Token` is read misaligned and comes out empty.

## Secure endpoint

| Protocol            | Coverage | Confirmed by |
|---------------------|----------|--------------|
| Secure Connection   | baseline handshake, `Register` (insecure register enabled) | required for any secure endpoint |
| Utility             | baseline | required baseline calls |
| Ranking (lib 3.6.0) | `GetRankings`, `GetRankingsAndCountByCategoryAndRankingOrderParam`, `UploadScore` (`InsertRankingByPIDAndRankingScoreData`), `GetCommonData`, `UploadCommonData` | leaderboard list + score submission against real hardware |
| DataStore           | `PostMetaBinary` (0x73/0x15), `PrepareGetObject` (by DataID and by persistence target), `PreparePostObject` → S3 presigned PUT → `CompletePostObject`, object period / meta-binary / data-type updates | real console logs: `PostMetaBinary` on score submit; `GetObjectInfoByPersistenceTargetWithPassword not defined` once the ranking list returned real data |

### Ranking modes

`Range` (0) and `User` (4) as in the base protocol, plus `Near` (1) and
friend-range (2). Rio 2016 has no friends system here, so friend-range runs
the same query as `User` (the protocol's own documented fallback).

## Not implemented

- Matchmaking / NAT traversal — the title's online events are leaderboard
  only in this deployment.
- Service Item — not used by this title.
