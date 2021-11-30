package main

import (
	"encoding/base64"
	"fmt"
	"path"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/yaml"

	"github.com/opensourceways/robot-gitee-repo-watcher/community"
)

type watchingFileObject interface {
	Validate() error
}

type watchingFile struct {
	log      *logrus.Entry
	loadFile func(string) (string, string, error)

	file string
	sha  string
	obj  watchingFileObject
}

type getSHAFunc func(string) string

func (w *watchingFile) update(f getSHAFunc, newObject func() watchingFileObject) {
	if sha := f(w.file); sha == "" || sha == w.sha {
		return
	}

	c, sha, err := w.loadFile(w.file)
	if err != nil {
		w.log.Errorf("load file:%s, err:%s", w.file, err.Error())
		return
	}

	v := newObject()

	if err := decodeYamlFile(c, v); err != nil {
		w.log.Errorf("decode file:%s, err:%s", w.file, err.Error())
		return
	}

	if err := v.Validate(); err != nil {
		w.log.Errorf("validate the data of file:%s, err:%s", w.file, err.Error())
	} else {
		w.obj = v
		w.sha = sha
	}
}

type expectRepos struct {
	wf watchingFile
}

func (e *expectRepos) refresh(f getSHAFunc) *community.Repos {
	e.wf.update(f, func() watchingFileObject {
		return new(community.Repos)
	})

	if v, ok := e.wf.obj.(*community.Repos); ok {
		return v
	}
	return nil
}

type orgSigs struct {
	wf watchingFile
}

func (s *orgSigs) refresh(f getSHAFunc) *community.Sigs {
	s.wf.update(f, func() watchingFileObject {
		return new(community.Sigs)
	})

	if v, ok := s.wf.obj.(*community.Sigs); ok {
		return v
	}
	return nil
}

type expectSigOwners struct {
	wf watchingFile
}

func (e *expectSigOwners) refresh(f getSHAFunc) *community.RepoOwners {
	e.wf.update(f, func() watchingFileObject {
		return new(community.RepoOwners)
	})

	if v, ok := e.wf.obj.(*community.RepoOwners); ok {
		return v
	}
	return nil
}

type expectState struct {
	log *logrus.Entry
	cli iClient

	w         repoBranch
	sig       orgSigs
	repos     expectRepos
	sigDir    string
	sigOwners map[string]*expectSigOwners
}

func (e *expectState) init(repoFilePath, sigFilePath, sigDir string) (string, error) {
	e.repos = expectRepos{e.newWatchingFile(repoFilePath)}

	v := e.repos.refresh(func(string) string {
		return "init"
	})

	org := v.GetCommunity()
	if org == "" {
		return "", fmt.Errorf("load repository failed")
	}

	e.sig = orgSigs{e.newWatchingFile(sigFilePath)}
	e.sigDir = sigDir

	return org, nil
}

func (e *expectState) check(
	isStopped func() bool,
	clearLocal func(func(string) bool),
	checkRepo func(*community.Repository, []string, *logrus.Entry),
) {
	allFiles, err := e.listAllFilesOfRepo()
	if err != nil {
		e.log.Errorf("list all file, err:%s", err.Error())

		allFiles = make(map[string]string)
	}

	getSHA := func(p string) string {
		return allFiles[p]
	}

	allRepos := e.repos.refresh(getSHA)
	repoMap := allRepos.GetRepos()

	if len(repoMap) == 0 {
		// keep safe to do this. it is impossible to happen generally.
		e.log.Warning("there are not repos. Impossible!!!")
		return
	}

	clearLocal(func(r string) bool {
		_, ok := repoMap[r]
		return ok
	})

	done := sets.NewString()
	allSigs := e.sig.refresh(getSHA)
	sigs := allSigs.GetSigs()
	for i := range sigs {
		sig := &sigs[i]

		sigOwner := e.getSigOwner(sig.Name)
		owners := sigOwner.refresh(getSHA)

		for _, repoName := range sig.GetRepos() {
			if isStopped() {
				break
			}

			checkRepo(repoMap[repoName], owners.GetOwners(), e.log)

			done.Insert(repoName)
		}

		if isStopped() {
			break
		}
	}

	if len(repoMap) == done.Len() {
		return
	}

	for k, repo := range repoMap {
		if isStopped() {
			break
		}

		if !done.Has(k) {
			checkRepo(repo, nil, e.log)
		}
	}
}

func (e *expectState) getSigOwner(sigName string) *expectSigOwners {
	o, ok := e.sigOwners[sigName]
	if !ok {
		o = &expectSigOwners{
			wf: e.newWatchingFile(
				path.Join(e.sigDir, sigName, "OWNERS"),
			),
		}

		e.sigOwners[sigName] = o
	}

	return o
}

func (e *expectState) newWatchingFile(p string) watchingFile {
	return watchingFile{
		file:     p,
		log:      e.log,
		loadFile: e.loadFile,
	}
}

func (e *expectState) listAllFilesOfRepo() (map[string]string, error) {
	trees, err := e.cli.GetDirectoryTree(e.w.Org, e.w.Repo, e.w.Branch, 1)
	if err != nil || len(trees.Tree) == 0 {
		return nil, err
	}

	r := make(map[string]string)
	for i := range trees.Tree {
		item := &trees.Tree[i]
		r[item.Path] = item.Sha
	}

	return r, nil
}

func (e *expectState) loadFile(f string) (string, string, error) {
	c, err := e.cli.GetPathContent(e.w.Org, e.w.Repo, f, e.w.Branch)
	if err != nil {
		return "", "", err
	}

	return c.Content, c.Sha, nil
}

func decodeYamlFile(content string, v interface{}) error {
	c, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(c, v)
}
