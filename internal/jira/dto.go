package jira

import "encoding/json"

type LogTimeRequest struct {
	Instance        string
	IssueKey        string
	Date            string
	Seconds         int
	Comment         string
	BillableSeconds *int
	DryRun          bool
}

type ListIssuesRequest struct {
	Instance   string
	JQL        string
	MaxResults int
}

type LogTimeResult struct {
	Instance string          `json:"instance"`
	Endpoint string          `json:"endpoint"`
	Request  tempoWorklog    `json:"request"`
	Response json.RawMessage `json:"response,omitempty"`
	DryRun   bool            `json:"dryRun"`
}

type HealthResult struct {
	Instance      string `json:"instance"`
	BaseURL       string `json:"baseUrl"`
	MyselfOK      bool   `json:"myselfOk"`
	TempoEndpoint string `json:"tempoEndpoint"`
	TempoStatus   int    `json:"tempoStatus"`
	TempoOK       bool   `json:"tempoOk"`
}

type ListIssuesResult struct {
	Instance   string  `json:"instance"`
	JQL        string  `json:"jql"`
	StartAt    int     `json:"startAt"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
	Issues     []Issue `json:"issues"`
}

type User struct {
	Name        string `json:"name"`
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
}

type Issue struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
		Status  struct {
			Name string `json:"name"`
		} `json:"status"`
	} `json:"fields"`
}

type tempoWorklog struct {
	TimeSpentSeconds int    `json:"timeSpentSeconds"`
	BillableSeconds  int    `json:"billableSeconds,omitempty"`
	Started          string `json:"started"`
	Comment          string `json:"comment"`
	Worker           string `json:"worker"`
	OriginTaskID     string `json:"originTaskId"`
}
