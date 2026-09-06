package nex

import (
	"fmt"
	"os"
	"strconv"

	nex "github.com/PretendoNetwork/nex-go/v2"
	common_datastore "github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore"
	common_ranking "github.com/PretendoNetwork/nex-protocols-common-go/v2/ranking"
	common_secure "github.com/PretendoNetwork/nex-protocols-common-go/v2/secure-connection"
	common_utility "github.com/PretendoNetwork/nex-protocols-common-go/v2/utility"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	ranking "github.com/PretendoNetwork/nex-protocols-go/v2/ranking"
	secure "github.com/PretendoNetwork/nex-protocols-go/v2/secure-connection"
	utility "github.com/PretendoNetwork/nex-protocols-go/v2/utility"
	"github.com/Protarium-Network/mario-sonic-rio-2016-nex/database"
	"github.com/Protarium-Network/mario-sonic-rio-2016-nex/globals"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var SecureServer *nex.PRUDPServer
var SecureEndpoint *nex.PRUDPEndPoint

func StartSecureServer() {
	SecureServer = nex.NewPRUDPServer()

	SecureEndpoint = nex.NewPRUDPEndPoint(1)
	SecureEndpoint.IsSecureEndPoint = true
	SecureEndpoint.ServerAccount = globals.SecureServerAccount
	SecureEndpoint.AccountDetailsByPID = globals.AccountDetailsByPID
	SecureEndpoint.AccountDetailsByUsername = globals.AccountDetailsByUsername
	SecureServer.BindPRUDPEndPoint(SecureEndpoint)

	SecureServer.LibraryVersions.SetDefault(nex.NewLibraryVersion(globals.NEXMajor, globals.NEXMinor, globals.NEXPatch))
	// RankingRankData.UpdateTime is only serialized when the Ranking library
	// version is >= 3.6.0 (nex-protocols-go). With the 3.4.7 default that
	// field gets silently dropped from every entry - invisible for a single
	// row (nothing after it to misread), but for 2+ rows every entry after
	// the first is read at the wrong offset client-side, corrupting the
	// whole response. Bump Ranking specifically; leave everything else
	// (DataStore, Main, etc.) at 3.4.7 since those flows are confirmed
	// working already.
	SecureServer.LibraryVersions.Ranking = nex.NewLibraryVersion(3, 6, 0)
	SecureServer.AccessKey = globals.AccessKey
	// See authentication.go: Rio 2016 writes Structure headers, confirmed
	// byte-for-byte against a real capture. Must match on both servers.
	SecureServer.ByteStreamSettings.UseStructureHeader = true

	SecureEndpoint.OnData(func(packet nex.PacketInterface) {
		request := packet.RMCMessage()
		if request == nil {
			return
		}
		pid := uint64(packet.Sender().PID())
		fmt.Printf("[Rio2016 Secure] PID=%d protocol=0x%02X method=0x%02X\n", pid, request.ProtocolID, request.MethodID)
	})

	SecureEndpoint.OnConnectionEnded(func(connection *nex.PRUDPConnection) {
		fmt.Printf("[Rio2016 Secure] PID=%d disconnected\n", uint64(connection.PID()))
	})

	registerSecureServerProtocols()

	port, _ := strconv.Atoi(os.Getenv("PN_RIO2016_SECURE_PORT"))
	globals.Logger.Successf("[Rio2016] Secure server listening on UDP %d", port)
	SecureServer.Listen(port)
}

// registerSecureServerProtocols wires up only what Mario & Sonic Rio 2016
// actually needs: the baseline secure-connection handshake (required for
// any secure endpoint), utility (baseline misc calls), ranking (leaderboards,
// the only gameplay-facing protocol this title uses) and the DataStore calls
// its score-attachment flow makes. No matchmaking, no service-item.
func registerSecureServerProtocols() {
	secureProtocol := secure.NewProtocol()
	SecureEndpoint.RegisterServiceProtocol(secureProtocol)
	secureCommon := common_secure.NewCommonProtocol(secureProtocol)
	secureCommon.EnableInsecureRegister()
	secureCommon.CreateReportDBRecord = database.CreateReportDBRecord

	utilityProtocol := utility.NewProtocol()
	SecureEndpoint.RegisterServiceProtocol(utilityProtocol)
	common_utility.NewCommonProtocol(utilityProtocol)

	rankingProtocol := ranking.NewProtocol()
	SecureEndpoint.RegisterServiceProtocol(rankingProtocol)
	rankingCommon := common_ranking.NewCommonProtocol(rankingProtocol)
	rankingCommon.GetRankingsAndCountByCategoryAndRankingOrderParam = database.Rio2016GetRankingsAndCountByCategoryAndRankingOrderParam
	rankingCommon.GetRankingsByMode = database.Rio2016GetRankings
	rankingCommon.GetCommonData = database.Rio2016GetCommonData
	rankingCommon.UploadCommonData = database.Rio2016UploadCommonData
	rankingCommon.InsertRankingByPIDAndRankingScoreData = database.Rio2016InsertRankingByPIDAndRankingScoreData

	// The score submission flow calls DataStore::PostMetaBinary (protocol
	// 0x73 method 0x15, confirmed from a real console log - attaches a
	// screenshot/ghost alongside the score). data_id is server-generated and
	// globally unique.
	datastoreProtocol := datastore.NewProtocol()
	SecureEndpoint.RegisterServiceProtocol(datastoreProtocol)
	datastoreCommon := common_datastore.NewCommonProtocol(datastoreProtocol)
	datastoreCommon.GetObjectInfosByDataStoreSearchParam = database.GetObjectInfosByDataStoreSearchParam
	datastoreCommon.InitializeObjectByPreparePostParam = database.InitializeObjectByPreparePostParam
	datastoreCommon.InitializeObjectRatingWithSlot = database.InitializeObjectRatingWithSlot
	datastoreCommon.GetObjectInfoByDataID = database.GetObjectInfoByDataID
	datastoreCommon.UpdateObjectPeriodByDataIDWithPassword = database.UpdateObjectPeriodByDataIDWithPassword
	datastoreCommon.UpdateObjectMetaBinaryByDataIDWithPassword = database.UpdateObjectMetaBinaryByDataIDWithPassword
	datastoreCommon.UpdateObjectDataTypeByDataIDWithPassword = database.UpdateObjectDataTypeByDataIDWithPassword
	// DataStore::PrepareGetObject (ghost/replay download) - needed for the
	// "race against other players' ghosts" feature. Confirmed missing via a
	// real console log ("GetObjectInfoByPersistenceTargetWithPassword not
	// defined") once the ranking list itself started returning real data.
	datastoreCommon.GetObjectInfoByDataIDWithPassword = database.GetObjectInfoByDataIDWithPassword
	datastoreCommon.GetObjectInfoByPersistenceTargetWithPassword = database.GetObjectInfoByPersistenceTargetWithPassword

	// DataStore::PreparePostObject (score-upload attachment, e.g. a ghost/photo
	// alongside the score) needs a real S3-compatible presigned-URL backend -
	// the Wii U client uploads the binary itself, directly to this URL, over
	// HTTP(S). Without it the call hard-fails with NotImplemented and the
	// client aborts the whole "send score" flow before ever calling
	// Ranking::UploadScore. Point PN_S3_ENDPOINT at any S3-compatible service
	// (self-hosted MinIO works well).
	s3Endpoint := os.Getenv("PN_S3_ENDPOINT")
	if s3Endpoint != "" {
		minioClient, err := minio.New(s3Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(os.Getenv("PN_S3_ACCESS_KEY"), os.Getenv("PN_S3_SECRET_KEY"), ""),
			Secure: true,
		})
		if err != nil {
			globals.Logger.Errorf("[Rio2016] Failed to create MinIO client: %s", err.Error())
		} else {
			s3Bucket := os.Getenv("PN_S3_BUCKET")
			if s3Bucket == "" {
				s3Bucket = "rio2016-datastore"
			}
			datastoreCommon.S3Bucket = s3Bucket
			datastoreCommon.SetDataKeyBase("rio2016")
			datastoreCommon.SetMinIOClient(minioClient)
		}
	} else {
		globals.Logger.Warning("[Rio2016] PN_S3_ENDPOINT not set - DataStore::PreparePostObject will fail")
	}
}
