package mcp

import "encoding/json"

type InstanceIn struct {
	Instance string `json:"instance,omitempty" jsonschema:"Jira instance alias from config. Omit for default."`
}

type GetIssueIn struct {
	Instance string `json:"instance,omitempty" jsonschema:"Jira instance alias from config. Omit for default."`
	IssueKey string `json:"issueKey" jsonschema:"Jira issue key, for example PROJ-123."`
}

type ListIssuesIn struct {
	Instance   string `json:"instance,omitempty" jsonschema:"Jira instance alias from config. Omit for default."`
	JQL        string `json:"jql,omitempty" jsonschema:"Jira JQL query. Omit to list unresolved issues assigned to current user."`
	MaxResults int    `json:"maxResults,omitempty" jsonschema:"Maximum number of issues to return. Omit for default."`
}

type LogTimeIn struct {
	Instance        string `json:"instance,omitempty" jsonschema:"Jira instance alias from config. Omit for default."`
	IssueKey        string `json:"issueKey" jsonschema:"Jira issue key, for example PROJ-123."`
	Date            string `json:"date" jsonschema:"Worklog date in YYYY-MM-DD format."`
	Seconds         int    `json:"seconds" jsonschema:"Time spent in seconds."`
	Comment         string `json:"comment" jsonschema:"Worklog comment."`
	BillableSeconds *int   `json:"billableSeconds,omitempty" jsonschema:"Billable time in seconds. Omit to use instance default."`
	DryRun          bool   `json:"dryRun,omitempty" jsonschema:"Return the Tempo request without creating a worklog."`
}

type WhoAmIOut struct {
	Instance       string `json:"instance"`
	Name           string `json:"name,omitempty"`
	Key            string `json:"key,omitempty"`
	DisplayName    string `json:"displayName,omitempty"`
	SelectedWorker string `json:"selectedWorker"`
}

type IssueOut struct {
	ID      string `json:"id"`
	Key     string `json:"key"`
	Summary string `json:"summary"`
	Status  string `json:"status"`
}

type ListIssuesOut struct {
	Instance   string     `json:"instance"`
	JQL        string     `json:"jql"`
	StartAt    int        `json:"startAt"`
	MaxResults int        `json:"maxResults"`
	Total      int        `json:"total"`
	Issues     []IssueOut `json:"issues"`
}

type HealthOut struct {
	Instance      string `json:"instance"`
	BaseURL       string `json:"baseUrl"`
	MyselfOK      bool   `json:"myselfOk"`
	TempoEndpoint string `json:"tempoEndpoint"`
	TempoStatus   int    `json:"tempoStatus"`
	TempoOK       bool   `json:"tempoOk"`
}

type TempoWorklogOut struct {
	TimeSpentSeconds int    `json:"timeSpentSeconds"`
	BillableSeconds  int    `json:"billableSeconds,omitempty"`
	Started          string `json:"started"`
	Comment          string `json:"comment"`
	Worker           string `json:"worker"`
	OriginTaskID     string `json:"originTaskId"`
}

type LogTimeOut struct {
	Instance string          `json:"instance"`
	Endpoint string          `json:"endpoint"`
	Request  TempoWorklogOut `json:"request"`
	Response json.RawMessage `json:"response,omitempty"`
	DryRun   bool            `json:"dryRun"`
}
