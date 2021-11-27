package main

import (
	"sync"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/panjf2000/ants/v2"
)

const botName = "repo-watcher"

type iClient interface {
	GetRef(org, repo, ref string) (string, error)
	GetRepo(org, repo string) (sdk.Project, error)
	GetRepos(org string) ([]sdk.Project, error)
	CreateRepo(org string, repo sdk.RepositoryPostParam) error
	UpdateRepo(org, repo string, info sdk.RepoPatchParam) error
	SetRepoReviewer(org, repo string, reviewer sdk.SetRepoReviewer) error

	GetPathContent(org, repo, path, ref string) (sdk.Content, error)
	CreateFile(org, repo, branch, path, content, commitMsg string) (sdk.CommitContent, error)
	GetDirectoryTree(org, repo, sha string, recursive int32) (sdk.Tree, error)

	RemoveRepoMember(org, repo, login string) error
	AddRepoMember(org, repo, login, permission string) error

	GetRepoAllBranch(org, repo string) ([]sdk.Branch, error)
	CreateBranch(org, repo, branch, parentBranch string) error
	SetProtectionBranch(org, repo, branch string) error
	CancelProtectionBranch(org, repo, branch string) error
}

func newRobot(cli iClient, pool *ants.Pool, cfg *botConfig) *robot {
	return &robot{cli: cli, pool: pool, cfg: cfg}
}

type robot struct {
	pool *ants.Pool
	cfg  *botConfig
	cli  iClient
	wg   sync.WaitGroup
}
