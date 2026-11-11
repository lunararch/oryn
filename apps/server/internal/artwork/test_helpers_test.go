package artwork

import "github.com/lunararch/oryn/apps/server/internal/store/db"

func getArtworkParams(gameID, kind string) db.GetArtworkForGameParams {
	return db.GetArtworkForGameParams{GameID: gameID, Kind: kind}
}
