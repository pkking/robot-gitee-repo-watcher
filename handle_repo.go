package main

import (
	"fmt"
	"strconv"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/sirupsen/logrus"

	"github.com/opensourceways/robot-gitee-repo-watcher/community"
	"github.com/opensourceways/robot-gitee-repo-watcher/models"
)

func (bot *robot) createRepo(
	expectRepo expectRepoInfo,
	log *logrus.Entry,
	hook func(string, *logrus.Entry),
) models.RepoState {
	org := expectRepo.org
	repo := expectRepo.expectRepoState
	repoName := expectRepo.getNewRepoName()

	if n := repo.RenameFrom; n != "" && n != repoName {
		return bot.renameRepo(org, repoName, n, log, hook)
	}

	log = log.WithField("create repo", repoName)
	log.Info("start")

	err := bot.cli.CreateRepo(org, sdk.RepositoryPostParam{
		Name:        repoName,
		Description: repo.Description,
		HasIssues:   true,
		HasWiki:     true,
		AutoInit:    true, // set `auto_init` as true to initialize `master` branch with README after repo creation
		CanComment:  repo.Commentable,
		Private:     repo.IsPrivate(),
	})
	if err != nil {
		if s, b := bot.getRepoState(org, repoName, log); b {
			return s
		}

		log.Errorf("create repo, err:%s", err.Error())

		return models.RepoState{}
	}

	defer func() {
		hook(repoName, log)
	}()

	if err := bot.initRepoReviewer(org, repoName); err != nil {
		log.Errorf("initialize the reviewers, err:%s", err.Error())
	}

	branches := []community.RepoBranch{}
	for _, item := range repo.Branches {
		if item.Name == "master" {
			continue
		}

		if b, ok := bot.createBranch(org, repoName, item, log); ok {
			branches = append(branches, b)
		}
	}

	members := []string{}
	for _, item := range expectRepo.expectOwners {
		if err := bot.addRepoMember(org, repoName, item); err != nil {
			log.Errorf("add member:%s, err:%s", item, err)
		} else {
			members = append(members, item)
		}
	}

	return models.RepoState{
		Available: true,
		Branches:  branches,
		Members:   members,
		Property: models.RepoProperty{
			CanComment: repo.Commentable,
			Private:    repo.IsPrivate(),
		},
	}
}

func (bot *robot) renameRepo(
	org, newRepo, oldRepo string,
	log *logrus.Entry,
	hook func(string, *logrus.Entry),
) models.RepoState {
	log = log.WithField("rename repo", fmt.Sprintf("from %s to %s", oldRepo, newRepo))
	log.Info("start")

	err := bot.cli.UpdateRepo(
		org,
		oldRepo,
		sdk.RepoPatchParam{
			Name: newRepo,
			Path: newRepo,
		},
	)

	defer func(b bool) {
		if b {
			hook(newRepo, log)
		}
	}(err == nil)

	// if the err == nil, invoke 'getRepoState' obviously.
	// if the err != nil, it is better to call 'getRepoState' to
	// avoid the case that the repo already exists.
	if s, b := bot.getRepoState(org, newRepo, log); b {
		return s
	}

	if err != nil {
		log.Error(err)

		return models.RepoState{}
	}

	return models.RepoState{Available: true}
}

func (bot *robot) getRepoState(org, repo string, log *logrus.Entry) (models.RepoState, bool) {
	newRepo, err := bot.cli.GetRepo(org, repo)
	if err != nil {
		log.Errorf("get repo, err:%s", err.Error())

		return models.RepoState{}, false
	}

	r := models.RepoState{
		Available: true,
		Members:   newRepo.Members,
		Property: models.RepoProperty{
			Private:    newRepo.Private,
			CanComment: newRepo.CanComment,
		},
	}

	branches, err := bot.listAllBranchOfRepo(org, repo)
	if err != nil {
		log.Errorf("list branch, err:%s", err.Error())
	} else {
		r.Branches = branches
	}

	return r, true
}

func (bot *robot) initRepoReviewer(org, repo string) error {
	return bot.cli.SetRepoReviewer(
		org,
		repo,
		sdk.SetRepoReviewer{
			Assignees:       " ", // This parameter is a required one according to the Gitee API
			Testers:         " ", // Ditto
			AssigneesNumber: 0,
			TestersNumber:   0,
		},
	)
}

func (bot *robot) updateRepo(expectRepo expectRepoInfo, lp models.RepoProperty, log *logrus.Entry) models.RepoProperty {
	org := expectRepo.org
	repo := expectRepo.expectRepoState
	repoName := expectRepo.getNewRepoName()

	ep := repo.IsPrivate()
	ec := repo.Commentable
	if ep != lp.Private || ec != lp.CanComment {
		log = log.WithField("update repo", repoName)
		log.Info("start")

		err := bot.cli.UpdateRepo(
			org,
			repoName,
			sdk.RepoPatchParam{
				Name:       repoName,
				CanComment: strconv.FormatBool(ec),
				Private:    strconv.FormatBool(ep),
			},
		)
		if err == nil {
			return models.RepoProperty{
				Private:    ep,
				CanComment: ec,
			}
		}

		log.WithFields(logrus.Fields{
			"Private":    ep,
			"CanComment": ec,
		}).Error(err)
	}
	return lp
}
