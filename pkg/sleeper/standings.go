package sleeper

import (
	"context"
	"fmt"
	"sort"
)

// Standing represents a team's computed position in the league standings.
type Standing struct {
	OwnerID       string
	DisplayName   string
	TeamName      string
	Wins          int
	Losses        int
	Ties          int
	PointsFor     float64
	PointsAgainst float64
	H2HWins       map[string]int // wins against each specific opponent
	FinalRank     int            // 0 for in-progress seasons; 1-N for completed
}

// CalculateStandings fetches all necessary data from the Sleeper API and returns
// sorted standings for the given league. For completed leagues it applies
// playoff-based ranking for the top finishers.
func CalculateStandings(ctx context.Context, c ISleeperClient, leagueID string) ([]Standing, SleeperLeague, error) {
	league, err := c.GetLeague(ctx, leagueID)
	if err != nil {
		return nil, SleeperLeague{}, fmt.Errorf("fetching league: %w", err)
	}

	users, err := c.GetUsersInLeague(ctx, leagueID)
	if err != nil {
		return nil, SleeperLeague{}, fmt.Errorf("fetching users: %w", err)
	}

	rosters, err := c.GetRostersInLeague(ctx, leagueID)
	if err != nil {
		return nil, SleeperLeague{}, fmt.Errorf("fetching rosters: %w", err)
	}

	userMap := users.ToMap()

	// rosterID → ownerID for matchup pairing
	rosterOwner := make(map[int]string, len(rosters))
	for _, r := range rosters {
		rosterOwner[r.ID] = r.OwnerID
	}

	// Seed standings from roster data (Sleeper maintains accurate aggregate W/L/PF/PA)
	standingsMap := make(map[string]*Standing, len(rosters))
	for _, r := range rosters {
		if r.OwnerID == "" {
			continue
		}
		u := userMap[r.OwnerID]
		standingsMap[r.OwnerID] = &Standing{
			OwnerID:       r.OwnerID,
			DisplayName:   u.DisplayName,
			TeamName:      u.TeamName(),
			Wins:          r.Settings.Wins,
			Losses:        r.Settings.Losses,
			Ties:          r.Settings.Ties,
			PointsFor:     r.GetPointsFor(),
			PointsAgainst: r.GetPointsAgainst(),
			H2HWins:       make(map[string]int),
		}
	}

	// Build H2H records from regular season matchup data
	regularSeasonEnd := league.Settings.PlayoffWeekStart - 1
	if league.Settings.PlayoffWeekStart == 0 {
		// League hasn't set playoff week yet; use a safe default
		regularSeasonEnd = 14
	}
	for week := 1; week <= regularSeasonEnd; week++ {
		matchups, err := c.GetMatchupsForWeek(ctx, leagueID, week)
		if err != nil {
			return nil, SleeperLeague{}, fmt.Errorf("fetching week %d: %w", week, err)
		}
		for _, pair := range matchups.GroupByMatchupID() {
			if len(pair) != 2 {
				continue
			}
			a := rosterOwner[pair[0].RosterID]
			b := rosterOwner[pair[1].RosterID]
			if a == "" || b == "" {
				continue
			}
			if pair[0].Points > pair[1].Points {
				standingsMap[a].H2HWins[b]++
			} else if pair[1].Points > pair[0].Points {
				standingsMap[b].H2HWins[a]++
			}
		}
	}

	regularStandings := sortByRecord(standingsMap)

	if league.Status == "complete" {
		if err := applyPlayoffRanking(ctx, c, leagueID, league, rosterOwner, standingsMap); err != nil {
			// Playoff data unavailable; return regular season standings rather than failing
			return regularStandings, league, nil
		}
		return buildFinalStandings(standingsMap, regularStandings), league, nil
	}

	return regularStandings, league, nil
}

// sortByRecord sorts standings: wins desc → H2H within tied group desc → PF desc → PA asc
func sortByRecord(sm map[string]*Standing) []Standing {
	groups := make(map[int][]*Standing)
	for _, s := range sm {
		groups[s.Wins] = append(groups[s.Wins], s)
	}

	winCounts := make([]int, 0, len(groups))
	for w := range groups {
		winCounts = append(winCounts, w)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(winCounts)))

	var result []Standing
	for _, wc := range winCounts {
		group := groups[wc]
		sortGroupWithTiebreakers(group)
		for _, s := range group {
			result = append(result, *s)
		}
	}
	return result
}

func sortGroupWithTiebreakers(group []*Standing) {
	sort.SliceStable(group, func(i, j int) bool {
		// H2H wins within the group only
		h2hI, h2hJ := 0, 0
		for _, s := range group {
			if s.OwnerID == group[i].OwnerID || s.OwnerID == group[j].OwnerID {
				continue
			}
			h2hI += group[i].H2HWins[s.OwnerID]
			h2hJ += group[j].H2HWins[s.OwnerID]
		}
		// Also count direct H2H between i and j
		h2hI += group[i].H2HWins[group[j].OwnerID]
		h2hJ += group[j].H2HWins[group[i].OwnerID]

		if h2hI != h2hJ {
			return h2hI > h2hJ
		}
		if group[i].PointsFor != group[j].PointsFor {
			return group[i].PointsFor > group[j].PointsFor
		}
		return group[i].PointsAgainst < group[j].PointsAgainst
	})
}

// applyPlayoffRanking assigns FinalRank based on the winners bracket from the Sleeper API.
// Bracket games with a non-zero Placement field are final-standings games
// (e.g. p=1 → championship, p=3 → 3rd place, p=5 → 5th place).
func applyPlayoffRanking(
	ctx context.Context,
	c ISleeperClient,
	leagueID string,
	_ SleeperLeague,
	rosterOwner map[int]string,
	standingsMap map[string]*Standing,
) error {
	bracket, err := c.GetWinnersBracket(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("fetching winners bracket: %w", err)
	}

	for _, match := range bracket {
		if match.Placement == 0 {
			continue
		}
		winner := rosterOwner[match.Winner]
		loser := rosterOwner[match.Loser]
		if s, ok := standingsMap[winner]; ok {
			s.FinalRank = match.Placement
		}
		if s, ok := standingsMap[loser]; ok {
			s.FinalRank = match.Placement + 1
		}
	}

	return nil
}

// buildFinalStandings produces a fully sorted slice for a completed season.
// Playoff-ranked teams (1–playoffTeams) come first, remaining teams follow
// in regular-season order.
func buildFinalStandings(standingsMap map[string]*Standing, regularStandings []Standing) []Standing {
	regRank := make(map[string]int, len(regularStandings))
	for i, s := range regularStandings {
		regRank[s.OwnerID] = i
	}

	all := make([]*Standing, 0, len(standingsMap))
	for _, s := range standingsMap {
		all = append(all, s)
	}

	sort.Slice(all, func(i, j int) bool {
		fi, fj := all[i].FinalRank, all[j].FinalRank
		if fi > 0 && fj > 0 {
			return fi < fj
		}
		if fi > 0 {
			return true
		}
		if fj > 0 {
			return false
		}
		return regRank[all[i].OwnerID] < regRank[all[j].OwnerID]
	})

	result := make([]Standing, len(all))
	for i, s := range all {
		result[i] = *s
		if result[i].FinalRank == 0 {
			result[i].FinalRank = i + 1
		}
	}
	return result
}
