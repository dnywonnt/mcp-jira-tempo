package mcp

import (
	"context"

	"github.com/dnywonnt/mcp-jira-tempo/internal/jira"
)

//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -source=./deps.go -destination=./deps_mock_test.go -package $GOPACKAGE
type JiraClient interface {
	WhoAmI(ctx context.Context, alias string) (jira.User, string, error)
	GetIssue(ctx context.Context, alias, issueKey string) (jira.Issue, error)
	ListIssues(ctx context.Context, req jira.ListIssuesRequest) (jira.ListIssuesResult, error)
	LogTime(ctx context.Context, req jira.LogTimeRequest) (jira.LogTimeResult, error)
	Health(ctx context.Context, alias string) (jira.HealthResult, error)
}
