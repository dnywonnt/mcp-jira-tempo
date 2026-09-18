package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/dnywonnt/mcp-jira-tempo/internal/config"
	"github.com/dnywonnt/mcp-jira-tempo/internal/jira"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestWhoAmIHandler(t *testing.T) {
	tests := []struct {
		name    string
		input   InstanceIn
		setup   func(client *MockJiraClient)
		want    WhoAmIOut
		wantErr string
	}{
		{
			name:  "success",
			input: InstanceIn{},
			setup: func(client *MockJiraClient) {
				client.EXPECT().
					WhoAmI(gomock.Any(), "").
					Return(jira.User{
						Name:        "eugene",
						Key:         "JIRAUSER123",
						DisplayName: "Eugene",
					}, "worker-1", nil)
			},
			want: WhoAmIOut{
				Instance:       config.DefaultAlias,
				Name:           "eugene",
				Key:            "JIRAUSER123",
				DisplayName:    "Eugene",
				SelectedWorker: "worker-1",
			},
		},
		{
			name:  "error",
			input: InstanceIn{Instance: "corp"},
			setup: func(client *MockJiraClient) {
				client.EXPECT().WhoAmI(gomock.Any(), "corp").Return(jira.User{}, "", errors.New("boom"))
			},
			wantErr: "jira_whoami: boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			client := NewMockJiraClient(ctrl)
			tt.setup(client)

			result, out, err := whoAmIHandler(client)(context.Background(), nil, tt.input)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Nil(t, result)
			require.Equal(t, tt.want, out)
		})
	}
}

func TestGetIssueHandler(t *testing.T) {
	tests := []struct {
		name    string
		input   GetIssueIn
		setup   func(t *testing.T, client *MockJiraClient)
		want    IssueOut
		wantErr string
	}{
		{
			name:  "success",
			input: GetIssueIn{Instance: "corp", IssueKey: "PROJ-123"},
			setup: func(t *testing.T, client *MockJiraClient) {
				t.Helper()
				client.EXPECT().
					GetIssue(gomock.Any(), "corp", "PROJ-123").
					Return(mustIssue(t, "10001", "PROJ-123", "Build MCP", "Issue description", "In Progress"), nil)
			},
			want: IssueOut{
				ID:          "10001",
				Key:         "PROJ-123",
				Summary:     "Build MCP",
				Description: "Issue description",
				Status:      "In Progress",
			},
		},
		{
			name:  "error",
			input: GetIssueIn{Instance: "corp", IssueKey: "PROJ-123"},
			setup: func(t *testing.T, client *MockJiraClient) {
				t.Helper()
				client.EXPECT().GetIssue(gomock.Any(), "corp", "PROJ-123").Return(jira.Issue{}, errors.New("boom"))
			},
			wantErr: "jira_get_issue: boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			client := NewMockJiraClient(ctrl)
			tt.setup(t, client)

			result, out, err := getIssueHandler(client)(context.Background(), nil, tt.input)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Nil(t, result)
			require.Equal(t, tt.want, out)
		})
	}
}

func TestListIssuesHandler(t *testing.T) {
	tests := []struct {
		name    string
		input   ListIssuesIn
		setup   func(t *testing.T, client *MockJiraClient)
		want    ListIssuesOut
		wantErr string
	}{
		{
			name: "success",
			input: ListIssuesIn{
				Instance:   "corp",
				JQL:        "project = PROJ ORDER BY updated DESC",
				MaxResults: 25,
			},
			setup: func(t *testing.T, client *MockJiraClient) {
				t.Helper()
				client.EXPECT().
					ListIssues(gomock.Any(), jira.ListIssuesRequest{
						Instance:   "corp",
						JQL:        "project = PROJ ORDER BY updated DESC",
						MaxResults: 25,
					}).
					Return(jira.ListIssuesResult{
						Instance:   "corp",
						JQL:        "project = PROJ ORDER BY updated DESC",
						StartAt:    0,
						MaxResults: 25,
						Total:      2,
						Issues: []jira.Issue{
							mustIssue(t, "10001", "PROJ-123", "Build MCP", "Build description", "In Progress"),
							mustIssue(t, "10002", "PROJ-124", "Test MCP", "Test description", "To Do"),
						},
					}, nil)
			},
			want: ListIssuesOut{
				Instance:   "corp",
				JQL:        "project = PROJ ORDER BY updated DESC",
				StartAt:    0,
				MaxResults: 25,
				Total:      2,
				Issues: []IssueOut{
					{ID: "10001", Key: "PROJ-123", Summary: "Build MCP", Description: "Build description", Status: "In Progress"},
					{ID: "10002", Key: "PROJ-124", Summary: "Test MCP", Description: "Test description", Status: "To Do"},
				},
			},
		},
		{
			name:  "error",
			input: ListIssuesIn{Instance: "corp"},
			setup: func(t *testing.T, client *MockJiraClient) {
				t.Helper()
				client.EXPECT().
					ListIssues(gomock.Any(), jira.ListIssuesRequest{Instance: "corp"}).
					Return(jira.ListIssuesResult{}, errors.New("boom"))
			},
			wantErr: "jira_list_issues: boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			client := NewMockJiraClient(ctrl)
			tt.setup(t, client)

			result, out, err := listIssuesHandler(client)(context.Background(), nil, tt.input)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Nil(t, result)
			require.Equal(t, tt.want, out)
		})
	}
}

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name    string
		input   InstanceIn
		setup   func(client *MockJiraClient)
		want    HealthOut
		wantErr string
	}{
		{
			name:  "success",
			input: InstanceIn{Instance: "corp"},
			setup: func(client *MockJiraClient) {
				client.EXPECT().
					Health(gomock.Any(), "corp").
					Return(jira.HealthResult{
						Instance:      "corp",
						BaseURL:       "https://jira.example.test",
						MyselfOK:      true,
						TempoEndpoint: "https://jira.example.test/rest/tempo-timesheets/4/worklogs",
						TempoStatus:   405,
						TempoOK:       true,
					}, nil)
			},
			want: HealthOut{
				Instance:      "corp",
				BaseURL:       "https://jira.example.test",
				MyselfOK:      true,
				TempoEndpoint: "https://jira.example.test/rest/tempo-timesheets/4/worklogs",
				TempoStatus:   405,
				TempoOK:       true,
			},
		},
		{
			name:  "error",
			input: InstanceIn{Instance: "corp"},
			setup: func(client *MockJiraClient) {
				client.EXPECT().Health(gomock.Any(), "corp").Return(jira.HealthResult{}, errors.New("boom"))
			},
			wantErr: "tempo_health_check: boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			client := NewMockJiraClient(ctrl)
			tt.setup(client)

			result, out, err := healthHandler(client)(context.Background(), nil, tt.input)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Nil(t, result)
			require.Equal(t, tt.want, out)
		})
	}
}

