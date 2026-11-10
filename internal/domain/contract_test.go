package domain_test

import (
	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/teach"
)

// teach.Store and domain.TeachStore must keep the same method set, so a store
// written for one works for the other. Either side drifting fails the build.
var (
	_ domain.TeachStore = (teach.Store)(nil)
	_ teach.Store       = (domain.TeachStore)(nil)
)
