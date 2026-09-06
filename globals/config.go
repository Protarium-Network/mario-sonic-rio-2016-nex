package globals

// NEX configuration for "Mario & Sonic at the Rio 2016 Olympic Games"
// (EUR retail title 0x000500001019... , game_server_id 10190300).
//
// AccessKey was recovered from the retail unison.rpx (EUR digital title,
// 101e5400): the first isolated UTF-16BE string in the binary, in the same
// 8-char lowercase-hex form every other Wii U NEX title uses.
const (
	GameServerID = "10190300"
	AccessKey    = "63fecb0f"

	// PRUDP library version reported by both endpoints. Rio 2016 ships NEX
	// 3.4.7; the Ranking library alone is bumped to 3.6.0 at runtime (see
	// nex/secure.go) so RankingRankData.UpdateTime is serialized.
	NEXMajor = 3
	NEXMinor = 4
	NEXPatch = 7
)
