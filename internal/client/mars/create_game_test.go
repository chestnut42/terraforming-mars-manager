package mars

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/chestnut42/terraforming-mars-manager/pkg/api"
)

//go:embed test_create_game_request.json
var testRequest []byte

//go:embed test_create_game_response.json
var testResponse []byte

func TestDAO(t *testing.T) {
	jd := json.NewDecoder(bytes.NewReader(testRequest))
	jd.DisallowUnknownFields()

	var cg createGame
	err := jd.Decode(&cg)
	assert.NilError(t, err)

	var cgResp createGameResponse
	err = json.Unmarshal(testResponse, &cgResp)
	assert.NilError(t, err)
	assert.DeepEqual(t, cgResp, createGameResponse{
		Id: "g15db787ffe07",
		Players: []newPlayerResponse{
			{Id: "pd102a414e5e1", Name: "qweasd", Color: "orange"},
			{Id: "pe3f6d5f8be7e", Name: "asdqwe", Color: "yellow"},
		},
		PurgeDateMs: 1723486813151,
	})
}

func TestRequestPlayers(t *testing.T) {
	tests := []struct {
		name string
		in   []NewPlayer
		want []newPlayer
	}{
		{
			name: "no conflicts",
			in: []NewPlayer{
				{Name: "name 1", Color: api.PlayerColor_GREEN},
				{Name: "name 2", Color: api.PlayerColor_BLUE},
				{Name: "name 3", Color: api.PlayerColor_BLACK},
			},
			want: []newPlayer{
				{Name: "name 1", Color: "green"},
				{Name: "name 2", Color: "blue"},
				{Name: "name 3", Color: "black"},
			},
		},
		{
			name: "conflicts",
			in: []NewPlayer{
				{Name: "name 1", Color: api.PlayerColor_GREEN},
				{Name: "name 2", Color: api.PlayerColor_BLUE},
				{Name: "name 3", Color: api.PlayerColor_GREEN},
				{Name: "name 4", Color: api.PlayerColor_RED},
				{Name: "name 5", Color: api.PlayerColor_RED},
			},
			want: []newPlayer{
				{Name: "name 1", Color: "green"},
				{Name: "name 2", Color: "blue"},
				{Name: "name 3", Color: "yellow"},
				{Name: "name 4", Color: "red"},
				{Name: "name 5", Color: "black"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := requestPlayers(tt.in)

			firstCount := 0
			for i, v := range got {
				if v.First {
					firstCount++
				}
				got[i].First = false
			}
			assert.Equal(t, 1, firstCount)
			assert.DeepEqual(t, got, tt.want)
		})
	}
}
