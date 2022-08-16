package main

import (
	"github.com/opensourceways/robot-gitee-repo-watcher/models"
)

type localState struct {
	repos map[string]*models.Repo
}

func (r *localState) getOrNewRepo(repo string) *models.Repo {
	if v, ok := r.repos[repo]; ok {
		return v
	}

	v := models.NewRepo(repo, models.RepoState{})
	r.repos[repo] = v

	return v
}

func (r *localState) clear(isExpectedRepo func(string) bool) {
	for k := range r.repos {
		if !isExpectedRepo(k) {
			delete(r.repos, k)
		}
	}
}

func (bot *robot) loadALLRepos(org string) (*localState, error) {
	items, err := bot.cli.GetRepos(org)
	if err != nil {
		return nil, err
	}

	r := localState{
		repos: make(map[string]*models.Repo),
	}

	for i := range items {
		item := &items[i]

		colls, _ := bot.cli.ListCollaborators(org, item.Path)

		cols := make([]string, 0)
		for _, c := range colls {
			if c.Permissions.Admin == true {
				cols = append(cols, c.Login)
			}
		}

		r.repos[item.Path] = models.NewRepo(item.Path, models.RepoState{
			Available: true,
			Members:   toLowerOfMembers(item.Members),
			Admins:    toLowerOfMembers(cols),
			Property: models.RepoProperty{
				Private:    item.Private,
				CanComment: item.CanComment,
			},
			Owner: item.Owner.Login,
		})
	}

	return &r, nil
}
