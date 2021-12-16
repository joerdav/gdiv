package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v41/github"
	"golang.org/x/oauth2"
)

func main() {
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	var token string
	pp := flag.String("pat-path", "", "A file containing your github PAT.")
	p := flag.String("pat", "", "Your github PAT.")
	all := flag.Bool("a", false, "Show all repos including failed searches.")
	b, err := os.ReadFile(path.Join(dirname, ".gdivpat"))
	if err == nil {
		token = string(b)
	}
	flag.Parse()
	if *pp != "" {
		usr, _ := user.Current()
		dir := usr.HomeDir
		path := *pp
		if path == "~" {
			path = dir
		} else if strings.HasPrefix(path, "~/") {
			path = filepath.Join(dir, path[2:])
		}
		b, err := os.ReadFile(*pp)
		if err != nil {
			panic(err)
		}
		token = string(b)
	}
	if *p != "" {
		token = *p
	}
	if len(flag.Args()) != 3 {
		panic("usage: gdiv [owner] [base] [head]")
	}
	name := flag.Arg(0)
	base := flag.Arg(1)
	head := flag.Arg(2)
	client := newGitClient(token)
	repos, err := client.getRepos(name)
	if err != nil {
		panic(err)
	}
	ch := make(chan string, len(repos))
	for _, r := range repos {
		go func(r string) {
			a, b, err := client.getDiff(context.Background(), name, r, base, head)
			if err == nil {
				ch <- writeRow(r, fmt.Sprintf("ahead by %d, behind by %d", a, b))
				return
			}
			if *all {
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
		&oauth2.Token{AccessToken: strings.TrimSpace(token)},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)
	return gitClient{client}
}

func (cli gitClient) getRepos(org string) (names []string, err error) {
	rs, _, err := cli.client.Repositories.ListByOrg(context.Background(), "aviva-verde", &github.RepositoryListByOrgOptions{})
	if err != nil {
		return
	}
	for _, r := range rs {
		names = append(names, r.GetName())
	}
	return
}

func (cli gitClient) getDiff(ctx context.Context, org, repo, base, head string) (ahead int, behind int, err error) {
	notFound := ""
	m, _, err := cli.client.Repositories.GetBranch(ctx, org, repo, base, false)
	if err != nil {
		notFound += fmt.Sprintf("no branch %s ", base)
	}
	s, _, err := cli.client.Repositories.GetBranch(ctx, org, repo, head, false)
	if err != nil {
		notFound += fmt.Sprintf("no branch %s ", head)
	}
	if notFound != "" {
		err = errors.New(notFound)
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
