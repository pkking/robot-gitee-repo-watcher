package main

import (
	"strconv"
	"testing"

	"github.com/opensourceways/robot-gitee-repo-watcher/community"
)

func TestCanProcess(t *testing.T) {
	testCase := [][]string{
		{"", "", "true"},
		{"xxx", "", "false"},
		{"xx", "github", "false"},
		{"", "gitee", "true"},
		{"", "github", "false"},
	}

	for k, v := range testCase {
		expect := expectRepoInfo{
			org: "src-openeuler",
			expectRepoState: &community.Repository{
				Name:     "i3",
				RepoUrl:  v[0],
				Platform: v[1],
			},
		}
		if strconv.FormatBool(CanProcess(expect)) != v[2] {
			t.Errorf("case num %d failed", k)
		}
	}
}
