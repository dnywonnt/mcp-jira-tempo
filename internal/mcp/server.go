package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dnywonnt/mcp-jira-tempo/internal/config"
	"github.com/dnywonnt/mcp-jira-tempo/internal/jira"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	serverName    = "mcp-jira-tempo"
	serverVersion = "0.0.1"

	toolJiraWhoAmI         = "jira_whoami"
	toolJiraGetIssue       = "jira_get_issue"
	toolJiraListIssues     = "jira_list_issues"
	toolTempoHealth        = "tempo_health_check"
	toolTempoLogTime       = "tempo_log_time"
	toolTempoLogTimeBulk   = "tempo_log_time_bulk"
	descriptionWhoAmI      = "Resolve the authenticated Jira user and selected Tempo worker for an instance."
	descriptionGetIssue    = "Fetch a Jira issue summary and status."
	descriptionListIssues  = "Search Jira issues by JQL, or list unresolved issues assigned to current user by default."
	descriptionHealth      = "Check Jira authentication and Tempo Timesheets worklog endpoint availability."
	descriptionLogTime     = "Create a Tempo Timesheets worklog on self-hosted Jira Data Center."
	descriptionLogTimeBulk = "Create multiple Tempo Timesheets worklogs on self-hosted Jira Data Center."
)

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

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolTempoLogTimeBulk,
		Description: descriptionLogTimeBulk,
	}, logTimeBulkHandler(client))

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
			ID:          issue.ID,
			Key:         issue.Key,
			Summary:     issue.Fields.Summary,
			Description: issue.Fields.Description,
			Status:      issue.Fields.Status.Name,
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
				ID:          issue.ID,
				Key:         issue.Key,
				Summary:     issue.Fields.Summary,
				Description: issue.Fields.Description,
				Status:      issue.Fields.Status.Name,
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
			StartTime:       In.StartTime,
			EndTime:         In.EndTime,
			Comment:         In.Comment,
			BillableSeconds: In.BillableSeconds,
			DryRun:          In.DryRun,
		})
		if err != nil {
			return nil, LogTimeOut{}, fmt.Errorf("%s: %w", toolTempoLogTime, err)
		}

		out, err := logTimeOut(result)
		if err != nil {
			return nil, LogTimeOut{}, fmt.Errorf("%s: %w", toolTempoLogTime, err)
		}

		return nil, out, nil
	}
}

func logTimeBulkHandler(client JiraClient) sdkmcp.ToolHandlerFor[LogTimeBulkIn, LogTimeBulkOut] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, In LogTimeBulkIn) (*sdkmcp.CallToolResult, LogTimeBulkOut, error) {
		if len(In.Entries) == 0 {
			return nil, LogTimeBulkOut{}, fmt.Errorf("%s: entries are required", toolTempoLogTimeBulk)
		}

		out := LogTimeBulkOut{
			Instance: instanceOrDefault(In.Instance),
			Total:    len(In.Entries),
			DryRun:   In.DryRun,
			Entries:  make([]LogTimeBulkEntryOut, 0, len(In.Entries)),
		}

		for i, entry := range In.Entries {
			item := LogTimeBulkEntryOut{
				Index:    i,
				IssueKey: entry.IssueKey,
			}

			result, err := client.LogTime(ctx, jira.LogTimeRequest{
				Instance:        In.Instance,
				IssueKey:        entry.IssueKey,
				Date:            entry.Date,
				Seconds:         entry.Seconds,
				StartTime:       entry.StartTime,
				EndTime:         entry.EndTime,
				Comment:         entry.Comment,
				BillableSeconds: entry.BillableSeconds,
				DryRun:          In.DryRun,
			})
			if err != nil {
				item.Error = err.Error()
				out.Failed++
				out.Entries = append(out.Entries, item)
				continue
			}

			logged, err := logTimeOut(result)
			if err != nil {
				item.Error = err.Error()
				out.Failed++
				out.Entries = append(out.Entries, item)
				continue
			}

			item.Success = true
			item.Result = &logged
			out.Succeeded++
			out.Entries = append(out.Entries, item)
		}

		return nil, out, nil
	}
}

func logTimeOut(result jira.LogTimeResult) (LogTimeOut, error) {
	response, err := decodeRawJSONResponse(result.Response)
	if err != nil {
		return LogTimeOut{}, err
	}

	return LogTimeOut{
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
		Response: response,
		DryRun:   result.DryRun,
	}, nil
}

func decodeRawJSONResponse(body json.RawMessage) (any, error) {
	if len(body) == 0 {
		return nil, nil
	}

	var response any
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode raw JSON response: %w", err)
	}
	return response, nil
}

func instanceOrDefault(instance string) string {
	if instance == "" {
		return config.DefaultAlias
	}
	return instance
}
