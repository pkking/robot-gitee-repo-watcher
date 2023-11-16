package main

import (
	"testing"

	"github.com/opensourceways/robot-gitee-repo-watcher/community"
)

func TestCanCreateRepoNoRepoUrl(t *testing.T) {
	expect := expectRepoInfo{
		org: "src-openeuler",
		expectRepoState: &community.Repository{
			Name: "i3",
		},
	}
	if !CanProcess(expect) {
		t.Fail()
	}
}

func TestCanCreateRepoWithRepoUrl(t *testing.T) {

	expect := expectRepoInfo{
		org: "src-openeuler",
		expectRepoState: &community.Repository{
			Name:    "i3",
			RepoUrl: "https://gitee.com/src-openeuler/i3",
		},
	}
	if !CanProcess(expect) {
		t.Fail()
	}
}

func TestCanCreateRepoWithGithubRepoUrl(t *testing.T) {
	expect := expectRepoInfo{
		org: "src-openeuler",
		expectRepoState: &community.Repository{
			Name:    "i3",
			RepoUrl: "https://github.com/src-openeuler/i3",
		},
	}
	if CanProcess(expect) {
		t.Fail()
	}
}

func TestCanCreateRepoWithInvalidRepoUrl(t *testing.T) {
	expect := expectRepoInfo{
		org: "src-openeuler",
		expectRepoState: &community.Repository{
			Name:    "i3",
			RepoUrl: "src-openeuler/i3",
		},
	}
	if CanProcess(expect) {
		t.Fail()
	}
}

func TestCanCreateRepoWithValidRepo(t *testing.T) {

	expect := expectRepoInfo{
		org: "src-openeuler",
		expectRepoState: &community.Repository{
			Name:    "i3",
			RepoUrl: "gitee.com/openeuler/i3",
		},
	}
	if CanProcess(expect) {
		t.Fail()
	}
}