func TestLogTimeHandler(t *testing.T) {
	billableSeconds := 1800

	tests := []struct {
		name         string
		input        LogTimeIn
		setup        func(t *testing.T, client *MockJiraClient)
		want         LogTimeOut
		wantResponse string
		wantErr      string
	}{
		{
			name: "success",
			input: LogTimeIn{
				Instance:        "corp",
				IssueKey:        "PROJ-123",
				Date:            "2026-09-18",
				Seconds:         3600,
				StartTime:       "09:00",
				Comment:         "Build MCP",
				BillableSeconds: &billableSeconds,
				DryRun:          true,
			},
			setup: func(t *testing.T, client *MockJiraClient) {
				t.Helper()
				client.EXPECT().
					LogTime(gomock.Any(), jira.LogTimeRequest{
						Instance:        "corp",
						IssueKey:        "PROJ-123",
						Date:            "2026-09-18",
						Seconds:         3600,
						StartTime:       "09:00",
						Comment:         "Build MCP",
						BillableSeconds: &billableSeconds,
						DryRun:          true,
					}).
					Return(mustLogTimeResult(t, `{"id":"worklog-1"}`), nil)
			},
			want: LogTimeOut{
				Instance: "corp",
				Endpoint: "https://jira.example.test/rest/tempo-timesheets/4/worklogs",
				Request: TempoWorklogOut{
					TimeSpentSeconds: 3600,
					BillableSeconds:  1800,
					Started:          "2026-09-18",
					Comment:          "Build MCP",
					Worker:           "worker-1",
					OriginTaskID:     "PROJ-123",
				},
				DryRun: true,
			},
			wantResponse: `{"id":"worklog-1"}`,
		},
		{
			name: "array response",
			input: LogTimeIn{
				Instance: "corp",
				IssueKey: "PROJ-123",
				Date:     "2026-09-18",
				Seconds:  3600,
				Comment:  "Build MCP",
			},
			setup: func(t *testing.T, client *MockJiraClient) {
				t.Helper()
				client.EXPECT().
					LogTime(gomock.Any(), jira.LogTimeRequest{
						Instance: "corp",
						IssueKey: "PROJ-123",
						Date:     "2026-09-18",
						Seconds:  3600,
						Comment:  "Build MCP",
					}).
					Return(mustLogTimeResult(t, `[{"tempoWorklogId":353650,"timeSpentSeconds":300,"comment":"тест"}]`), nil)
			},
			want: LogTimeOut{
				Instance: "corp",
				Endpoint: "https://jira.example.test/rest/tempo-timesheets/4/worklogs",
				Request: TempoWorklogOut{
					TimeSpentSeconds: 3600,
					BillableSeconds:  1800,
					Started:          "2026-09-18",
					Comment:          "Build MCP",
					Worker:           "worker-1",
					OriginTaskID:     "PROJ-123",
				},
				DryRun: true,
			},
			wantResponse: `[{"tempoWorklogId":353650,"timeSpentSeconds":300,"comment":"тест"}]`,
		},
		{
			name:  "error",
			input: LogTimeIn{Instance: "corp"},
			setup: func(t *testing.T, client *MockJiraClient) {
				t.Helper()
				client.EXPECT().LogTime(gomock.Any(), jira.LogTimeRequest{Instance: "corp"}).Return(jira.LogTimeResult{}, errors.New("boom"))
			},
			wantErr: "tempo_log_time: boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			client := NewMockJiraClient(ctrl)
			tt.setup(t, client)

			result, out, err := logTimeHandler(client)(context.Background(), nil, tt.input)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Nil(t, result)
			response := out.Response
			out.Response = nil
			require.Equal(t, tt.want, out)
			responseJSON, err := json.Marshal(response)
			require.NoError(t, err)
			require.JSONEq(t, tt.wantResponse, string(responseJSON))
		})
	}
}

