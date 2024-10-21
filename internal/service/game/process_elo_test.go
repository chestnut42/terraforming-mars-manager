package game

import (
	"context"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
	"gotest.tools/v3/assert"
	"testing"
)

func TestUpdateElo(t *testing.T) {
	tests := []struct {
		name    string
		state   storage.EloUpdateState
		want    storage.EloResults
		wantErr error
	}{
		{
			name: "two players - equal elo - p1 wins by vp",
			state: storage.EloUpdateState{
				Game: storage.Game{
					Players: []storage.Player{
						{UserId: "u1", PlayerId: "p1"},
						{UserId: "u2", PlayerId: "p2"},
					},
					GameResults: &storage.GameResults{
						Raw: map[string]any{"players": []map[string]any{
							{
								"id": "p1",
								"victoryPointsBreakdown": map[string]any{
									"total": 42,
								},
							},
							{
								"id": "p2",
								"victoryPointsBreakdown": map[string]any{
									"total": 40,
								},
							},
						}},
					},
				},
				Users: []storage.EloStateUser{
					{UserId: "u1", Elo: 1000},
					{UserId: "u2", Elo: 1000},
				},
			},
			want: storage.EloResults{
				Pairs: []storage.EloResultsPair{
					{LeftPlayerId: "p1", RightPlayerId: "p2", LeftPlayerElo: 1000, RightPlayerElo: 1000,
						LeftEloDelta: 10, LeftPlayerScore: 1},
				},
				Players: []storage.EloResultsPlayer{
					{UserId: "u1", PlayerId: "p1", OldElo: 1000, NewElo: 1010},
					{UserId: "u2", PlayerId: "p2", OldElo: 1000, NewElo: 990},
				},
			},
		},
		{
			name: "two players - equal elo - p1 wins by mega credits",
			state: storage.EloUpdateState{
				Game: storage.Game{
					Players: []storage.Player{
						{UserId: "u1", PlayerId: "p1"},
						{UserId: "u2", PlayerId: "p2"},
					},
					GameResults: &storage.GameResults{
						Raw: map[string]any{"players": []map[string]any{
							{
								"id": "p1",
								"victoryPointsBreakdown": map[string]any{
									"total": 42,
								},
								"megaCredits": 82,
							},
							{
								"id": "p2",
								"victoryPointsBreakdown": map[string]any{
									"total": 42,
								},
								"megaCredits": 81,
							},
						}},
					},
				},
				Users: []storage.EloStateUser{
					{UserId: "u1", Elo: 1000},
					{UserId: "u2", Elo: 1000},
				},
			},
			want: storage.EloResults{
				Pairs: []storage.EloResultsPair{
					{LeftPlayerId: "p1", RightPlayerId: "p2", LeftPlayerElo: 1000, RightPlayerElo: 1000,
						LeftEloDelta: 10, LeftPlayerScore: 1},
				},
				Players: []storage.EloResultsPlayer{
					{UserId: "u1", PlayerId: "p1", OldElo: 1000, NewElo: 1010},
					{UserId: "u2", PlayerId: "p2", OldElo: 1000, NewElo: 990},
				},
			},
		},
		{
			name: "two players - equal elo - draw",
			state: storage.EloUpdateState{
				Game: storage.Game{
					Players: []storage.Player{
						{UserId: "u1", PlayerId: "p1"},
						{UserId: "u2", PlayerId: "p2"},
					},
					GameResults: &storage.GameResults{
						Raw: map[string]any{"players": []map[string]any{
							{
								"id": "p1",
								"victoryPointsBreakdown": map[string]any{
									"total": 42,
								},
								"megaCredits": 82,
							},
							{
								"id": "p2",
								"victoryPointsBreakdown": map[string]any{
									"total": 42,
								},
								"megaCredits": 82,
							},
						}},
					},
				},
				Users: []storage.EloStateUser{
					{UserId: "u1", Elo: 1000},
					{UserId: "u2", Elo: 1000},
				},
			},
			want: storage.EloResults{
				Pairs: []storage.EloResultsPair{
					{LeftPlayerId: "p1", RightPlayerId: "p2", LeftPlayerElo: 1000, RightPlayerElo: 1000,
						LeftEloDelta: 0, LeftPlayerScore: 0.5},
				},
				Players: []storage.EloResultsPlayer{
					{UserId: "u1", PlayerId: "p1", OldElo: 1000, NewElo: 1000},
					{UserId: "u2", PlayerId: "p2", OldElo: 1000, NewElo: 1000},
				},
			},
		},
		{
			name: "four players",
			state: storage.EloUpdateState{
				Game: storage.Game{
					Players: []storage.Player{
						{UserId: "u1", PlayerId: "p1"},
						{UserId: "u2", PlayerId: "p2"},
						{UserId: "u3", PlayerId: "p3"},
						{UserId: "u4", PlayerId: "p4"},
					},
					GameResults: &storage.GameResults{
						Raw: map[string]any{"players": []map[string]any{
							{
								"id": "p2",
								"victoryPointsBreakdown": map[string]any{
									"total": 100,
								},
							},
							{
								"id": "p4",
								"victoryPointsBreakdown": map[string]any{
									"total": 90,
								},
							},
							{
								"id": "p1",
								"victoryPointsBreakdown": map[string]any{
									"total": 80,
								},
							},
							{
								"id": "p3",
								"victoryPointsBreakdown": map[string]any{
									"total": 80,
								},
							},
						}},
					},
				},
				Users: []storage.EloStateUser{
					{UserId: "u1", Elo: 1000},
					{UserId: "u2", Elo: 1240},
					{UserId: "u3", Elo: 1480},
					{UserId: "u4", Elo: 100},
				},
			},
			want: storage.EloResults{
				Pairs: []storage.EloResultsPair{
					{LeftPlayerId: "p2", RightPlayerId: "p4", LeftPlayerElo: 1240, RightPlayerElo: 100,
						LeftEloDelta: 1, LeftPlayerScore: 1},
					{LeftPlayerId: "p2", RightPlayerId: "p1", LeftPlayerElo: 1240, RightPlayerElo: 1000,
						LeftEloDelta: 5, LeftPlayerScore: 1},
					{LeftPlayerId: "p2", RightPlayerId: "p3", LeftPlayerElo: 1240, RightPlayerElo: 1480,
						LeftEloDelta: 16, LeftPlayerScore: 1},
					{LeftPlayerId: "p4", RightPlayerId: "p1", LeftPlayerElo: 100, RightPlayerElo: 1000,
						LeftEloDelta: 20, LeftPlayerScore: 1},
					{LeftPlayerId: "p4", RightPlayerId: "p3", LeftPlayerElo: 100, RightPlayerElo: 1480,
						LeftEloDelta: 20, LeftPlayerScore: 1},
					{LeftPlayerId: "p1", RightPlayerId: "p3", LeftPlayerElo: 1000, RightPlayerElo: 1480,
						LeftEloDelta: 9, LeftPlayerScore: 0.5},
				},
				Players: []storage.EloResultsPlayer{
					{UserId: "u2", PlayerId: "p2", OldElo: 1240, NewElo: 1262},
					{UserId: "u4", PlayerId: "p4", OldElo: 100, NewElo: 139},
					{UserId: "u1", PlayerId: "p1", OldElo: 1000, NewElo: 984},
					{UserId: "u3", PlayerId: "p3", OldElo: 1480, NewElo: 1435},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			got, err := updateElo(ctx, tt.state)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, got, tt.want)
		})
	}
}
