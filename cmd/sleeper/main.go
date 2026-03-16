package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sam-maryland/sleeper-cli/internal/config"
	"github.com/sam-maryland/sleeper-cli/internal/tui"
	"github.com/sam-maryland/sleeper-cli/pkg/sleeper"
	"github.com/spf13/cobra"
)

var cfgFile string

func main() {
	rootCmd := &cobra.Command{
		Use:   "sleeper",
		Short: "Sleeper fantasy football CLI",
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.sleeper/config.yaml)")

	rootCmd.AddCommand(standingsCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func standingsCmd() *cobra.Command {
	var year string
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "standings",
		Short: "View standings for a league season",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			if len(cfg.Leagues) == 0 {
				return fmt.Errorf("no leagues configured — add entries to ~/.sleeper/config.yaml")
			}

			if !asJSON {
				p := tea.NewProgram(tui.New(cfg), tea.WithAltScreen())
				_, err = p.Run()
				return err
			}

			// Resolve year: explicit flag > most recent configured year
			if year == "" {
				years := cfg.Years()
				year = years[0]
			}
			leagueID, ok := cfg.Leagues[year]
			if !ok {
				return fmt.Errorf("no league configured for year %q", year)
			}

			client := sleeper.NewSleeperClient(&http.Client{})
			standings, league, err := sleeper.CalculateStandings(context.Background(), client, leagueID)
			if err != nil {
				return fmt.Errorf("calculating standings: %w", err)
			}

			out := struct {
				League    sleeper.SleeperLeague `json:"league"`
				Standings []sleeper.Standing    `json:"standings"`
			}{
				League:    league,
				Standings: standings,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}

	cmd.Flags().StringVar(&year, "year", "", "season year (default: most recent configured year)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output standings as JSON instead of launching the TUI")

	return cmd
}
