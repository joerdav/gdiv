package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
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
	ds := make(chan Diff, len(repos))
	wd := writeDiff
	if cfg.Short {
		wd = writeShort
	}
	for _, r := range repos {
		go func(r string) {
			diff, err := client.getDiff(context.Background(), cfg.Org, r, cfg.Base, cfg.Head)
			if err != nil && !cfg.ShowAll {
				ch <- ""
				ds <- Diff{}
				return
			}
			if err != nil {
				ch <- writeError(r, err)
				ds <- Diff{}
				return
			}
			if cfg.ShowAll {
				ch <- wd(r, diff, cfg)
				ds <- diff
				return
			}
			if diff.Behind == 0 && diff.Ahead == 0 {
				ch <- ""
				ds <- Diff{}
				return
			}
			if cfg.AheadOnly && diff.Ahead == 0 {
				ch <- ""
				ds <- Diff{}
				return
			}
			if cfg.BehindOnly && diff.Behind == 0 {
				ch <- ""
				ds <- Diff{}
				return
			}
			ch <- wd(r, diff, cfg)
			ds <- diff
		}(r)
	}
	if cfg.Json {
		var diffs []Diff
		for range repos {
			d := <-ds
			if d.Name == "" {
				continue
			}
			diffs = append(diffs, d)
		}
		b := new(bytes.Buffer)
		json.NewEncoder(b).Encode(diffs)
		fmt.Print(b.String())
		return
	}
	for range repos {
		fmt.Print(<-ch)
	}
}

func writeDiff(repoName string, diff Diff, cfg cfg.Config) string {
	sb := new(strings.Builder)
	msg := []string{}
	if cfg.AheadOnly || !cfg.BehindOnly {
		msg = append(msg, fmt.Sprintf("ahead by %d", diff.Ahead))
	}
	if !cfg.AheadOnly || cfg.BehindOnly {
		msg = append(msg, fmt.Sprintf("behind by %d", diff.Behind))
	}
	sb.WriteString(writeRow(repoName, strings.Join(msg, ", ")))
	sb.WriteString(writeRow("", fmt.Sprintf("  %s -> %s", diff.BaseHash, diff.HeadHash)))
	sb.WriteString(writeRow("", "  "+diff.URL))
	return sb.String()
}
func writeShort(repoName string, diff Diff, cfg cfg.Config) string {
	sb := new(strings.Builder)
	msg := []string{}
	if cfg.AheadOnly || !cfg.BehindOnly {
		msg = append(msg, fmt.Sprintf("ahead by %d", diff.Ahead))
	}
	if !cfg.AheadOnly || cfg.BehindOnly {
		msg = append(msg, fmt.Sprintf("behind by %d", diff.Behind))
	}
	msg = append(msg, diff.URL)
	sb.WriteString(writeRow(repoName, strings.Join(msg, ", ")))
	return sb.String()
}

func writeError(repoName string, err error) string {
	return fmt.Sprintln(repoName, strings.Repeat(" ", 45-len(repoName)), err.Error())
}

func writeRow(repoName, message string) string {
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

func (cli gitClient) getRepos(org string) ([]string, error) {
	var names []string
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	// get all pages of results
	for {
		repos, resp, err := cli.client.Repositories.ListByOrg(context.Background(), org, opt)
		if err != nil {
			return names, err
		}
		for _, r := range repos {
			names = append(names, r.GetName())
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return names, nil
}

type Diff struct {
	Name     string `json:"name"`
	BaseHash string `json:"baseHash"`
	HeadHash string `json:"headHash"`
	URL      string `json:"url"`
	Ahead    int    `json:"ahead"`
	Behind   int    `json:"behind"`
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
	diff.Name = repo
	return
}