func TestLogTimeBulkHandler(t *testing.T) {
	billableSeconds := 1800

	tests := []struct {
		name    string
		input   LogTimeBulkIn
		setup   func(t *testing.T, client *MockJiraClient)
		want    LogTimeBulkOut
		wantErr string
	}{
		{
			name: "partial success",
			input: LogTimeBulkIn{
				Instance: "corp",
				DryRun:   true,
				Entries: []LogTimeEntryIn{
					{
						IssueKey:        "PROJ-123",
						Date:            "2026-09-18",
						Seconds:         3600,
						StartTime:       "09:00",
						Comment:         "Build MCP",
						BillableSeconds: &billableSeconds,
					},
					{
						IssueKey: "PROJ-124",
						Date:     "2026-09-18",
						Seconds:  1800,
						Comment:  "Test MCP",
					},
				},
			},
			setup: func(t *testing.T, client *MockJiraClient) {
				t.Helper()
				client.EXPECT().
					LogTime(gomock.Any(), jira.LogTimeRequest{
						Instance:        "corp",
						IssueKey:        "PROJ-123",
						Date:            "2026-09-18",
						Seconds:         3600,
						StartTime:       "09:00",
						Comment:         "Build MCP",
						BillableSeconds: &billableSeconds,
						DryRun:          true,
					}).
					Return(mustLogTimeResult(t, `{"id":"worklog-1"}`), nil)
				client.EXPECT().
					LogTime(gomock.Any(), jira.LogTimeRequest{
						Instance: "corp",
						IssueKey: "PROJ-124",
						Date:     "2026-09-18",
						Seconds:  1800,
						Comment:  "Test MCP",
						DryRun:   true,
					}).
					Return(jira.LogTimeResult{}, errors.New("boom"))
			},
			want: LogTimeBulkOut{
				Instance:  "corp",
				Total:     2,
				Succeeded: 1,
				Failed:    1,
				DryRun:    true,
				Entries: []LogTimeBulkEntryOut{
					{
						Index:    0,
						IssueKey: "PROJ-123",
						Success:  true,
						Result: &LogTimeOut{
							Instance: "corp",
							Endpoint: "https://jira.example.test/rest/tempo-timesheets/4/worklogs",
							Request: TempoWorklogOut{
								TimeSpentSeconds: 3600,
								BillableSeconds:  1800,
								Started:          "2026-09-18",
								Comment:          "Build MCP",
								Worker:           "worker-1",
								OriginTaskID:     "PROJ-123",
							},
							Response: map[string]any{"id": "worklog-1"},
							DryRun:   true,
						},
					},
					{
						Index:    1,
						IssueKey: "PROJ-124",
						Error:    "boom",
					},
				},
			},
		},
		{
			name:    "requires entries",
			input:   LogTimeBulkIn{Instance: "corp"},
			setup:   func(t *testing.T, client *MockJiraClient) { t.Helper() },
			wantErr: "tempo_log_time_bulk: entries are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			client := NewMockJiraClient(ctrl)
			tt.setup(t, client)

			result, out, err := logTimeBulkHandler(client)(context.Background(), nil, tt.input)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Nil(t, result)
			require.Equal(t, tt.want, out)
		})
	}
}

func mustIssue(t *testing.T, id, key, summary, description, status string) jira.Issue {
	t.Helper()

	raw, err := json.Marshal(map[string]any{
		"id":  id,
		"key": key,
		"fields": map[string]any{
			"summary":     summary,
			"description": description,
			"status": map[string]string{
				"name": status,
			},
		},
	})
	require.NoError(t, err)

	var issue jira.Issue
	require.NoError(t, json.Unmarshal(raw, &issue))
	return issue
}

func mustLogTimeResult(t *testing.T, response string) jira.LogTimeResult {
	t.Helper()

	var result jira.LogTimeResult
	raw := []byte(`{
		"instance": "corp",
		"endpoint": "https://jira.example.test/rest/tempo-timesheets/4/worklogs",
		"request": {
			"timeSpentSeconds": 3600,
			"billableSeconds": 1800,
			"started": "2026-09-18",
			"comment": "Build MCP",
			"worker": "worker-1",
			"originTaskId": "PROJ-123"
		},
		"dryRun": true
	}`)
	require.NoError(t, json.Unmarshal(raw, &result))
	result.Response = json.RawMessage(response)
	return result
}
