package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v41/github"
	"github.com/joe-davidson1802/gdiv/cfg"
	"golang.org/x/oauth2"
)

func main() {
	cfg, err := cfg.LoadArgs()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	client := newGitClient(cfg.GitPat)
	repos, err := client.getRepos(cfg.Org)
	if err != nil {
		panic(err)
	}
	ch := make(chan string, len(repos))
	wd := writeDiff
	if cfg.Short {
		wd = writeShort
	}
	for _, r := range repos {
		go func(r string) {
			diff, err := client.getDiff(context.Background(), cfg.Org, r, cfg.Base, cfg.Head)
			if err != nil && !cfg.ShowAll {
				ch <- ""
				return
			}
			if cfg.AheadOnly && diff.Ahead == 0 {
				ch <- ""
				return
			}
			if cfg.BehindOnly && diff.Behind == 0 {
				ch <- ""
				return
			}
			if err != nil {
				ch <- writeError(r, err)
				return
			}
			ch <- wd(r, diff, cfg)
		}(r)
	}
	if cfg.OneLine {
		var b bytes.Buffer
		for range repos {
			var t = <-ch
			if t != "" {
				b.WriteString(t)
				b.WriteString("\\n")
			}
		}
		fmt.Print(b.String())
	} else {
		for range repos {
			fmt.Print(<-ch)
		}
	}
}

func writeDiff(repoName string, diff Diff, cfg cfg.Config) string {
	sb := new(strings.Builder)
	msg := []string{}
	if cfg.AheadOnly || (!cfg.AheadOnly && !cfg.BehindOnly) {
		msg = append(msg, fmt.Sprintf("ahead by %d", diff.Ahead))
	}
	if cfg.BehindOnly || (!cfg.AheadOnly && !cfg.BehindOnly) {
		msg = append(msg, fmt.Sprintf("behind by %d", diff.Behind))
	}
	sb.WriteString(writeRow(repoName, strings.Join(msg, ", "), cfg.OneLine))
	sb.WriteString(writeRow("", fmt.Sprintf("  %s -> %s", diff.BaseHash, diff.HeadHash), cfg.OneLine))
	sb.WriteString(writeRow("", "  "+diff.URL, cfg.OneLine))
	return sb.String()
}
func writeShort(repoName string, diff Diff, cfg cfg.Config) string {
	sb := new(strings.Builder)
	msg := []string{}
	if cfg.AheadOnly || (!cfg.AheadOnly && !cfg.BehindOnly) {
		msg = append(msg, fmt.Sprintf("ahead by %d", diff.Ahead))
	}
	if cfg.BehindOnly || (!cfg.AheadOnly && !cfg.BehindOnly) {
		msg = append(msg, fmt.Sprintf("behind by %d", diff.Behind))
	}
	msg = append(msg, diff.URL)

	sb.WriteString(writeRow(repoName, strings.Join(msg, ", "), cfg.OneLine))
	return sb.String()
}

func writeError(repoName string, err error) string {
	return fmt.Sprintln(repoName, strings.Repeat(" ", 45-len(repoName)), err.Error())
}

func writeRow(repoName, message string, oneline bool) string {
	if oneline {
		return fmt.Sprint(repoName, strings.Repeat(" ", 45-len(repoName)), message)
	}
	return fmt.Sprintln(repoName, strings.Repeat(" ", 45-len(repoName)), message)
}

type gitClient struct {
	client *github.Client
}

func newGitClient(token string) gitClient {
	if token == "" {
		client := github.NewClient(nil)
		return gitClient{client}
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)
	return gitClient{client}
}

func (cli gitClient) getRepos(org string) (names []string, err error) {
	rs, _, err := cli.client.Repositories.ListByOrg(context.Background(), org, &github.RepositoryListByOrgOptions{})
	if err != nil {
		return
	}
	for _, r := range rs {
		names = append(names, r.GetName())
	}
	return
}

type Diff struct {
	BaseHash string
	HeadHash string
	URL      string
	Ahead    int
	Behind   int
}

func (cli gitClient) getDiff(ctx context.Context, org, repo, base, head string) (diff Diff, err error) {
	m, _, err := cli.client.Repositories.GetBranch(ctx, org, repo, base, true)
	if err != nil {
		return
	}
	s, _, err := cli.client.Repositories.GetBranch(ctx, org, repo, head, true)
	if err != nil {
		return
	}
	r, _, err := cli.client.Repositories.CompareCommits(ctx, org, repo, s.GetName(), m.GetName(), nil)
	if err != nil {
		return
	}
	diff.BaseHash = m.GetCommit().GetSHA()
	diff.HeadHash = s.GetCommit().GetSHA()
	diff.URL = r.GetHTMLURL()
	diff.Ahead = r.GetAheadBy()
	diff.Behind = r.GetBehindBy()
	return
}
