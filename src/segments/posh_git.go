package segments

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/po1o/prompto/src/log"
)

const (
	poshGitEnv = "PROMPTO_GIT_STATUS"
)

type poshGit struct {
	Index        *poshGitStatus `json:"Index"`
	Working      *poshGitStatus `json:"Working"`
	RepoName     string         `json:"RepoName"`
	Branch       string         `json:"Branch"`
	GitDir       string         `json:"GitDir"`
	Upstream     string         `json:"Upstream"`
	StashCount   int            `json:"StashCount"`
	AheadBy      int            `json:"AheadBy"`
	BehindBy     int            `json:"BehindBy"`
	HasWorking   bool           `json:"HasWorking"`
	HasIndex     bool           `json:"HasIndex"`
	HasUntracked bool           `json:"HasUntracked"`
}

type poshGitStatus struct {
	Added    []string `json:"Added"`
	Modified []string `json:"Modified"`
	Deleted  []string `json:"Deleted"`
	Unmerged []string `json:"Unmerged"`
}

func (s *GitStatus) parsePoshGitStatus(p *poshGitStatus) {
	if p == nil {
		return
	}

	s.Added = len(p.Added)
	s.Deleted = len(p.Deleted)
	s.Modified = len(p.Modified)
	s.Unmerged = len(p.Unmerged)
}

func (g *Git) hasPoshGitStatus() bool {
	envStatus := g.env.Getenv(poshGitEnv)
	if envStatus == "" {
		log.Error(fmt.Errorf("%s environment variable not set, do you have the prompto-git module installed?", poshGitEnv))
		return false
	}

	var prompto poshGit
	err := json.Unmarshal([]byte(envStatus), &prompto)
	if err != nil {
		log.Error(err)
		return false
	}

	g.setDir(prompto.GitDir)
	g.Working = &GitStatus{}
	g.Working.parsePoshGitStatus(prompto.Working)
	g.Staging = &GitStatus{}
	g.Staging.parsePoshGitStatus(prompto.Index)
	g.HEAD = g.parsePoshGitHEAD(prompto.Branch)
	g.stashCount = prompto.StashCount
	g.Ahead = prompto.AheadBy
	g.Behind = prompto.BehindBy
	g.UpstreamGone = prompto.Upstream == ""
	g.Upstream = prompto.Upstream

	g.setBranchStatus()

	if len(g.Upstream) != 0 && g.options.Bool(FetchUpstreamIcon, false) {
		g.UpstreamIcon = g.getUpstreamIcon()
	}

	g.poshgit = true
	return true
}

func (g *Git) parsePoshGitHEAD(head string) string {
	// commit
	if strings.HasSuffix(head, "...)") {
		head = strings.TrimLeft(head, "(")
		head = strings.TrimRight(head, ".)")
		return fmt.Sprintf("%s%s", g.options.String(CommitIcon, "\uF417"), head)
	}
	// tag
	if strings.HasPrefix(head, "(") {
		head = strings.TrimLeft(head, "(")
		head = strings.TrimRight(head, ")")
		return fmt.Sprintf("%s%s", g.options.String(TagIcon, "\uF412"), head)
	}
	// regular branch
	return fmt.Sprintf("%s%s", g.options.String(BranchIcon, "\uE0A0"), g.formatBranch(head))
}
