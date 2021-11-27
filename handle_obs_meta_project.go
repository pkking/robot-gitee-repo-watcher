package main

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

func (bot *robot) createOBSMetaProject(repo string, log *logrus.Entry) {
	if !bot.cfg.EnableCreatingOBSMetaProject {
		return
	}

	project := &bot.cfg.OBSMetaProject
	path := project.genProjectFilePath(repo)
	b := &project.Branch

	// file exists
	if _, err := bot.cli.GetPathContent(b.Org, b.Repo, path, b.Branch); err == nil {
		return
	}

	content, err := project.genProjectFileContent(repo)
	if err != nil {
		log.Errorf("generate file of project:%s, err:%s", repo, err.Error())
		return
	}

	w := &bot.cfg.WatchingFiles
	msg := fmt.Sprintf(
		"add project according to the file: %s/%s/%s:%s",
		w.Org, w.Repo, w.Branch, w.RepoFilePath,
	)

	_, err = bot.cli.CreateFile(b.Org, b.Repo, b.Branch, path, content, msg)
	if err != nil {
		log.Errorf("ceate file: %s, err:%s", path, err.Error())
	}
}
