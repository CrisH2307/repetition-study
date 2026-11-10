//go:build tools

// Package deps pins third-party modules so later units never edit go.mod.
// ponytail: delete an import here once a real package imports that module.
package deps

import (
	_ "github.com/anthropics/anthropic-sdk-go"
	_ "github.com/jackc/pgx/v5"
	_ "github.com/open-spaced-repetition/go-fsrs/v4"
	_ "google.golang.org/genai"
)
