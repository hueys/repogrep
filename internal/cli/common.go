package cli

import (
	"flag"
	"time"

	"github.com/hueys/repogrep/internal/config"
	"github.com/hueys/repogrep/internal/model"
	gsource "github.com/hueys/repogrep/internal/source"
	"github.com/hueys/repogrep/internal/store"
)

// commonFlags are registered on every subcommand's FlagSet (not just the
// root) so global options can be passed after the subcommand name, e.g.
// `repogrep list --db path.db -v`.
type commonFlags struct {
	db      string
	verbose bool
}

// registerCommon adds --db/-v to fs, defaulted from cfg, and returns a
// handle to read their values back after fs.Parse.
func registerCommon(fs *flag.FlagSet, cfg config.Config) *commonFlags {
	cf := &commonFlags{}
	fs.StringVar(&cf.db, "db", cfg.DBPath, "path to the repogrep SQLite database")
	fs.BoolVar(&cf.verbose, "v", false, "verbose output")
	return cf
}

func openStore(cf *commonFlags) (*store.Store, error) {
	return store.Open(cf.db)
}

// storeUpserter is the slice of *store.Store that ingest needs; narrowing
// it to an interface keeps ingest easy to unit test with a fake.
type storeUpserter interface {
	UpsertRepo(model.Repo) (id int64, inserted bool, err error)
}

// recordToRepo maps a fetched source record onto the store's domain
// struct. now is used as first_seen/last_synced; UpsertRepo preserves the
// existing first_seen on conflict regardless of what's passed here.
func recordToRepo(rec gsource.RepoRecord, now time.Time) model.Repo {
	return model.Repo{
		Source:          rec.SourceName,
		SourceID:        rec.SourceID,
		Owner:           rec.Owner,
		Name:            rec.Name,
		FullName:        rec.FullName,
		Description:     rec.Description,
		URL:             rec.URL,
		Homepage:        rec.Homepage,
		PrimaryLanguage: rec.PrimaryLanguage,
		Topics:          rec.Topics,
		Stars:           rec.Stars,
		Forks:           rec.Forks,
		IsArchived:      rec.IsArchived,
		IsFork:          rec.IsFork,
		License:         rec.License,
		DefaultBranch:   rec.DefaultBranch,
		PushedAt:        rec.PushedAt,
		RepoCreatedAt:   rec.RepoCreatedAt,
		RepoUpdatedAt:   rec.RepoUpdatedAt,
		README:          rec.README,
		ReadmePath:      rec.ReadmePath,
		FirstSeen:       now,
		LastSynced:      now,
	}
}
