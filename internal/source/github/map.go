package github

import (
	"strconv"

	"github.com/google/go-github/v75/github"
	"github.com/hueys/repogrep/internal/source"
)

// mapRepo converts a github.Repository (as returned by ListStarred, without
// README content) into a source.RepoRecord. README/ReadmePath are filled in
// separately since they require a dedicated API call per repo.
func mapRepo(r *github.Repository) source.RepoRecord {
	rec := source.RepoRecord{
		SourceName:      "github",
		SourceID:        strconv.FormatInt(r.GetID(), 10),
		Owner:           r.GetOwner().GetLogin(),
		Name:            r.GetName(),
		FullName:        r.GetFullName(),
		Description:     r.GetDescription(),
		URL:             r.GetHTMLURL(),
		Homepage:        r.GetHomepage(),
		PrimaryLanguage: r.GetLanguage(),
		Topics:          append([]string(nil), r.Topics...),
		Stars:           r.GetStargazersCount(),
		Forks:           r.GetForksCount(),
		IsArchived:      r.GetArchived(),
		IsFork:          r.GetFork(),
		DefaultBranch:   r.GetDefaultBranch(),
		PushedAt:        r.GetPushedAt().Time,
		RepoCreatedAt:   r.GetCreatedAt().Time,
		RepoUpdatedAt:   r.GetUpdatedAt().Time,
	}
	if lic := r.GetLicense(); lic != nil {
		if spdx := lic.GetSPDXID(); spdx != "" && spdx != "NOASSERTION" {
			rec.License = spdx
		} else {
			rec.License = lic.GetName()
		}
	}
	return rec
}
