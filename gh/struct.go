package gh

import (
	"log/slog"

	"github.com/google/go-github/v73/github"
)

type Github struct {
	Logger *slog.Logger
	Client *github.Client
}
