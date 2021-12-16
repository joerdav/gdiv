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
	pat, patPath    string
	org, head, base string
	all, help       bool
}

type Config struct {
	GitPat          string
	Org, Head, Base string
	ShowAll         bool
}

func LoadArgs() (cfg Config, err error) {
	var cmd cmdArgs

	flag.Usage = func() {
		fmt.Println("usage: gdiv [owner] [base] [head]")
		flag.PrintDefaults()
	}
	flag.StringVar(&cmd.patPath, "pat-path", "", "A file containing your github PAT.")
	flag.StringVar(&cmd.pat, "pat", "", "Your github PAT.")
	flag.BoolVar(&cmd.all, "a", false, "Show all repos including failed searches.")
	flag.BoolVar(&cmd.help, "h", false, "Show help text.")
	flag.Parse()

	args := flag.Args()
	if len(args) != 3 {
		err = errors.New("missing arguments")
		flag.Usage()
		return
	}
	if cmd.help {
		flag.Usage()
		return
	}

	cfg = Config{
		GitPat:  cmd.pat,
		Org:     args[0],
		Base:    args[1],
		Head:    args[2],
		ShowAll: cmd.all,
	}

	if cfg.GitPat != "" {
		return
	}

	path := defaultPath
	if cmd.patPath != "" {
		path = cmd.patPath
	}

	usr, _ := user.Current()
	dir := usr.HomeDir
	if path == "~" {
		path = dir
	} else if strings.HasPrefix(path, "~/") {
		path = filepath.Join(dir, path[2:])
	}
	b, _ := os.ReadFile(path)
	cfg.GitPat = strings.TrimSpace(string(b))

	if cfg.GitPat == "" {
		err = errors.New("pat not found")
		return
	}

	return
}
