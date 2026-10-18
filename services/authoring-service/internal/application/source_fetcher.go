package application

import "context"

// SourcePage is a fetched page reduced to its text.
type SourcePage struct {
	Text string
}

// FetchError is why a source page could not be read, in the Creator's words
// (a paywall, a missing page, an address the service may not call, a time out).
type FetchError struct {
	Reason string
}

func (e *FetchError) Error() string { return e.Reason }

// SourceFetcher downloads a web page for the source check. Every failure to
// read the page is a *FetchError; a cancelled context is returned as it is.
type SourceFetcher interface {
	Fetch(ctx context.Context, rawURL string) (SourcePage, error)
}

// ResearchSource is where research facts come from. The only source today is
// the Creator pasting the answer of an outside research tool; a search API
// can be added behind the same name without changing the fact table.
type ResearchSource interface {
	Name() string
}

// PasteSource is the research source where the Creator pastes an outside
// tool's answer.
type PasteSource struct{}

// Name returns "paste".
func (PasteSource) Name() string { return "paste" }
