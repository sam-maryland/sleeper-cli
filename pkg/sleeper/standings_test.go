package sleeper_test

import (
	"context"
	"testing"

	"github.com/sam-maryland/sleeper-cli/pkg/sleeper"
)

// mockClient implements ISleeperClient with configurable return values.
type mockClient struct {
	league        sleeper.SleeperLeague
	users         sleeper.SleeperUsers
	rosters       sleeper.Rosters
	matchupsByWk  map[int]sleeper.Matchups
	bracket       []sleeper.BracketMatchup
}

func (m *mockClient) GetLeague(_ context.Context, _ string) (sleeper.SleeperLeague, error) {
	return m.league, nil
}
func (m *mockClient) GetUsersInLeague(_ context.Context, _ string) (sleeper.SleeperUsers, error) {
	return m.users, nil
}
func (m *mockClient) GetRostersInLeague(_ context.Context, _ string) (sleeper.Rosters, error) {
	return m.rosters, nil
}
func (m *mockClient) GetMatchupsForWeek(_ context.Context, _ string, week int) (sleeper.Matchups, error) {
	return m.matchupsByWk[week], nil
}
func (m *mockClient) GetWinnersBracket(_ context.Context, _ string) ([]sleeper.BracketMatchup, error) {
	return m.bracket, nil
}
func (m *mockClient) GetUser(_ context.Context, _ string) (sleeper.SleeperUser, error) {
	return sleeper.SleeperUser{}, nil
}
func (m *mockClient) GetNFLState(_ context.Context) (sleeper.NFLState, error) {
	return sleeper.NFLState{}, nil
}
func (m *mockClient) FetchAllPlayers(_ context.Context) ([]byte, error) {
	return nil, nil
}

// --- helpers ---

func league(status string, playoffWeekStart, playoffTeams int) sleeper.SleeperLeague {
	return sleeper.SleeperLeague{
		Status: status,
		Season: "2025",
		Settings: sleeper.LeagueSettings{
			PlayoffWeekStart: playoffWeekStart,
			PlayoffTeams:     playoffTeams,
		},
	}
}

func user(id, name string) sleeper.SleeperUser {
	return sleeper.SleeperUser{ID: id, DisplayName: name}
}

func roster(id int, ownerID string, wins, losses int, pf, pa float64) sleeper.Roster {
	pfInt := int(pf)
	pfDec := float32(pf - float64(pfInt))
	paInt := int(pa)
	paDec := float32(pa - float64(paInt))
	return sleeper.Roster{
		ID:      id,
		OwnerID: ownerID,
		Settings: sleeper.RosterSettings{
			Wins:                 wins,
			Losses:               losses,
			PointsFor:            pfInt,
			PointsForDecimal:     pfDec * 100,
			PointsAgainst:        paInt,
			PointsAgainstDecimal: paDec * 100,
		},
	}
}

func matchup(matchupID, rosterID int, points float64) sleeper.Matchup {
	return sleeper.Matchup{MatchupID: matchupID, RosterID: rosterID, Points: points}
}

func bracket(matchupID, round, winner, loser, placement int) sleeper.BracketMatchup {
	return sleeper.BracketMatchup{
		MatchupID: matchupID,
		Round:     round,
		Winner:    winner,
		Loser:     loser,
		Team1:     winner,
		Team2:     loser,
		Placement: placement,
	}
}

// --- tests ---

