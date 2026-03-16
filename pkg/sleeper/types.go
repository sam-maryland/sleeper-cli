package sleeper

import (
	"fmt"
	"sort"
)

// SleeperUser represents a user from the Sleeper API
type SleeperUser struct {
	ID          string       `json:"user_id"`
	Username    string       `json:"username"`
	DisplayName string       `json:"display_name"`
	Avatar      string       `json:"avatar"`
	Metadata    UserMetadata `json:"metadata"`
}

type UserMetadata struct {
	TeamName string `json:"team_name"`
}

func (u SleeperUser) TeamName() string {
	if u.Metadata.TeamName == "" {
		return u.DisplayName
	}
	return u.Metadata.TeamName
}

type SleeperUsers []SleeperUser

func (us SleeperUsers) WithID(id string) SleeperUser {
	for _, u := range us {
		if u.ID == id {
			return u
		}
	}
	return SleeperUser{}
}

func (us SleeperUsers) ToMap() map[string]SleeperUser {
	m := make(map[string]SleeperUser, len(us))
	for _, u := range us {
		m[u.ID] = u
	}
	return m
}

// NFLState represents the current NFL season state from the Sleeper API
type NFLState struct {
	Week            int    `json:"week"`
	DisplayWeek     int    `json:"display_week"`
	SeasonType      string `json:"season_type"`
	SeasonStartDate string `json:"season_start_date"`
	ActiveSeason    string `json:"season"`
	PreviousSeason  string `json:"previous_season"`
	Leg             int    `json:"leg"`
}

// Roster represents a fantasy roster from the Sleeper API
type Roster struct {
	ID       int            `json:"roster_id"`
	Players  []string       `json:"players"`
	Starters []string       `json:"starters"`
	Reserve  []string       `json:"reserve"`
	Taxi     []string       `json:"taxi"`
	Settings RosterSettings `json:"settings"`
	OwnerID  string         `json:"owner_id"`
	CoOwners []string       `json:"co_owners"`
	LeagueID string         `json:"league_id"`
	Metadata RosterMetadata `json:"metadata"`
}

type RosterMetadata struct {
	Streak string `json:"streak"`
	Record string `json:"record"`
}

type RosterSettings struct {
	Wins                 int     `json:"wins"`
	Losses               int     `json:"losses"`
	Ties                 int     `json:"ties"`
	WaiverPosition       int     `json:"waiver_position"`
	TotalMoves           int     `json:"total_moves"`
	MaxPoints            int     `json:"ppts"`
	MaxPointsDecimal     int     `json:"ppts_decimal"`
	PointsFor            int     `json:"fpts"`
	PointsForDecimal     float32 `json:"fpts_decimal"`
	PointsAgainst        int     `json:"fpts_against"`
	PointsAgainstDecimal float32 `json:"fpts_against_decimal"`
}

type Rosters []Roster

func (rl Rosters) WithID(id int) Roster {
	for _, r := range rl {
		if r.ID == id {
			return r
		}
	}
	return Roster{}
}

func (r Roster) GetPointsFor() float64 {
	return float64(r.Settings.PointsFor) + float64(r.Settings.PointsForDecimal)/100
}

func (r Roster) GetPointsAgainst() float64 {
	return float64(r.Settings.PointsAgainst) + float64(r.Settings.PointsAgainstDecimal)/100
}

// Matchup represents a single team's entry in a weekly matchup from the Sleeper API.
// Two entries sharing the same MatchupID form a head-to-head game.
type Matchup struct {
	MatchupID        int                `json:"matchup_id"`
	RosterID         int                `json:"roster_id"`
	Points           float64            `json:"points"`
	Players          []string           `json:"players"`
	Starters         []string           `json:"starters"`
	PlayersPointsMap map[string]float64 `json:"players_points,omitempty"`
}

type Matchups []Matchup

// GroupByMatchupID pairs individual team entries into head-to-head games.
// Bye weeks (MatchupID == 0) are excluded.
func (ms Matchups) GroupByMatchupID() map[int][]Matchup {
	grouped := make(map[int][]Matchup)
	for _, m := range ms {
		if m.MatchupID == 0 {
			continue
		}
		grouped[m.MatchupID] = append(grouped[m.MatchupID], m)
	}
	return grouped
}

// SleeperLeague represents the complete league data from the Sleeper API
type SleeperLeague struct {
	TotalRosters     int            `json:"total_rosters"`
	Status           string         `json:"status"`
	Sport            string         `json:"sport"`
	Settings         LeagueSettings `json:"settings"`
	SeasonType       string         `json:"season_type"`
	Season           string         `json:"season"`
	PreviousLeagueID string         `json:"previous_league_id"`
	Name             string         `json:"name"`
	LeagueID         string         `json:"league_id"`
	DraftID          string         `json:"draft_id"`
	Avatar           string         `json:"avatar"`
}

type LeagueSettings struct {
	PlayoffWeekStart int `json:"playoff_week_start"`
	PlayoffTeams     int `json:"playoff_teams"`
	PlayoffRounds    int `json:"playoff_rounds"`
	MaxKeepers       int `json:"max_keepers"`
	DraftRounds      int `json:"draft_rounds"`
	TradeDeadline    int `json:"trade_deadline"`
	ReserveSlots     int `json:"reserve_slots"`
	BenchSlots       int `json:"bench_slots"`
	WaiverType       int `json:"waiver_type"`
	WaiverBudget     int `json:"waiver_budget"`
	StartWeek        int `json:"start_week"`
	Leg              int `json:"leg"`
	BestBall         int `json:"best_ball"`
}

// Player represents NFL player data from the Sleeper API
type Player struct {
	PlayerID         string   `json:"player_id"`
	FirstName        string   `json:"first_name"`
	LastName         string   `json:"last_name"`
	FullName         string   `json:"full_name"`
	Position         string   `json:"position"`
	Team             string   `json:"team"`
	Status           string   `json:"status"`
	Active           bool     `json:"active"`
	Age              int      `json:"age"`
	YearsExp         int      `json:"years_exp"`
	FantasyPositions []string `json:"fantasy_positions"`
	Number           int      `json:"number"`
}

func (p Player) String() string {
	return fmt.Sprintf("%s - %s (%s)", p.Position, p.FullName, p.Team)
}

// BracketMatchup represents a single game in a playoff bracket from the Sleeper API.
// Games with a non-zero Placement field determine final standings positions.
type BracketMatchup struct {
	MatchupID int `json:"m"`
	Round     int `json:"r"`
	Winner    int `json:"w"` // roster ID
	Loser     int `json:"l"` // roster ID
	Team1     int `json:"t1"`
	Team2     int `json:"t2"`
	Placement int `json:"p"` // winner's final placement (e.g. 1, 3, 5); 0 if not a placement game
}

// SortedByStandings returns rosters sorted by wins descending, then points for descending.
func (rl Rosters) SortedByStandings() Rosters {
	sort.Slice(rl, func(i, j int) bool {
		if rl[i].Settings.Wins != rl[j].Settings.Wins {
			return rl[i].Settings.Wins > rl[j].Settings.Wins
		}
		return rl[i].GetPointsFor() > rl[j].GetPointsFor()
	})
	return rl
}
