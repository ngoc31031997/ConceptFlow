package domain

import "fmt"

// The video list is paged on the server. flow_step and run_state are
// derived in Go (a draft's step needs authoring-service content), so the list
// is filtered, counted and sliced here rather than with LIMIT/OFFSET in SQL —
// one set of rules, next to FlowStateFor.

// ListFilter is one of the video list's status chips.
type ListFilter string

const (
	ListAll     ListFilter = "all"
	ListRunning ListFilter = "running"
	ListWaiting ListFilter = "waiting" // waiting for the Creator
	ListProblem ListFilter = "problem" // failed or cancelled
	ListDone    ListFilter = "done"
)

const (
	DefaultProjectPageSize = 20
	MaxProjectPageSize     = 100
)

// ParseListFilter reads a filter query value; "" means all.
func ParseListFilter(s string) (ListFilter, error) {
	switch f := ListFilter(s); f {
	case "":
		return ListAll, nil
	case ListAll, ListRunning, ListWaiting, ListProblem, ListDone:
		return f, nil
	}
	return "", fmt.Errorf("unknown filter %q", s)
}

// Matches says whether a project belongs under this chip.
func (f ListFilter) Matches(s ProjectSummary) bool {
	switch f {
	case ListRunning:
		return s.RunState == RunRunning
	case ListProblem:
		return s.RunState == RunFailed || s.RunState == RunCancelled
	case ListDone:
		return s.FlowStep >= FlowResult && s.RunState != RunFailed
	case ListWaiting:
		return s.RunState == RunIdle && s.FlowStep > 0 && s.FlowStep < FlowResult
	}
	return true
}

// ProjectListQuery is one page request. Steps empty means every step.
type ProjectListQuery struct {
	Page, PageSize int
	Filter         ListFilter
	Steps          []int
}

// ProjectListCounts is how many projects each chip holds, over the whole list.
type ProjectListCounts struct {
	All, Running, Waiting, Problem, Done int
}

// ProjectPage is one page of the filtered list. Page is the page actually
// returned: a request past the last page gets the last page.
type ProjectPage struct {
	Projects []ProjectSummary
	Total    int
	Page     int
	PageSize int
	Counts   ProjectListCounts
}

// FillForkedFromTopics names each fork's source from the same list, since the
// source may sit on another page. A source that is gone stays "".
func FillForkedFromTopics(all []ProjectSummary) {
	topics := make(map[string]string, len(all))
	for _, p := range all {
		topics[p.ProjectID] = p.Topic
	}
	for i := range all {
		if all[i].ForkedFrom != "" {
			all[i].ForkedFromTopic = topics[all[i].ForkedFrom]
		}
	}
}

// PageProjects counts the chips over all, filters by q.Filter and q.Steps,
// then returns page q.Page (clamped to [1, last page]) keeping all's order.
func PageProjects(all []ProjectSummary, q ProjectListQuery) ProjectPage {
	FillForkedFromTopics(all)

	var counts ProjectListCounts
	steps := make(map[int]bool, len(q.Steps))
	for _, s := range q.Steps {
		steps[s] = true
	}
	filtered := make([]ProjectSummary, 0, len(all))
	for _, p := range all {
		counts.All++
		if ListRunning.Matches(p) {
			counts.Running++
		}
		if ListWaiting.Matches(p) {
			counts.Waiting++
		}
		if ListProblem.Matches(p) {
			counts.Problem++
		}
		if ListDone.Matches(p) {
			counts.Done++
		}
		if q.Filter.Matches(p) && (len(steps) == 0 || steps[p.FlowStep]) {
			filtered = append(filtered, p)
		}
	}

	size := q.PageSize
	if size < 1 {
		size = DefaultProjectPageSize
	}
	pages := (len(filtered) + size - 1) / size
	if pages < 1 {
		pages = 1
	}
	page := min(max(q.Page, 1), pages)
	start := (page - 1) * size
	end := min(start+size, len(filtered))
	return ProjectPage{
		Projects: filtered[start:end],
		Total:    len(filtered),
		Page:     page,
		PageSize: size,
		Counts:   counts,
	}
}
