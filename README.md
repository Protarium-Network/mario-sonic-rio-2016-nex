# Mario & Sonic at the Rio 2016 Olympic Games — NEX server

A preservation-oriented NEX server for the Wii U title **Mario & Sonic at the
Rio 2016 Olympic Games** (EUR retail, `game_server_id` `10190300`). It speaks
the game's PRUDP authentication and secure protocols and stores leaderboards
and score attachments in PostgreSQL.

Built on the [Pretendo Network](https://github.com/PretendoNetwork) NEX
libraries (AGPL-3.0). `internal/nex-protocols-common-go-patch/` is a vendored
fork of `nex-protocols-common-go` carrying the DataStore/S3 changes this
title's score-upload flow depends on.

## Recovered configuration

| Field                | Value                                             |
|----------------------|---------------------------------------------------|
| Game server ID       | `10190300`                                        |
| Access key           | `63fecb0f` (first UTF-16BE string in `unison.rpx`)|
| NEX SDK version       | `3.4.7`, Ranking library bumped to `3.6.0`        |
| Byte stream          | Structure headers on (`UseStructureHeader = true`)|

The `3.6.0` Ranking bump makes `RankingRankData.UpdateTime` serialize;
without it every leaderboard row after the first is misread client-side. The
Structure-header layout was confirmed byte-for-byte against a real 2024
`LoginEx` capture.

## Scope

Only what the title actually calls:

- **Ticket Granting** — login / secure-server handoff
- **Secure Connection**, **Utility** — baseline secure-endpoint handshake
- **Ranking** — leaderboards (`GetRankings`, `GetRankingsAndCount`,
  `UploadScore`, common data get/upload). Range, User and Near modes;
  friend-range degrades to User (no friends system for this title).
- **DataStore** — the score-attachment flow: `PostMetaBinary`,
  `PrepareGetObject` (ghost/replay download by DataID or persistence slot),
  `PreparePostObject` → S3 presigned upload → object metadata updates.

No matchmaking, no service-item.

## Running

### Local preservation mode (self-contained)

```bash
cp .env.example .env          # PN_RIO2016_LOCAL_MODE=1 is the default
cp settings.example.json settings.json   # add your console's PID + NEX password
docker compose up --build
```

In local mode there is no account server: player NEX passwords come from
`settings.json` and the login token is accepted unconditionally. Use it only
on an isolated network.

### Shared mode (behind an account server)

Set `PN_RIO2016_LOCAL_MODE` to anything but `1` and provide:

- `PN_RIO2016_NEX_TOKEN_AES_KEY` — 64 hex chars, matches the account server's
  NEX token key
- `PN_RIO2016_NEX_PASSWORD_SECRET` — ≥32 bytes hex, matches the account
  server's password secret (per-PID passwords are
  `HMAC-SHA256(secret, pid)`)

### Without Docker

```bash
go build -o rio2016-nex .
./rio2016-nex
```

Requires a reachable PostgreSQL (`PN_RIO2016_POSTGRES_URI`); the schema is
created on first start.

## Configuration

See [`.env.example`](.env.example) for every variable. Notes:

- `PN_RIO2016_SECURE_HOST` **must be short** (~15 chars). The retail binary
  truncates it into a fixed buffer — a console-side DNS capture showed
  `wsc.protarium.lol` arriving as `wsc.protarium.l`.
- DataStore score attachments need an S3-compatible endpoint
  (`PN_S3_ENDPOINT` etc.). Without one, `PreparePostObject` returns
  `NotImplemented` and the client aborts the whole score submission.

## Leaderboard admin panel

Set `PN_RIO2016_ADMIN_PASSWORD` to enable a minimal HTTP panel (HTTP basic
auth, server-rendered) for adding, editing and deleting ranking rows without
touching SQL. Listens on `PN_RIO2016_ADMIN_LISTEN` (default `:8090`). Leave
the password unset to disable it.

## License

AGPL-3.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
