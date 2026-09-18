package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dnywonnt/mcp-jira-tempo/internal/config"
	"github.com/stretchr/testify/require"
)

const testToken = "secret-token"

func TestClientWhoAmI(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantUser   User
		wantWorker string
		wantErr    string
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				requireRequest(t, r, http.MethodGet, pathMyself)
				writeJSON(t, w, http.StatusOK, map[string]string{
					"name":        "eugene",
					"key":         "JIRAUSER123",
					"displayName": "Eugene",
				})
			},
			wantUser: User{
				Name:        "eugene",
				Key:         "JIRAUSER123",
				DisplayName: "Eugene",
			},
			wantWorker: "worker-1",
		},
		{
			name: "api error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				requireRequest(t, r, http.MethodGet, pathMyself)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
			},
			wantErr: "c.getJSON: Jira/Tempo API error 401: unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := newTestClient(t, server)
			user, worker, err := client.WhoAmI(context.Background(), "")
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantUser, user)
			require.Equal(t, tt.wantWorker, worker)
		})
	}
}

func TestClientGetIssue(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		handler http.HandlerFunc
		want    Issue
		wantErr string
	}{
		{
			name: "success",
			key:  "PROJ-123",
			handler: func(w http.ResponseWriter, r *http.Request) {
				requireRequest(t, r, http.MethodGet, "/rest/api/2/issue/PROJ-123")
				require.Equal(t, issueFields, r.URL.Query().Get("fields"))
				writeIssue(t, w, http.StatusOK, "10001", "PROJ-123", "Build Jira MCP", "Issue description", "In Progress")
			},
			want: issueFromPayload(t, "10001", "PROJ-123", "Build Jira MCP", "Issue description", "In Progress"),
		},
		{
			name:    "requires issue key",
			key:     "",
			handler: failOnRequest(t),
			wantErr: "issueKey is required",
		},
		{
			name: "api error",
			key:  "PROJ-123",
			handler: func(w http.ResponseWriter, r *http.Request) {
				requireRequest(t, r, http.MethodGet, "/rest/api/2/issue/PROJ-123")
				http.Error(w, "missing issue", http.StatusNotFound)
			},
			wantErr: "c.getJSON: Jira/Tempo API error 404: missing issue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := newTestClient(t, server)
			issue, err := client.GetIssue(context.Background(), "", tt.key)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, issue)
		})
	}
}

func TestClientListIssues(t *testing.T) {
	tests := []struct {
		name           string
		req            ListIssuesRequest
		handler        http.HandlerFunc
		wantJQL        string
		wantMaxResults string
		wantErr        string
	}{
		{
			name:           "default query",
			req:            ListIssuesRequest{},
			wantJQL:        defaultIssueListJQL,
			wantMaxResults: "50",
		},
		{
			name: "custom query clamps max results",
			req: ListIssuesRequest{
				JQL:        "project = ABC ORDER BY updated DESC",
				MaxResults: 150,
			},
			wantJQL:        "project = ABC ORDER BY updated DESC",
			wantMaxResults: "100",
		},
		{
			name:    "rejects negative max results",
			req:     ListIssuesRequest{MaxResults: -1},
			handler: failOnRequest(t),
			wantErr: "maxResults must be greater than or equal to zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.handler
			if handler == nil {
				handler = func(w http.ResponseWriter, r *http.Request) {
					requireRequest(t, r, http.MethodGet, pathSearch)
					require.Equal(t, tt.wantJQL, r.URL.Query().Get("jql"))
					require.Equal(t, issueFields, r.URL.Query().Get("fields"))
					require.Equal(t, tt.wantMaxResults, r.URL.Query().Get("maxResults"))

					writeJSON(t, w, http.StatusOK, map[string]any{
						"startAt":    0,
						"maxResults": 1,
						"total":      1,
						"issues": []any{
							issuePayload("10001", "PROJ-123", "Build Jira MCP", "Issue description", "To Do"),
						},
					})
				}
			}

			server := httptest.NewServer(handler)
			defer server.Close()

			client := newTestClient(t, server)
			result, err := client.ListIssues(context.Background(), tt.req)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, config.DefaultAlias, result.Instance)
			require.Equal(t, tt.wantJQL, result.JQL)
			require.Equal(t, 1, result.MaxResults)
			require.Equal(t, 1, result.Total)
			require.Len(t, result.Issues, 1)
			require.Equal(t, "PROJ-123", result.Issues[0].Key)
		})
	}
}

