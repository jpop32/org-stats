package orgstats

import (
	"context"
	"github.com/caarlos0/spin"
	"strings"
	"time"

	"github.com/google/go-github/v28/github"
)

// Stat represents an user adds, rms and commits count
type Stat struct {
	Additions, Deletions, Commits int
}

// Stats contains the user->Stat mapping
type Stats map[string]Stat

// NewStats return a new Stats map
func NewStats() Stats {
	return make(map[string]Stat)
}

// Gather a given organization's stats
func Gather(token, org string, blacklist []string, url string, year int) (Stats, Stats, Stat, int, error) {
	var ctx = context.Background()
	var contribStats = NewStats()
	var weeklyStats = NewStats()
	totalStats := Stat{}
	client, err := newClient(ctx, token, url)
	if err != nil {
		return contribStats, weeklyStats, totalStats, 0, err
	}

	allRepos, err := repos(ctx, client, org)
	if err != nil {
		return contribStats, weeklyStats, totalStats, 0, err
	}

	for _, repo := range allRepos {
		if isBlacklisted(blacklist, repo.GetName()) {
			continue
		}
		var spinner = spin.New("  \033[36m%s Gathering data for '" + repo.GetName() + "'...\033[m")
		spinner.Start()
		statsContrib, serr := getContributorStats(ctx, client, org, *repo.Name)
		if serr != nil {
			return contribStats, weeklyStats, totalStats, 0, serr
		}
		for _, cs := range statsContrib {
			if isBlacklisted(blacklist, cs.Author.GetLogin()) {
				continue
			}
			contribStats.addContrib(cs, year, &totalStats)
		}
		statsWeekly, serr := getWeeklyStats(ctx, client, org, *repo.Name)
		if serr != nil {
			return contribStats, weeklyStats, totalStats, 0, serr
		}
		for _, cs := range statsWeekly {
			totalStats.Commits += weeklyStats.addWeekly(*repo.Name, cs, year)
		}
		spinner.Stop()
	}
	return contribStats, weeklyStats, totalStats, len(allRepos), err
}

func isBlacklisted(blacklist []string, s string) bool {
	for _, b := range blacklist {
		if strings.EqualFold(s, b) {
			return true
		}
	}
	return false
}

func (s Stats) addContrib(cs *github.ContributorStats, year int, totalStats *Stat) {
	if cs.Author == nil {
		return
	}
	stat := s[*cs.Author.Login]
	var adds int
	var rms int
	var commits int
	for _, week := range cs.Weeks {
		if year != -1 && week.Week.Year() != year {
			continue
		}
		adds += *week.Additions
		totalStats.Additions += *week.Additions
		rms += *week.Deletions
		totalStats.Deletions += *week.Deletions
		commits += *week.Commits
	}
	stat.Additions += adds
	stat.Deletions += rms
	stat.Commits += commits
	s[*cs.Author.Login] = stat
}

func (s Stats) addWeekly(repo string, cs *github.WeeklyCommitActivity, year int) int {
	if year != -1 && cs.Week.Year() != year {
		return 0
	}
	stat := s[repo]
	stat.Commits += *cs.Total
	s[repo] = stat
	return *cs.Total
}

func repos(ctx context.Context, client *github.Client, org string) ([]*github.Repository, error) {
	var opt = &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 10},
	}
	var allRepos []*github.Repository
	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, org, opt)
		if err != nil {
			return allRepos, err
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.ListOptions.Page = resp.NextPage
	}
	return allRepos, nil
}

func getContributorStats(ctx context.Context, client *github.Client, org, repo string) ([]*github.ContributorStats, error) {
	stats, _, err := client.Repositories.ListContributorsStats(ctx, org, repo)
	if err != nil {
		if _, ok := err.(*github.RateLimitError); ok {
			time.Sleep(time.Duration(15) * time.Second)
			return getContributorStats(ctx, client, org, repo)
		}
		if _, ok := err.(*github.AcceptedError); ok {
			return getContributorStats(ctx, client, org, repo)
		}
	}
	return stats, err
}

func getWeeklyStats(ctx context.Context, client *github.Client, org, repo string) ([]*github.WeeklyCommitActivity, error) {

	stats, _, err := client.Repositories.ListCommitActivity(ctx, org, repo)
	if err != nil {
		if _, ok := err.(*github.RateLimitError); ok {
			time.Sleep(time.Duration(15) * time.Second)
			return getWeeklyStats(ctx, client, org, repo)
		}
		if _, ok := err.(*github.AcceptedError); ok {
			return getWeeklyStats(ctx, client, org, repo)
		}
	}
	return stats, err
}
