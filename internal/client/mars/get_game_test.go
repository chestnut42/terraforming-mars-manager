package mars

import (
	_ "embed"
	"testing"

	"gotest.tools/v3/assert"
)

//go:embed test_get_game_response.json
var testGetGameResponse []byte

func TestGetGameDAO(t *testing.T) {
	resp, err := readResponse(testGetGameResponse)
	assert.NilError(t, err)

	assert.DeepEqual(t, resp.Game, GetGameModel{
		HasFinished: true,
		Players: []GetGamePlayer{
			{
				Id:          "pfd7bca2ed0cb",
				MegaCredits: 83,
				Score:       136,
			},
			{
				Id:          "p53cdbf44f911",
				MegaCredits: 66,
				Score:       122,
			},
		},
	})

	got, err := GetGameResponseFromRaw(resp.Raw)
	assert.NilError(t, err)
	assert.DeepEqual(t, got, resp)
}
