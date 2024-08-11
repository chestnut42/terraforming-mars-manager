package mars

import (
	_ "embed"
	"encoding/json"
	"strings"
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
				Id:    "pfd7bca2ed0cb",
				Score: 136,
			},
			{
				Id:    "p53cdbf44f911",
				Score: 122,
			},
		},
	})

	rawData, err := json.MarshalIndent(resp.Raw, "", "  ")
	assert.NilError(t, err)
	assert.Equal(t,
		len(strings.TrimSpace(string(rawData))),
		len(strings.TrimSpace(string(testGetGameResponse))))
}
