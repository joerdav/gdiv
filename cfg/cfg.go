package cfg

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const defaultPath = "~/.gdivpat"

type cmdArgs struct {
	pat, patPath          string
	org, head, base       string
	aheadOnly, behindOnly bool
	all, help, short      bool
}

type Config struct {
	GitPat                string
	Org, Head, Base       string
	AheadOnly, BehindOnly bool
	ShowAll, Short        bool
}

func LoadArgs() (cfg Config, err error) {
	var cmd cmdArgs

	flag.Usage = func() {
		fmt.Println("usage: gdiv [owner] [base] [head] [-pat | -pat-path] ")
		flag.PrintDefaults()
	}
	flag.StringVar(&cmd.patPath, "pat-path", "", "A file containing your github PAT.")
	flag.StringVar(&cmd.pat, "pat", "", "Your github PAT.")
	flag.BoolVar(&cmd.all, "a", false, "Show all repos including up to date branches and failed searches.")
	flag.BoolVar(&cmd.help, "h", false, "Show help text.")
	flag.BoolVar(&cmd.aheadOnly, "ahead", false, "Show only the aheadBy number.")
	flag.BoolVar(&cmd.behindOnly, "behind", false, "Show only the behindBy number.")
	flag.BoolVar(&cmd.short, "s", false, "Show a short version of the output.")

	flag.Parse()

	args := flag.Args()
	if cmd.pat == "" && cmd.patPath == "" {
		err = errors.New("Either pat or pat-path must be provided")
		flag.Usage()
		return
	}
	if cmd.help {
		flag.Usage()
		return
	}
	if cmd.aheadOnly && cmd.behindOnly {
		err = errors.New("You can only use -ahead and -behind exclusively.")
		return
	}

	cfg = Config{
		GitPat:     cmd.pat,
		Org:        args[0],
		Base:       args[1],
		Head:       args[2],
		ShowAll:    cmd.all,
		Short:      cmd.short,
		AheadOnly:  cmd.aheadOnly,
		BehindOnly: cmd.behindOnly,
	}

	if cfg.GitPat == "" {
		cfg.GitPat = loadFromPath(cmd.patPath)
	}

	return
}

func loadFromPath(p string) string {
	path := defaultPath
	if path != "" {
		path = p
	}
	usr, _ := user.Current()
	dir := usr.HomeDir
	if path == "~" {
		path = dir
	} else if strings.HasPrefix(path, "~/") {
		path = filepath.Join(dir, path[2:])
	}
	b, _ := os.ReadFile(path)
	return strings.TrimSpace(string(b))
}
