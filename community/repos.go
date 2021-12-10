package community

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	BranchMaster    = "master"
	BranchProtected = "protected"
)

type Repos struct {
	Version      string       `json:"version,omitempty"`
	Community    string       `json:"community" required:"true"`
	Repositories []Repository `json:"repositories,omitempty"`

	repos map[string]*Repository `json:"-"`
}

func (r *Repos) GetCommunity() string {
	if r == nil {
		return ""
	}
	return r.Community
}

func (r *Repos) GetRepos() map[string]*Repository {
	if r == nil {
		return nil
	}

	return r.repos
}

func (r *Repos) Validate() error {
	if r == nil {
		return fmt.Errorf("empty repos")
	}

	s := sets.NewString()
	for i := range r.Repositories {
		item := &r.Repositories[i]

		if err := item.validate(); err != nil {
			return fmt.Errorf("validate %d repository, err:%s", i, err.Error())
		}

		n := item.Name
		if s.Has(n) {
			return fmt.Errorf("validate %d repository, err:duplicate repo:%s", i, n)
		}
		s.Insert(n)
	}

	r.convert()
	return nil
}

func (r *Repos) convert() {
	v := make(map[string]*Repository)

	items := r.Repositories
	for i := range items {
		item := &items[i]
		v[item.Name] = item
	}

	r.repos = v
}

type Repository struct {
	Name              string       `json:"name" required:"true"`
	Type              string       `json:"type" required:"true"`
	RenameFrom        string       `json:"rename_from,omitempty"`
	Description       string       `json:"description,omitempty"`
	Commentable       bool         `json:"commentable,omitempty"`
	ProtectedBranches []string     `json:"protected_branches,omitempty"`
	Branches          []RepoBranch `json:"branches,omitempty"`

	RepoMember
}

func (r *Repository) IsPrivate() bool {
	return r.Type == "private"
}

func (r *Repository) validate() error {
	if r.Name == "" {
		return fmt.Errorf("missing repo name")
	}

	if r.Type == "" {
		return fmt.Errorf("missing repo type")
	}

	for i := range r.Branches {
		if err := r.Branches[i].validate(); err != nil {
			return fmt.Errorf("validate %d branch, err:%s", i, err)
		}
	}

	if n := len(r.ProtectedBranches); n > 0 {
		v := make([]RepoBranch, n)
		for i, item := range r.ProtectedBranches {
			v[i] = RepoBranch{Name: item, Type: BranchProtected}
		}

		if len(r.Branches) > 0 {
			r.Branches = append(r.Branches, v...)
		} else {
			r.Branches = v
		}
	}

	return nil
}

type RepoMember struct {
	Viewers    []string `json:"viewers,omitempty"`
	Managers   []string `json:"managers,omitempty"`
	Reporters  []string `json:"reporters,omitempty"`
	Developers []string `json:"developers,omitempty"`
}

type RepoBranch struct {
	Name       string `json:"name" required:"true"`
	Type       string `json:"type,omitempty"`
	CreateFrom string `json:"create_from,omitempty"`
}

func (r *RepoBranch) validate() error {
	if r.Name == "" {
		return fmt.Errorf("missing branch name")
	}
	return nil
}

type Sigs struct {
	Items []Sig `json:"sigs,omitempty"`
}

func (s *Sigs) GetSigs() []Sig {
	if s == nil {
		return nil
	}

	return s.Items
}

func (s *Sigs) Validate() error {
	if s == nil {
		return fmt.Errorf("empty sigs")
	}

	for i := range s.Items {
		if err := s.Items[i].validate(); err != nil {
			return fmt.Errorf("validate %d sig, err:%s", i, err)
		}
	}

	return nil
}

type Sig struct {
	Name         string   `json:"name" required:"true"`
	Repositories []string `json:"repositories,omitempty"`

	repos map[string][]string `json:"-"`
}

func (s *Sig) GetRepos(org string) []string {
	if s == nil || s.repos == nil {
		return nil
	}

	return s.repos[org]
}

func (s *Sig) validate() error {
	if s.Name == "" {
		return fmt.Errorf("missing sig name")
	}

	s.convert()
	return nil
}

func (s *Sig) convert() {
	v := make(map[string][]string)
	f := func(org, repo string) {
		if r, ok := v[org]; ok {
			v[org] = append(r, repo)
		} else {
			v[org] = []string{repo}
		}
	}

	for _, r := range s.Repositories {
		if a := strings.Split(r, "/"); len(a) > 1 {
			f(a[0], a[1])
		}
	}

	s.repos = v
}

type RepoOwners struct {
	Maintainers []string `json:"maintainers,omitempty"`
}

func (r *RepoOwners) GetOwners() []string {
	if r == nil {
		return nil
	}

	return r.Maintainers
}

func (r *RepoOwners) Validate() error {
	if r == nil {
		return fmt.Errorf("empty repo owners")
	}

	return nil
}
