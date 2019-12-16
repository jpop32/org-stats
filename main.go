package main

import "C"
import (
	"fmt"
	"os"

	"github.com/caarlos0/org-stats/orgstats"
	"github.com/urfave/cli"
)

var version = "master"

func main() {
	app := cli.NewApp()
	app.Name = "org-stats"
	app.Version = version
	app.Authors = []cli.Author{{Name: "Carlos Alexandro Becker", Email: "(caarlos0@gmail.com)"}, {Name: "Jovica Popović", Email: "(jpop32@gmail.com)"}}
	app.Usage = "Get the contributor and repo stats summary from all repos of any given organization"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			EnvVar: "GITHUB_TOKEN",
			Name:   "token",
			Usage:  "Your GitHub token",
		},
		cli.StringFlag{
			Name:  "org, o",
			Usage: "GitHub organization to scan",
		},
		cli.StringSliceFlag{
			Name:  "blacklist, b",
			Usage: "Blacklist repos and/or users",
		},
		cli.IntFlag{
			Name:  "top",
			Usage: "How many users/repos to show",
			Value: 3,
		},
		cli.StringFlag{
			Name:  "github-url",
			Usage: "Custom GitHub URL (for GitHub Enterprise for example)",
		},
		cli.IntFlag{
			Name:  "year",
			Usage: "Limit the stats to this year only",
			Value: -1,
		},
	}
	app.Action = func(c *cli.Context) error {
		var token = c.String("token")
		var org = c.String("org")
		var blacklist = c.StringSlice("blacklist")
		var top = c.Int("top")
		if token == "" {
			return cli.NewExitError("missing github api token", 1)
		}
		if org == "" {
			return cli.NewExitError("missing organization name", 1)
		}
		year := c.Int("year")
		if year != -1 {
			fmt.Printf("Data gathering started for top %d for year %d.\n", top, year)
		} else {
			fmt.Printf("Data gathering started for top %d.\n", top)
		}
		contribStats, weeklyStats, totalStats, nrRepos, err := orgstats.Gather(token, org, blacklist, c.String("github-url"), year)
		fmt.Println("Done!")
		fmt.Println("")
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		printHighlights(contribStats, weeklyStats, totalStats, nrRepos, top)
		return nil
	}
	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

func printHighlights(s orgstats.Stats, w orgstats.Stats, totalStats orgstats.Stat, nrRepos int, top int) {
	data := []struct {
		stats  []orgstats.StatPair
		trophy string
		kind   string
	}{
		{
			stats:  orgstats.Sort(s, orgstats.ExtractCommits),
			trophy: "Commit",
			kind:   "commits",
		}, {
			stats:  orgstats.Sort(s, orgstats.ExtractAdditions),
			trophy: "Lines Added",
			kind:   "lines added",
		}, {
			stats:  orgstats.Sort(s, orgstats.ExtractDeletions),
			trophy: "Housekeeper",
			kind:   "lines removed",
		},
	}
	for _, d := range data {
		fmt.Printf("\033[1m%s champions are:\033[0m\n", d.trophy)
		var j = top
		if len(d.stats) < j {
			j = len(d.stats)
		}
		for i := 0; i < j; i++ {
			fmt.Printf(
				"%s %s with %d %s!\n",
				emojiForPos(i),
				d.stats[i].Key,
				d.stats[i].Value,
				d.kind,
			)
		}
		fmt.Printf("\n")
	}

	fmt.Printf("\033[1mTop %d repos by number of commits:\033[0m\n", top)
	topRepos := orgstats.Sort(w, orgstats.ExtractCommits)
	for i := 0; i < top; i++ {
		w := topRepos[i]
		fmt.Printf("%s commits, %d\n", w.Key, w.Value)
	}
	fmt.Printf("\n")

	fmt.Printf("\033[1mTotals across all %d repos:\033[0m\n", nrRepos)
	fmt.Printf("Commits: %d\n", totalStats.Commits)
	fmt.Printf("Additions: %d\n", totalStats.Additions)
	fmt.Printf("Deletions: %d\n", totalStats.Deletions)
}

func emojiForPos(pos int) string {
	var emojis = []string{"\U0001f3c6", "\U0001f948", "\U0001f949"}
	if pos < len(emojis) {
		return emojis[pos]
	}
	return fmt.Sprint(pos + 1)
}
