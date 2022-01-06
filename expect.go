package main

import (
	"encoding/base64"
	"fmt"
	"path"
	"strings"

	sdk "gitee.com/openeuler/go-gitee/gitee"
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

func (e *expectRepos) refresh(f getSHAFunc) *community.Repository {
	e.wf.update(f, func() watchingFileObject {
		return new(community.Repository)
	})

	if v, ok := e.wf.obj.(*community.Repository); ok {
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
	log    *logrus.Entry
	cli    iClient
	w      repoBranch
	sigDir string

	tree      []sdk.TreeBasic
	reposInfo *community.Repos
	repos     map[string]*expectRepos
	sigOwners map[string]*expectSigOwners
}

func (e *expectState) init(orgPath, sigFilePath, sigDir string) (string, error) {
	trees, err := e.cli.GetDirectoryTree(e.w.Org, e.w.Repo, e.w.Branch, 1)
	if err != nil || len(trees.Tree) == 0 {
		return "", err
	}
	e.tree = trees.Tree

	reposInfo := new(community.Repos)
	e.repos = make(map[string]*expectRepos)
	for _, v := range e.tree {
		patharr := strings.Split(v.Path, "/")
		if patharr[0] != "sig" || len(patharr) != 5 || patharr[2] != orgPath {
			continue
		}

		exRepo := &expectRepos{e.newWatchingFile(v.Path)}
		e.repos[v.Path] = exRepo
		singleRepo := exRepo.refresh(func(string) string {
			return "init"
		})
		reposInfo.Repositories = append(reposInfo.Repositories, *singleRepo)
	}
	reposInfo.Validate()
	e.reposInfo = reposInfo

	org := orgPath
	if org == "" {
		return "", fmt.Errorf("load repository failed")
	}

	e.sigDir = sigDir

	return org, nil
}

func (e *expectState) check(
	org string,
	isStopped func() bool,
	clearLocal func(func(string) bool),
	checkRepo func(*community.Repository, []string, *logrus.Entry),
) {
	allFiles, allSigs, err := e.listAllFilesOfRepo(org)
	if err != nil {
		e.log.Errorf("list all file, err:%s", err.Error())

		allFiles = make(map[string]string)
	}
	getSHA := func(p string) string {
		return allFiles[p]
	}

	repoSigsInfo := make(map[string]string)

	for i := range allFiles {
		expState := e.getRepoFile(i)
		singleRepo := expState.refresh(getSHA)

		path := strings.Split(i, ".yaml")[0]
		pathArr := strings.Split(path, "/")
		repoName := pathArr[4]
		repoSigsInfo[repoName] = pathArr[1]

		for i := 0; i < len(e.reposInfo.Repositories); i++ {
			if e.reposInfo.Repositories[i].Name == repoName {
				e.reposInfo.Repositories = append(e.reposInfo.Repositories[:i], e.reposInfo.Repositories[i+1:]...)
				i--
				break
			}
		}

		e.reposInfo.Repositories = append(e.reposInfo.Repositories, *singleRepo)
	}

	for _, key := range e.reposInfo.Repositories {
		hasSameRepo := false
		for i := range allFiles {
			path := strings.Split(i, ".yaml")[0]
			repoName := strings.Split(path, "/")[4]
			if key.Name == repoName {
				hasSameRepo = true
				break
			}
		}
		if hasSameRepo {
			continue
		}
		for i := 0; i < len(e.reposInfo.Repositories); i++ {
			if e.reposInfo.Repositories[i].Name == key.Name {
				e.reposInfo.Repositories = append(e.reposInfo.Repositories[:i], e.reposInfo.Repositories[i+1:]...)
				delete(repoSigsInfo, key.Name)
				break
			}
		}
	}

	e.reposInfo.Validate()
	repoMap := e.reposInfo.GetRepos()

	if len(repoMap) == 0 {
		// keep safe to do this. it is impossible to happen generally.
		e.log.Warning("there are not repos. Impossible!!!")
		return
	}

	clearLocal(func(r string) bool {
		_, ok := repoMap[r]
		return ok
	})
	getSigSHA := func(p string) string {
		return allSigs[p]
	}

	done := sets.NewString()
	for repo := range repoSigsInfo {
		sigOwner := e.getSigOwner(repoSigsInfo[repo])
		owners := sigOwner.refresh(getSigSHA)
		if isStopped() {
			break
		}

		if org == "openeuler" && repo == "blog" {
			continue
		}

		checkRepo(repoMap[repo], owners.GetOwners(), e.log)

		done.Insert(repo)
	}

	if len(repoMap) == done.Len() {
		return
	}

	for k, repo := range repoMap {
		if isStopped() {
			break
		}

		if !done.Has(k) {
			if org == "openeuler" && k == "blog" {
				continue
			}

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

func (e *expectState) getRepoFile(repoPath string) *expectRepos {
	o, ok := e.repos[repoPath]
	if !ok {
		o = &expectRepos{
			wf: e.newWatchingFile(repoPath),
		}

		e.repos[repoPath] = o
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

func (e *expectState) listAllFilesOfRepo(org string) (map[string]string, map[string]string, error) {
	trees, err := e.cli.GetDirectoryTree(e.w.Org, e.w.Repo, e.w.Branch, 1)
	if err != nil || len(trees.Tree) == 0 {
		return nil, nil, err
	}

	r := make(map[string]string)
	s := make(map[string]string)
	for i := range trees.Tree {
		item := &trees.Tree[i]
		patharr := strings.Split(item.Path, "/")
		if len(patharr) == 0 {
			continue
		}
		if patharr[0] == "sig" && len(patharr) == 5 && patharr[2] == org {
			form := strings.Split(patharr[4], ".")
			if len(form) != 2 || form[1] != "yaml" {
				continue
			}
			r[item.Path] = item.Sha
			continue
		}
		if patharr[0] == "sig" && len(patharr) == 3 && patharr[2] == "OWNERS" {
			s[item.Path] = item.Sha
			continue
		}
	}

	return r, s, nil
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
