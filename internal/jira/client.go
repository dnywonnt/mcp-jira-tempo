package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dnywonnt/mcp-jira-tempo/internal/config"
)

const (
	dateLayout        = "2006-01-02"
	timeLayout        = "15:04"
	startedTimeLayout = "2006-01-02T15:04:05.000"

	pathMyself        = "/rest/api/2/myself"
	pathSearch        = "/rest/api/2/search"
	pathTempoWorklogs = "/rest/tempo-timesheets/4/worklogs"

	issueFields         = "summary,status,description"
	defaultIssueListJQL = "assignee = currentUser() AND resolution = Unresolved ORDER BY updated DESC"
	defaultIssueListMax = 50
	issueListMaxLimit   = 100

	headerAuthorization = "Authorization"
	headerAccept        = "Accept"
	headerContentType   = "Content-Type"

	contentTypeJSON = "application/json"
	bearerPrefix    = "Bearer "
)

type Client struct {
	cfg       config.Config
	http      *http.Client
	workerMu  sync.Mutex
	workerMap map[string]string
}

type searchResult struct {
	StartAt    int     `json:"startAt"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
	Issues     []Issue `json:"issues"`
}

func NewClient(cfg config.Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		cfg:       cfg,
		http:      httpClient,
		workerMap: make(map[string]string),
	}
}

func (c *Client) WhoAmI(ctx context.Context, alias string) (User, string, error) {
	inst, err := c.instance(alias)
	if err != nil {
		return User{}, "", fmt.Errorf("c.instance: %w", err)
	}

	var user User
	if err := c.getJSON(ctx, inst, pathMyself, &user); err != nil {
		return User{}, "", fmt.Errorf("c.getJSON: %w", err)
	}

	worker, err := c.resolveWorker(ctx, inst, user)
	if err != nil {
		return User{}, "", fmt.Errorf("c.resolveWorker: %w", err)
	}
	return user, worker, nil
}

func (c *Client) GetIssue(ctx context.Context, alias, issueKey string) (Issue, error) {
	inst, err := c.instance(alias)
	if err != nil {
		return Issue{}, fmt.Errorf("c.instance: %w", err)
	}
	if strings.TrimSpace(issueKey) == "" {
		return Issue{}, errors.New("issueKey is required")
	}

	path := issuePath(issueKey)
	var issue Issue
	if err := c.getJSON(ctx, inst, path, &issue); err != nil {
		return Issue{}, fmt.Errorf("c.getJSON: %w", err)
	}
	return issue, nil
}

func (c *Client) ListIssues(ctx context.Context, req ListIssuesRequest) (ListIssuesResult, error) {
	inst, err := c.instance(req.Instance)
	if err != nil {
		return ListIssuesResult{}, fmt.Errorf("c.instance: %w", err)
	}

	jql := strings.TrimSpace(req.JQL)
	if jql == "" {
		jql = defaultIssueListJQL
	}

	maxResults := req.MaxResults
	if maxResults == 0 {
		maxResults = defaultIssueListMax
	}
	if maxResults < 0 {
		return ListIssuesResult{}, errors.New("maxResults must be greater than or equal to zero")
	}
	if maxResults > issueListMaxLimit {
		maxResults = issueListMaxLimit
	}

	path := searchPath(jql, maxResults)
	var search searchResult
	if err := c.getJSON(ctx, inst, path, &search); err != nil {
		return ListIssuesResult{}, fmt.Errorf("c.getJSON: %w", err)
	}

	return ListIssuesResult{
		Instance:   inst.Alias,
		JQL:        jql,
		StartAt:    search.StartAt,
		MaxResults: search.MaxResults,
		Total:      search.Total,
		Issues:     search.Issues,
	}, nil
}

func (c *Client) LogTime(ctx context.Context, req LogTimeRequest) (LogTimeResult, error) {
	inst, err := c.instance(req.Instance)
	if err != nil {
		return LogTimeResult{}, fmt.Errorf("c.instance: %w", err)
	}
	if strings.TrimSpace(req.IssueKey) == "" {
		return LogTimeResult{}, errors.New("issueKey is required")
	}
	if strings.TrimSpace(req.Date) == "" {
		return LogTimeResult{}, errors.New("date is required")
	}
	seconds, started, err := resolveWorklogTiming(req)
	if err != nil {
		return LogTimeResult{}, err
	}

	worker, err := c.resolveWorker(ctx, inst, User{})
	if err != nil {
		return LogTimeResult{}, fmt.Errorf("c.resolveWorker: %w", err)
	}

	billableSeconds := 0
	if req.BillableSeconds != nil {
		billableSeconds = *req.BillableSeconds
	} else if inst.BillableByDefault {
		billableSeconds = seconds
	}

	worklog := tempoWorklog{
		TimeSpentSeconds: seconds,
		BillableSeconds:  billableSeconds,
		Started:          started,
		Comment:          req.Comment,
		Worker:           worker,
		OriginTaskID:     req.IssueKey,
	}
	result := LogTimeResult{
		Instance: inst.Alias,
		Endpoint: inst.BaseURL + pathTempoWorklogs,
		Request:  worklog,
		DryRun:   req.DryRun,
	}
	if req.DryRun {
		return result, nil
	}

	body, err := c.postJSON(ctx, inst, pathTempoWorklogs, worklog)
	if err != nil {
		return LogTimeResult{}, fmt.Errorf("c.postJSON: %w", err)
	}
	result.Response = body
	return result, nil
}

func resolveWorklogTiming(req LogTimeRequest) (int, string, error) {
	date, err := time.Parse(dateLayout, req.Date)
	if err != nil {
		return 0, "", fmt.Errorf("parse date: %w", err)
	}

	startTime := strings.TrimSpace(req.StartTime)
	endTime := strings.TrimSpace(req.EndTime)
	if startTime == "" && endTime == "" {
		if req.Seconds <= 0 {
			return 0, "", errors.New("seconds must be greater than zero")
		}
		return req.Seconds, req.Date, nil
	}
	if startTime == "" {
		return 0, "", errors.New("startTime is required when endTime is provided")
	}

	start, err := time.Parse(timeLayout, startTime)
	if err != nil {
		return 0, "", fmt.Errorf("time.Parse: %w", err)
	}
	startedAt := time.Date(date.Year(), date.Month(), date.Day(), start.Hour(), start.Minute(), 0, 0, time.UTC)
	if endTime == "" {
		if req.Seconds <= 0 {
			return 0, "", errors.New("seconds must be greater than zero when endTime is omitted")
		}
		return req.Seconds, startedAt.Format(startedTimeLayout), nil
	}

	end, err := time.Parse(timeLayout, endTime)
	if err != nil {
		return 0, "", fmt.Errorf("time.Parse: %w", err)
	}

	endedAt := time.Date(date.Year(), date.Month(), date.Day(), end.Hour(), end.Minute(), 0, 0, time.UTC)
	if !endedAt.After(startedAt) {
		return 0, "", errors.New("endTime must be after startTime")
	}

	seconds := int(endedAt.Sub(startedAt).Seconds())
	if req.Seconds > 0 && req.Seconds != seconds {
		return 0, "", fmt.Errorf("seconds must match time range duration: got %d, want %d", req.Seconds, seconds)
	}

	return seconds, startedAt.Format(startedTimeLayout), nil
}

func (c *Client) Health(ctx context.Context, alias string) (HealthResult, error) {
	inst, err := c.instance(alias)
	if err != nil {
		return HealthResult{}, fmt.Errorf("c.instance: %w", err)
	}

	result := HealthResult{
		Instance:      inst.Alias,
		BaseURL:       inst.BaseURL,
		TempoEndpoint: inst.BaseURL + pathTempoWorklogs,
	}

	var user User
	if err := c.getJSON(ctx, inst, pathMyself, &user); err != nil {
		return result, fmt.Errorf("c.getJSON: %w", err)
	}
	result.MyselfOK = true

	status, _, err := c.do(ctx, inst, http.MethodGet, pathTempoWorklogs, nil)
	if err != nil {
		return result, fmt.Errorf("c.do: %w", err)
	}
	result.TempoStatus = status
	result.TempoOK = status == http.StatusMethodNotAllowed || status == http.StatusOK || status == http.StatusBadRequest
	if !result.TempoOK {
		return result, fmt.Errorf("unexpected Tempo endpoint status: %d", status)
	}
	return result, nil
}

func (c *Client) instance(alias string) (config.Instance, error) {
	inst, ok := c.cfg.Instance(alias)
	if !ok {
		return config.Instance{}, fmt.Errorf("unknown Jira instance %q", alias)
	}
	return inst, nil
}

func (c *Client) resolveWorker(ctx context.Context, inst config.Instance, user User) (string, error) {
	if inst.Worker != "" {
		return inst.Worker, nil
	}

	c.workerMu.Lock()
	cached := c.workerMap[inst.Alias]
	c.workerMu.Unlock()
	if cached != "" {
		return cached, nil
	}

	if user.Key == "" && user.Name == "" {
		if err := c.getJSON(ctx, inst, pathMyself, &user); err != nil {
			return "", fmt.Errorf("c.getJSON: %w", err)
		}
	}

	worker := user.Key
	if inst.WorkerField == config.WorkerFieldName {
		worker = user.Name
	}
	if worker == "" {
		return "", fmt.Errorf("could not resolve Tempo worker from %s using field %q", pathMyself, inst.WorkerField)
	}

	c.workerMu.Lock()
	c.workerMap[inst.Alias] = worker
	c.workerMu.Unlock()
	return worker, nil
}

func (c *Client) getJSON(ctx context.Context, inst config.Instance, path string, target any) error {
	status, body, err := c.do(ctx, inst, http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("c.do: %w", err)
	}
	if status < 200 || status >= 300 {
		return apiError(status, body)
	}
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) postJSON(ctx context.Context, inst config.Instance, path string, payload any) (json.RawMessage, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	status, body, err := c.do(ctx, inst, http.MethodPost, path, bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("c.do: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, apiError(status, body)
	}
	return json.RawMessage(body), nil
}

func (c *Client) do(ctx context.Context, inst config.Instance, method, path string, body []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, inst.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return 0, nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set(headerAuthorization, bearerPrefix+inst.Token)
	req.Header.Set(headerAccept, contentTypeJSON)
	if body != nil {
		req.Header.Set(headerContentType, contentTypeJSON)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, c.cfg.MaxResponseBytes()))
	if err != nil {
		return 0, nil, fmt.Errorf("read response: %w", err)
	}
	return resp.StatusCode, respBody, nil
}

func issuePath(issueKey string) string {
	values := url.Values{}
	values.Set("fields", issueFields)
	return "/rest/api/2/issue/" + url.PathEscape(issueKey) + "?" + values.Encode()
}

func searchPath(jql string, maxResults int) string {
	values := url.Values{}
	values.Set("jql", jql)
	values.Set("fields", issueFields)
	values.Set("maxResults", strconv.Itoa(maxResults))
	return pathSearch + "?" + values.Encode()
}

func apiError(status int, body []byte) error {
	text := strings.TrimSpace(string(body))
	if text == "" {
		text = http.StatusText(status)
	}
	return fmt.Errorf("Jira/Tempo API error %d: %s", status, text)
}
