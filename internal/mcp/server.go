package mcp

import (
	"context"
	"fmt"

	"github.com/dnywonnt/mcp-jira-tempo/internal/config"
	"github.com/dnywonnt/mcp-jira-tempo/internal/jira"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	serverName    = "mcp-jira-tempo"
	serverVersion = "0.0.1"

	toolJiraWhoAmI        = "jira_whoami"
	toolJiraGetIssue      = "jira_get_issue"
	toolJiraListIssues    = "jira_list_issues"
	toolTempoHealth       = "tempo_health_check"
	toolTempoLogTime      = "tempo_log_time"
	descriptionWhoAmI     = "Resolve the authenticated Jira user and selected Tempo worker for an instance."
	descriptionGetIssue   = "Fetch a Jira issue summary and status."
	descriptionListIssues = "Search Jira issues by JQL, or list unresolved issues assigned to current user by default."
	descriptionHealth     = "Check Jira authentication and Tempo Timesheets worklog endpoint availability."
	descriptionLogTime    = "Create a Tempo Timesheets worklog on self-hosted Jira Data Center."
)

type JiraClient interface {
	WhoAmI(ctx context.Context, alias string) (jira.User, string, error)
	GetIssue(ctx context.Context, alias, issueKey string) (jira.Issue, error)
	ListIssues(ctx context.Context, req jira.ListIssuesRequest) (jira.ListIssuesResult, error)
	LogTime(ctx context.Context, req jira.LogTimeRequest) (jira.LogTimeResult, error)
	Health(ctx context.Context, alias string) (jira.HealthResult, error)
}

func NewServer(client JiraClient) *sdkmcp.Server {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    serverName,
		Version: serverVersion,
	}, nil)

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolJiraWhoAmI,
		Description: descriptionWhoAmI,
	}, whoAmIHandler(client))

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolJiraGetIssue,
		Description: descriptionGetIssue,
	}, getIssueHandler(client))

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolJiraListIssues,
		Description: descriptionListIssues,
	}, listIssuesHandler(client))

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolTempoHealth,
		Description: descriptionHealth,
	}, healthHandler(client))

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolTempoLogTime,
		Description: descriptionLogTime,
	}, logTimeHandler(client))

	return server
}

func Serve(ctx context.Context, client JiraClient) error {
	return NewServer(client).Run(ctx, &sdkmcp.StdioTransport{})
}

func whoAmIHandler(client JiraClient) sdkmcp.ToolHandlerFor[InstanceIn, WhoAmIOut] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, In InstanceIn) (*sdkmcp.CallToolResult, WhoAmIOut, error) {
		user, worker, err := client.WhoAmI(ctx, In.Instance)
		if err != nil {
			return nil, WhoAmIOut{}, fmt.Errorf("%s: %w", toolJiraWhoAmI, err)
		}

		return nil, WhoAmIOut{
			Instance:       instanceOrDefault(In.Instance),
			Name:           user.Name,
			Key:            user.Key,
			DisplayName:    user.DisplayName,
			SelectedWorker: worker,
		}, nil
	}
}

func getIssueHandler(client JiraClient) sdkmcp.ToolHandlerFor[GetIssueIn, IssueOut] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, In GetIssueIn) (*sdkmcp.CallToolResult, IssueOut, error) {
		issue, err := client.GetIssue(ctx, In.Instance, In.IssueKey)
		if err != nil {
			return nil, IssueOut{}, fmt.Errorf("%s: %w", toolJiraGetIssue, err)
		}

		return nil, IssueOut{
			ID:      issue.ID,
			Key:     issue.Key,
			Summary: issue.Fields.Summary,
			Status:  issue.Fields.Status.Name,
		}, nil
	}
}

func listIssuesHandler(client JiraClient) sdkmcp.ToolHandlerFor[ListIssuesIn, ListIssuesOut] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, In ListIssuesIn) (*sdkmcp.CallToolResult, ListIssuesOut, error) {
		result, err := client.ListIssues(ctx, jira.ListIssuesRequest{
			Instance:   In.Instance,
			JQL:        In.JQL,
			MaxResults: In.MaxResults,
		})
		if err != nil {
			return nil, ListIssuesOut{}, fmt.Errorf("%s: %w", toolJiraListIssues, err)
		}

		issues := make([]IssueOut, 0, len(result.Issues))
		for _, issue := range result.Issues {
			issues = append(issues, IssueOut{
				ID:      issue.ID,
				Key:     issue.Key,
				Summary: issue.Fields.Summary,
				Status:  issue.Fields.Status.Name,
			})
		}

		return nil, ListIssuesOut{
			Instance:   result.Instance,
			JQL:        result.JQL,
			StartAt:    result.StartAt,
			MaxResults: result.MaxResults,
			Total:      result.Total,
			Issues:     issues,
		}, nil
	}
}

func healthHandler(client JiraClient) sdkmcp.ToolHandlerFor[InstanceIn, HealthOut] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, In InstanceIn) (*sdkmcp.CallToolResult, HealthOut, error) {
		health, err := client.Health(ctx, In.Instance)
		if err != nil {
			return nil, HealthOut{}, fmt.Errorf("%s: %w", toolTempoHealth, err)
		}

		return nil, HealthOut{
			Instance:      health.Instance,
			BaseURL:       health.BaseURL,
			MyselfOK:      health.MyselfOK,
			TempoEndpoint: health.TempoEndpoint,
			TempoStatus:   health.TempoStatus,
			TempoOK:       health.TempoOK,
		}, nil
	}
}

func logTimeHandler(client JiraClient) sdkmcp.ToolHandlerFor[LogTimeIn, LogTimeOut] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, In LogTimeIn) (*sdkmcp.CallToolResult, LogTimeOut, error) {
		result, err := client.LogTime(ctx, jira.LogTimeRequest{
			Instance:        In.Instance,
			IssueKey:        In.IssueKey,
			Date:            In.Date,
			Seconds:         In.Seconds,
			Comment:         In.Comment,
			BillableSeconds: In.BillableSeconds,
			DryRun:          In.DryRun,
		})
		if err != nil {
			return nil, LogTimeOut{}, fmt.Errorf("%s: %w", toolTempoLogTime, err)
		}

		return nil, LogTimeOut{
			Instance: result.Instance,
			Endpoint: result.Endpoint,
			Request: TempoWorklogOut{
				TimeSpentSeconds: result.Request.TimeSpentSeconds,
				BillableSeconds:  result.Request.BillableSeconds,
				Started:          result.Request.Started,
				Comment:          result.Request.Comment,
				Worker:           result.Request.Worker,
				OriginTaskID:     result.Request.OriginTaskID,
			},
			Response: result.Response,
			DryRun:   result.DryRun,
		}, nil
	}
}

func instanceOrDefault(instance string) string {
	if instance == "" {
		return config.DefaultAlias
	}
	return instance
}
