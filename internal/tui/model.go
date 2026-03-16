package tui

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sam-maryland/sleeper-cli/internal/config"
	"github.com/sam-maryland/sleeper-cli/pkg/sleeper"
)

type appState int

const (
	stateSelectYear appState = iota
	stateLoading
	stateStandings
	stateError
)

// yearItem implements list.Item for a single season entry.
type yearItem struct {
	year     string
	leagueID string
}

func (i yearItem) FilterValue() string { return i.year }
func (i yearItem) Title() string       { return i.year }
func (i yearItem) Description() string { return "League ID: " + i.leagueID }

// --- messages ---

type standingsFetchedMsg struct {
	standings []sleeper.Standing
	league    sleeper.SleeperLeague
}

type errMsg struct{ err error }

// --- model ---

type Model struct {
	cfg       config.Config
	state     appState
	list      list.Model
	spinner   spinner.Model
	standings []sleeper.Standing
	league    sleeper.SleeperLeague
	err       error
	width     int
	height    int
}

func New(cfg config.Config) Model {
	years := cfg.Years()
	items := make([]list.Item, len(years))
	for i, y := range years {
		items[i] = yearItem{year: y, leagueID: cfg.Leagues[y]}
	}

	l := list.New(items, list.NewDefaultDelegate(), 40, 14)
	l.Title = "Sleeper Standings"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	s := spinner.New()
	s.Spinner = spinner.Dot

	return Model{
		cfg:     cfg,
		state:   stateSelectYear,
		list:    l,
		spinner: s,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-2)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.state == stateStandings || m.state == stateError {
				m.state = stateSelectYear
				return m, nil
			}
		case "enter":
			if m.state == stateSelectYear {
				item, ok := m.list.SelectedItem().(yearItem)
				if !ok {
					return m, nil
				}
				m.state = stateLoading
				return m, tea.Batch(
					m.spinner.Tick,
					fetchStandings(item.leagueID),
				)
			}
		}

	case standingsFetchedMsg:
		m.standings = msg.standings
		m.league = msg.league
		m.state = stateStandings
		return m, nil

	case errMsg:
		m.err = msg.err
		m.state = stateError
		return m, nil

	case spinner.TickMsg:
		if m.state == stateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	if m.state == stateSelectYear {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	switch m.state {
	case stateSelectYear:
		return m.list.View()
	case stateLoading:
		return fmt.Sprintf("\n\n  %s  Fetching standings…\n", m.spinner.View())
	case stateStandings:
		return m.standingsView()
	case stateError:
		return fmt.Sprintf("\n\n  %s\n\n  %s\n",
			errorStyle.Render("Error: "+m.err.Error()),
			hintStyle.Render("esc to go back • q to quit"),
		)
	}
	return ""
}

func (m Model) standingsView() string {
	var sb strings.Builder

	isComplete := m.league.Status == "complete"
	playoffLine := m.league.Settings.PlayoffTeams

	title := fmt.Sprintf("🏆  %s Standings", m.league.Season)
	if isComplete {
		title = fmt.Sprintf("🏆  %s Final Standings", m.league.Season)
	}

	sb.WriteString("\n")
	sb.WriteString(titleStyle.Render(title))
	sb.WriteString("\n\n")

	// Column header
	header := fmt.Sprintf("%-4s  %-22s  %3s  %3s  %3s  %7s  %7s",
		"#", "Team", "W", "L", "T", "PF", "PA")
	sb.WriteString(headerStyle.Render(header))
	sb.WriteString("\n")
	sb.WriteString(dividerStyle.Render(strings.Repeat("─", 61)))
	sb.WriteString("\n")

	for i, s := range m.standings {
		// Playoff line separator (in-progress only)
		if !isComplete && playoffLine > 0 && i == playoffLine {
			sb.WriteString("\n")
			sb.WriteString(dividerStyle.Render("── Playoff Line " + strings.Repeat("─", 45)))
			sb.WriteString("\n\n")
		}

		rank := fmt.Sprintf("%d.", i+1)

		name := s.TeamName
		if len([]rune(name)) > 22 {
			name = string([]rune(name)[:20]) + ".."
		}

		row := fmt.Sprintf("%-4s  %-22s  %3d  %3d  %3d  %7.1f  %7.1f",
			rank, name, s.Wins, s.Losses, s.Ties, s.PointsFor, s.PointsAgainst)

		if !isComplete && playoffLine > 0 && i < playoffLine {
			sb.WriteString("  " + playoffRowStyle.Render(row))
		} else {
			sb.WriteString("  " + regularRowStyle.Render(row))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(hintStyle.Render("esc to go back • q to quit"))
	sb.WriteString("\n")

	return sb.String()
}

func fetchStandings(leagueID string) tea.Cmd {
	return func() tea.Msg {
		client := sleeper.NewSleeperClient(&http.Client{})
		standings, league, err := sleeper.CalculateStandings(context.Background(), client, leagueID)
		if err != nil {
			return errMsg{err}
		}
		return standingsFetchedMsg{standings: standings, league: league}
	}
}