// TestRegularSeasonStandings verifies basic win-based ordering for an in-progress league.
func TestRegularSeasonStandings(t *testing.T) {
	// Three teams: A(10W), B(8W), C(6W)
	c := &mockClient{
		league: league("in_season", 14, 6),
		users: sleeper.SleeperUsers{
			user("A", "Alice"),
			user("B", "Bob"),
			user("C", "Charlie"),
		},
		rosters: sleeper.Rosters{
			roster(1, "A", 10, 3, 1500, 1200),
			roster(2, "B", 8, 5, 1400, 1300),
			roster(3, "C", 6, 7, 1300, 1400),
		},
		// Minimal matchup data for weeks 1–13 (all empty — H2H won't tiebreak here)
		matchupsByWk: map[int]sleeper.Matchups{},
	}

	standings, _, err := sleeper.CalculateStandings(context.Background(), c, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(standings) != 3 {
		t.Fatalf("expected 3 standings, got %d", len(standings))
	}
	wantOrder := []string{"A", "B", "C"}
	for i, want := range wantOrder {
		if standings[i].OwnerID != want {
			t.Errorf("position %d: want owner %q, got %q", i+1, want, standings[i].OwnerID)
		}
	}
}

// TestTiebreakerH2H verifies that H2H record breaks a win tie correctly.
func TestTiebreakerH2H(t *testing.T) {
	// A and B both have 8 wins. A beat B head-to-head, so A ranks higher.
	c := &mockClient{
		league: league("in_season", 14, 6),
		users: sleeper.SleeperUsers{
			user("A", "Alice"),
			user("B", "Bob"),
		},
		rosters: sleeper.Rosters{
			roster(1, "A", 8, 5, 1200, 1100),
			roster(2, "B", 8, 5, 1210, 1090), // higher PF, but should lose tiebreak to H2H
		},
		matchupsByWk: map[int]sleeper.Matchups{
			// Week 1: A (roster 1) beats B (roster 2)
			1: {matchup(1, 1, 120), matchup(1, 2, 100)},
		},
	}

	standings, _, err := sleeper.CalculateStandings(context.Background(), c, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if standings[0].OwnerID != "A" {
		t.Errorf("expected A first (H2H winner), got %q", standings[0].OwnerID)
	}
	if standings[1].OwnerID != "B" {
		t.Errorf("expected B second, got %q", standings[1].OwnerID)
	}
}

// TestTiebreakerPointsFor verifies that PF breaks a tie when H2H is even.
func TestTiebreakerPointsFor(t *testing.T) {
	// A and B tied in wins and never played each other — PF decides.
	c := &mockClient{
		league: league("in_season", 14, 6),
		users: sleeper.SleeperUsers{
			user("A", "Alice"),
			user("B", "Bob"),
		},
		rosters: sleeper.Rosters{
			roster(1, "A", 8, 5, 1300, 1100), // higher PF
			roster(2, "B", 8, 5, 1200, 1100),
		},
		matchupsByWk: map[int]sleeper.Matchups{},
	}

	standings, _, err := sleeper.CalculateStandings(context.Background(), c, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if standings[0].OwnerID != "A" {
		t.Errorf("expected A first (higher PF), got %q", standings[0].OwnerID)
	}
}

// TestCompletedSeasonFinalRanks verifies playoff bracket rankings are applied
// correctly for a completed season with a 4-team bracket (2 placement games).
func TestCompletedSeasonFinalRanks(t *testing.T) {
	// Regular season order: A(10W), B(9W), C(8W), D(7W)
	// Playoff bracket: C beats A in championship (p=1), B beats D for 3rd (p=3)
	// Final: C=1, A=2, B=3, D=4
	c := &mockClient{
		league: league("complete", 14, 4),
		users: sleeper.SleeperUsers{
			user("A", "Alice"),
			user("B", "Bob"),
			user("C", "Charlie"),
			user("D", "Dave"),
		},
		rosters: sleeper.Rosters{
			roster(1, "A", 10, 3, 1500, 1000),
			roster(2, "B", 9, 4, 1400, 1100),
			roster(3, "C", 8, 5, 1300, 1200),
			roster(4, "D", 7, 6, 1200, 1300),
		},
		matchupsByWk: map[int]sleeper.Matchups{},
		bracket: []sleeper.BracketMatchup{
			// Round 1 (SF): no placement
			bracket(1, 1, 3, 2, 0), // C beats B
			bracket(2, 1, 1, 4, 0), // A beats D... wait, need correct losers for 3rd
			// Round 2 (Finals): placement games
			bracket(3, 2, 3, 1, 1), // C beats A → C=1st, A=2nd
			bracket(4, 2, 2, 4, 3), // B beats D → B=3rd, D=4th
		},
	}

	standings, _, err := sleeper.CalculateStandings(context.Background(), c, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantRanks := map[string]int{"C": 1, "A": 2, "B": 3, "D": 4}
	for _, s := range standings {
		want, ok := wantRanks[s.OwnerID]
		if !ok {
			continue
		}
		if s.FinalRank != want {
			t.Errorf("owner %q: want FinalRank %d, got %d", s.OwnerID, want, s.FinalRank)
		}
	}
}

// TestCompletedSeasonOrdering verifies that playoff-ranked teams sort above
// non-playoff teams in the final standings slice.
func TestCompletedSeasonOrdering(t *testing.T) {
	// 6 teams, top 4 in playoffs, bottom 2 non-playoff.
	// Bracket places: E=1, F=2, A=3, B=4; C and D are non-playoff (sorted by regular season).
	c := &mockClient{
		league: league("complete", 13, 4),
		users: sleeper.SleeperUsers{
			user("A", "Alice"), user("B", "Bob"), user("C", "Carol"),
			user("D", "Dave"), user("E", "Eve"), user("F", "Frank"),
		},
		rosters: sleeper.Rosters{
			roster(1, "A", 10, 2, 1600, 1000),
			roster(2, "B", 9, 3, 1500, 1100),
			roster(3, "C", 5, 7, 1100, 1500),
			roster(4, "D", 4, 8, 1000, 1600),
			roster(5, "E", 8, 4, 1400, 1200),
			roster(6, "F", 7, 5, 1300, 1300),
		},
		matchupsByWk: map[int]sleeper.Matchups{},
		bracket: []sleeper.BracketMatchup{
			bracket(1, 2, 5, 1, 1), // E beats A → E=1st, A=2nd
			bracket(2, 2, 6, 2, 3), // F beats B → F=3rd, B=4th
		},
	}

	standings, _, err := sleeper.CalculateStandings(context.Background(), c, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(standings) != 6 {
		t.Fatalf("expected 6 standings, got %d", len(standings))
	}

	// First 4 must be playoff-ranked
	for i := 0; i < 4; i++ {
		if standings[i].FinalRank == 0 {
			t.Errorf("position %d (%q): expected playoff rank, got 0", i+1, standings[i].OwnerID)
		}
	}
	// Last 2 must be non-playoff, in regular season order (C then D), with assigned ranks
	if standings[4].OwnerID != "C" {
		t.Errorf("position 5: want C, got %q", standings[4].OwnerID)
	}
	if standings[4].FinalRank != 5 {
		t.Errorf("C: want FinalRank 5, got %d", standings[4].FinalRank)
	}
	if standings[5].OwnerID != "D" {
		t.Errorf("position 6: want D, got %q", standings[5].OwnerID)
	}
	if standings[5].FinalRank != 6 {
		t.Errorf("D: want FinalRank 6, got %d", standings[5].FinalRank)
	}

	// Spot-check playoff positions
	rankOf := make(map[string]int)
	for _, s := range standings {
		rankOf[s.OwnerID] = s.FinalRank
	}
	if rankOf["E"] != 1 {
		t.Errorf("E: want FinalRank 1, got %d", rankOf["E"])
	}
	if rankOf["F"] != 3 {
		t.Errorf("F: want FinalRank 3, got %d", rankOf["F"])
	}
}
