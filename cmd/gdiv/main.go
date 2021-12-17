package main

import (
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
	for _, r := range repos {
		go func(r string) {
			a, b, err := client.getDiff(context.Background(), cfg.Org, r, cfg.Base, cfg.Head)
			if err == nil && (a+b > 0 || cfg.ShowAll) {
				ch <- writeRow(r, fmt.Sprintf("ahead by %d, behind by %d", a, b))
				return
			}
			if cfg.ShowAll {
				ch <- writeRow(r, err.Error())
				return
			}
			ch <- ""
		}(r)
	}
	for range repos {
		fmt.Print(<-ch)
	}
}

func writeRow(reponame, message string) string {
	return fmt.Sprintln(reponame, strings.Repeat(" ", 45-len(reponame)), message)
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

func (cli gitClient) getDiff(ctx context.Context, org, repo, base, head string) (ahead int, behind int, err error) {
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
	ahead = r.GetAheadBy()
	behind = r.GetBehindBy()
	return
}
