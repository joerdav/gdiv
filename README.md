# gdiv

Quickly find divergences between branches.

## Use case:

You are looking after a large number of git repos, you are using some kind of branching strategy e.g. main->staging->production.

If you want to find the repos where staging is behind run:

```
gdiv my-org main staging
```

This will produce a list of repos:

```
my-api                                         ahead by 0, behind by 0
my-front-end                                   ahead by 22, behind by 0
other-api                                      ahead by 1, behind by 0
```

## Installation 

Install the package:

```
go install github.com/joe-davidson1802/gdiv/cmd/gdiv@latest
```

(Optional) Configure your GitHub Personal Access Token.

- Generate a new PAT <https://github.com/settings/tokens>
- Save the new pat in `~/.gdivpat`