func TestClientLogTime(t *testing.T) {
	tests := []struct {
		name               string
		req                LogTimeRequest
		handler            http.HandlerFunc
		wantBillableSecond int
		wantDryRun         bool
		wantErr            string
	}{
		{
			name: "uses instance billable default",
			req: LogTimeRequest{
				IssueKey: "PROJ-123",
				Date:     "2026-09-18",
				Seconds:  3600,
				Comment:  "Work on MCP",
			},
			wantBillableSecond: 3600,
		},
		{
			name: "uses request billable seconds",
			req: LogTimeRequest{
				IssueKey:        "PROJ-123",
				Date:            "2026-09-18",
				Seconds:         3600,
				Comment:         "Work on MCP",
				BillableSeconds: new(1800),
			},
			wantBillableSecond: 1800,
		},
		{
			name: "dry run",
			req: LogTimeRequest{
				IssueKey: "PROJ-123",
				Date:     "2026-09-18",
				Seconds:  900,
				Comment:  "Dry run",
				DryRun:   true,
			},
			handler:    failOnRequest(t),
			wantDryRun: true,
		},
		{
			name: "requires issue key",
			req: LogTimeRequest{
				Date:    "2026-09-18",
				Seconds: 1,
			},
			handler: failOnRequest(t),
			wantErr: "issueKey is required",
		},
		{
			name: "requires date",
			req: LogTimeRequest{
				IssueKey: "PROJ-123",
				Seconds:  1,
			},
			handler: failOnRequest(t),
			wantErr: "date is required",
		},
		{
			name: "requires positive seconds",
			req: LogTimeRequest{
				IssueKey: "PROJ-123",
				Date:     "2026-09-18",
			},
			handler: failOnRequest(t),
			wantErr: "seconds must be greater than zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.handler
			if handler == nil {
				handler = func(w http.ResponseWriter, r *http.Request) {
					requireRequest(t, r, http.MethodPost, pathTempoWorklogs)
					require.Equal(t, contentTypeJSON, r.Header.Get(headerContentType))

					var worklog tempoWorklog
					require.NoError(t, json.NewDecoder(r.Body).Decode(&worklog))
					require.Equal(t, tt.req.Seconds, worklog.TimeSpentSeconds)
					require.Equal(t, tt.wantBillableSecond, worklog.BillableSeconds)
					require.Equal(t, tt.req.Date, worklog.Started)
					require.Equal(t, tt.req.Comment, worklog.Comment)
					require.Equal(t, "worker-1", worklog.Worker)
					require.Equal(t, tt.req.IssueKey, worklog.OriginTaskID)

					writeJSON(t, w, http.StatusOK, map[string]string{"id": "worklog-1"})
				}
			}

			server := httptest.NewServer(handler)
			defer server.Close()

			client := newTestClient(t, server)
			result, err := client.LogTime(context.Background(), tt.req)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantDryRun, result.DryRun)
			require.Equal(t, config.DefaultAlias, result.Instance)
			require.Equal(t, server.URL+pathTempoWorklogs, result.Endpoint)
			require.Equal(t, "worker-1", result.Request.Worker)
			if tt.wantDryRun {
				require.Empty(t, result.Response)
				return
			}
			require.Equal(t, "worklog-1", responseID(t, result.Response))
		})
	}
}

func TestClientHealth(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    HealthResult
		wantErr string
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case pathMyself:
					requireRequest(t, r, http.MethodGet, pathMyself)
					writeJSON(t, w, http.StatusOK, map[string]string{"key": "JIRAUSER123"})
				case pathTempoWorklogs:
					requireRequest(t, r, http.MethodGet, pathTempoWorklogs)
					w.WriteHeader(http.StatusMethodNotAllowed)
				default:
					t.Fatalf("unexpected request path: %s", r.URL.Path)
				}
			},
			want: HealthResult{
				Instance:    config.DefaultAlias,
				MyselfOK:    true,
				TempoStatus: http.StatusMethodNotAllowed,
				TempoOK:     true,
			},
		},
		{
			name: "unexpected tempo status",
			handler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case pathMyself:
					requireRequest(t, r, http.MethodGet, pathMyself)
					writeJSON(t, w, http.StatusOK, map[string]string{"key": "JIRAUSER123"})
				case pathTempoWorklogs:
					requireRequest(t, r, http.MethodGet, pathTempoWorklogs)
					w.WriteHeader(http.StatusForbidden)
				default:
					t.Fatalf("unexpected request path: %s", r.URL.Path)
				}
			},
			wantErr: "unexpected Tempo endpoint status: 403",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := newTestClient(t, server)
			result, err := client.Health(context.Background(), "")
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want.Instance, result.Instance)
			require.Equal(t, server.URL, result.BaseURL)
			require.Equal(t, tt.want.MyselfOK, result.MyselfOK)
			require.Equal(t, server.URL+pathTempoWorklogs, result.TempoEndpoint)
			require.Equal(t, tt.want.TempoStatus, result.TempoStatus)
			require.Equal(t, tt.want.TempoOK, result.TempoOK)
		})
	}
}

func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()

	cfg, err := config.Load([]string{
		"JIRA_BASE_URL=" + server.URL,
		"JIRA_TOKEN=" + testToken,
		"JIRA_WORKER=worker-1",
		"JIRA_BILLABLE_BY_DEFAULT=true",
	})
	require.NoError(t, err)

	return NewClient(cfg, server.Client())
}

func requireRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()

	require.Equal(t, method, r.Method)
	require.Equal(t, path, r.URL.Path)
	require.Equal(t, bearerPrefix+testToken, r.Header.Get(headerAuthorization))
	require.Equal(t, contentTypeJSON, r.Header.Get(headerAccept))
}

func writeIssue(t *testing.T, w http.ResponseWriter, status int, id, key, summary, description, issueStatus string) {
	t.Helper()
	writeJSON(t, w, status, issuePayload(id, key, summary, description, issueStatus))
}

func issueFromPayload(t *testing.T, id, key, summary, description, issueStatus string) Issue {
	t.Helper()

	raw, err := json.Marshal(issuePayload(id, key, summary, description, issueStatus))
	require.NoError(t, err)

	var issue Issue
	require.NoError(t, json.Unmarshal(raw, &issue))
	return issue
}

func issuePayload(id, key, summary, description, issueStatus string) map[string]any {
	return map[string]any{
		"id":  id,
		"key": key,
		"fields": map[string]any{
			"summary":     summary,
			"description": description,
			"status": map[string]string{
				"name": issueStatus,
			},
		},
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, payload any) {
	t.Helper()

	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(status)
	require.NoError(t, json.NewEncoder(w).Encode(payload))
}

func responseID(t *testing.T, body json.RawMessage) string {
	t.Helper()

	var response struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(body, &response))
	return response.ID
}

func failOnRequest(t *testing.T) http.HandlerFunc {
	t.Helper()

	return func(_ http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}
}
