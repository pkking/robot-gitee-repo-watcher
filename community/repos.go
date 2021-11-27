package community

import "fmt"

const BranchProtected = "protected"

type Repos struct {
	Version      string       `json:"version,omitempty"`
	Community    string       `json:"community" required:"true"`
	Repositories []Repository `json:"repositories,omitempty"`
}

func (r *Repos) GetCommunity() string {
	if r == nil {
		return ""
	}
	return r.Community
}

func (r *Repos) GetRepos() map[string]*Repository {
	v := make(map[string]*Repository)

	if r == nil {
		return v
	}

	items := r.Repositories
	for i := range items {
		item := &items[i]
		v[item.Name] = item
	}

	return v
}

func (r *Repos) Validate() error {
	for i := range r.Repositories {
		if err := r.Repositories[i].validate(); err != nil {
			return err
		}
	}
	return nil
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
		return fmt.Errorf("missing name")
	}

	if r.Type == "" {
		return fmt.Errorf("missing type")
	}

	for i := range r.Branches {
		if err := r.Branches[i].validate(); err != nil {
			return err
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
		return fmt.Errorf("missing name")
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
	for i := range s.Items {
		if err := s.Items[i].validate(); err != nil {
			return err
		}
	}

	return nil
}

type Sig struct {
	Name         string   `json:"name" required:"true"`
	Repositories []string `json:"repositories,omitempty"`
}

func (s *Sig) validate() error {
	if s.Name == "" {
		return fmt.Errorf("missing name")
	}
	return nil
}

type RepoOwners struct {
	Maintainers []string `json:"maintainers,omitempty"`
	Committers  []string `yaml:"committers,omitempty"`
}

func (r *RepoOwners) GetOwners() []string {
	if r == nil {
		return nil
	}

	v := r.Maintainers
	if len(r.Committers) > 0 {
		v = append(v, r.Committers...)
	}

	return v
}

func (r *RepoOwners) Validate() error {
	return nil
}
