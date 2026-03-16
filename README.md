# sleeper-cli

A terminal UI and Go library for interacting with the [Sleeper](https://sleeper.com) fantasy football API.

## Features

- Interactive TUI built with [Charm](https://charm.land/) (Bubble Tea + Lip Gloss)
- View standings for any configured league season
- Full H2H tiebreaking — same logic as the Sleeper app
- Playoff-aware final standings for completed seasons — infers bracket results (quarterfinals, semifinals, championship, third-place game) directly from the API with no database required
- Importable as a Go library for use in other projects

## Installation

```bash
go install github.com/sam-maryland/sleeper-cli/cmd/sleeper@latest
```

Or clone and build locally:

```bash
git clone https://github.com/sam-maryland/sleeper-cli
cd sleeper-cli
go build -o sleeper ./cmd/sleeper
```

## Configuration

Create `~/.sleeper/config.yaml` with your league IDs mapped to season years:

```yaml
leagues:
  "2024": "your_league_id_here"
  "2023": "your_league_id_here"
```

Your league ID can be found in the Sleeper app URL: `sleeper.com/leagues/<league_id>`

A full example is available at [`config.example.yaml`](./config.example.yaml).

You can also point to a custom config file with the `--config` flag:

```bash
sleeper --config ~/my-league.yaml standings
```

## Usage

```bash
sleeper standings
```

This opens the TUI. Select a season year with the arrow keys and press `Enter` to load standings. Press `Esc` to go back to the year selector, `q` to quit.

### Standings view

- **In-progress seasons** — teams are sorted by record (W-L-T), with H2H and points for as tiebreakers. A playoff line separates the field.
- **Completed seasons** — final placements 1–6 are determined from playoff results. Remaining teams are sorted by regular season record.

## Using as a library

The `pkg/sleeper` package is designed to be imported independently:

```go
import "github.com/sam-maryland/sleeper-cli/pkg/sleeper"

client := sleeper.NewSleeperClient(&http.Client{})

// Fetch and compute full standings for a league
standings, league, err := sleeper.CalculateStandings(ctx, client, "your_league_id")

// Or use the client directly
rosters, err := client.GetRostersInLeague(ctx, "your_league_id")
matchups, err := client.GetMatchupsForWeek(ctx, "your_league_id", 7)
state, err := client.GetNFLState(ctx)
```

### Available client methods

| Method | Description |
|---|---|
| `GetUser(ctx, userID)` | Fetch a user by ID |
| `GetLeague(ctx, leagueID)` | Fetch league settings and metadata |
| `GetUsersInLeague(ctx, leagueID)` | Fetch all users in a league |
| `GetRostersInLeague(ctx, leagueID)` | Fetch all rosters with W/L/PF stats |
| `GetMatchupsForWeek(ctx, leagueID, week)` | Fetch matchup data for a given week |
| `GetNFLState(ctx)` | Fetch current NFL season/week state |
| `FetchAllPlayers(ctx)` | Fetch raw NFL player data |

### `CalculateStandings`

```go
func CalculateStandings(ctx context.Context, c ISleeperClient, leagueID string) ([]Standing, SleeperLeague, error)
```

Returns standings sorted by record with full tiebreaking. For completed leagues, applies playoff-based ranking for the top finishers. Requires no external storage — all data is fetched live from the Sleeper API.

## How standings are calculated

**Regular season / in-progress:**
1. Roster W/L/T and PF/PA are seeded from `GetRostersInLeague` (Sleeper maintains these)
2. All regular season matchups are fetched week-by-week to build H2H win records
3. Tiebreakers applied in order: H2H record within tied group → points for → points against

**Completed seasons:**
1. Playoff weeks are fetched and the bracket is inferred from league settings (`playoff_week_start`, `playoff_rounds`, `playoff_teams`)
2. Places 1–4 are determined from championship and third-place game results
3. Places 5–6 (quarterfinal losers) are sorted by points for
4. Places 7+ follow regular season standings
