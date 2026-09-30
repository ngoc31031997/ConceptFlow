package domain

import (
	"fmt"
	"testing"
)

func TestListFilterMatches(t *testing.T) {
	cases := []struct {
		name   string
		step   int
		state  RunState
		filter ListFilter
		want   bool
	}{
		{"running", FlowRender, RunRunning, ListRunning, true},
		{"idle is not running", FlowRender, RunIdle, ListRunning, false},
		{"failed is a problem", FlowRender, RunFailed, ListProblem, true},
		{"cancelled is a problem", FlowRender, RunCancelled, ListProblem, true},
		{"result is done", FlowResult, RunIdle, ListDone, true},
		{"published is done", FlowPublish, RunDone, ListDone, true},
		{"failed publish is not done", FlowPublish, RunFailed, ListDone, false},
		{"idle mid-flow waits", FlowReview, RunIdle, ListWaiting, true},
		{"idle at result does not wait", FlowResult, RunIdle, ListWaiting, false},
		{"no step does not wait", 0, RunIdle, ListWaiting, false},
		{"all takes anything", FlowPublish, RunFailed, ListAll, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.filter.Matches(ProjectSummary{FlowStep: c.step, RunState: c.state})
			if got != c.want {
				t.Fatalf("%s.Matches(step %d, %s) = %v, want %v", c.filter, c.step, c.state, got, c.want)
			}
		})
	}
}

func TestParseListFilter(t *testing.T) {
	if f, err := ParseListFilter(""); err != nil || f != ListAll {
		t.Fatalf(`"" → %q, %v; want all`, f, err)
	}
	if f, err := ParseListFilter("problem"); err != nil || f != ListProblem {
		t.Fatalf(`"problem" → %q, %v`, f, err)
	}
	if _, err := ParseListFilter("x"); err == nil {
		t.Fatal(`"x" must be rejected`)
	}
}

func summaries(n int) []ProjectSummary {
	out := make([]ProjectSummary, n)
	for i := range out {
		out[i] = ProjectSummary{ProjectID: fmt.Sprintf("p%02d", i), Topic: fmt.Sprintf("t%02d", i), FlowStep: FlowReview, RunState: RunIdle}
	}
	return out
}

func TestPageProjects_Slices(t *testing.T) {
	page := PageProjects(summaries(45), ProjectListQuery{Page: 3, PageSize: 20, Filter: ListAll})
	if len(page.Projects) != 5 || page.Total != 45 || page.Page != 3 || page.PageSize != 20 {
		t.Fatalf("got %d rows, total %d, page %d, size %d", len(page.Projects), page.Total, page.Page, page.PageSize)
	}
	if page.Projects[0].ProjectID != "p40" {
		t.Fatalf("page 3 must start at the 41st project, got %s", page.Projects[0].ProjectID)
	}
}

func TestPageProjects_ClampsPastTheLastPage(t *testing.T) {
	page := PageProjects(summaries(45), ProjectListQuery{Page: 9, PageSize: 20, Filter: ListAll})
	if page.Page != 3 || len(page.Projects) != 5 {
		t.Fatalf("got page %d with %d rows, want page 3 with 5", page.Page, len(page.Projects))
	}
}

func TestPageProjects_Empty(t *testing.T) {
	page := PageProjects(nil, ProjectListQuery{Page: 4, PageSize: 20, Filter: ListAll})
	if page.Page != 1 || page.Total != 0 || len(page.Projects) != 0 {
		t.Fatalf("got %+v", page)
	}
}

func TestPageProjects_FiltersAndCountsOverTheWholeList(t *testing.T) {
	all := []ProjectSummary{
		{ProjectID: "a", FlowStep: FlowRender, RunState: RunRunning},
		{ProjectID: "b", FlowStep: FlowRender, RunState: RunFailed},
		{ProjectID: "c", FlowStep: FlowTTS, RunState: RunCancelled},
		{ProjectID: "d", FlowStep: FlowResult, RunState: RunIdle},
		{ProjectID: "e", FlowStep: FlowReview, RunState: RunIdle},
	}
	page := PageProjects(all, ProjectListQuery{Page: 1, PageSize: 20, Filter: ListProblem, Steps: []int{FlowRender}})
	if page.Total != 1 || page.Projects[0].ProjectID != "b" {
		t.Fatalf("problem at step 10 must be only b, got %+v", page.Projects)
	}
	want := ProjectListCounts{All: 5, Running: 1, Waiting: 1, Problem: 2, Done: 1}
	if page.Counts != want {
		t.Fatalf("counts %+v, want %+v (steps must not narrow them)", page.Counts, want)
	}
}

func TestPageProjects_NamesForkSourceOnAnotherPage(t *testing.T) {
	all := summaries(25)
	all[24].ForkedFrom = "p00"
	all[23].ForkedFrom = "gone"
	page := PageProjects(all, ProjectListQuery{Page: 3, PageSize: 10, Filter: ListAll})
	if got := page.Projects[4].ForkedFromTopic; got != "t00" {
		t.Fatalf("fork source topic = %q, want t00", got)
	}
	if got := page.Projects[3].ForkedFromTopic; got != "" {
		t.Fatalf("a deleted source must stay unnamed, got %q", got)
	}
}
