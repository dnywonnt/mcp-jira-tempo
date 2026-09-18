package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultAlias = "default"

	DefaultHTTPClientTimeout = 30 * time.Second
	DefaultMaxResponseBytes  = int64(1 << 20)

	WorkerFieldKey  = "key"
	WorkerFieldName = "name"

	envJiraInstances         = "JIRA_INSTANCES"
	envJiraDefaultInstance   = "JIRA_DEFAULT_INSTANCE"
	envJiraHTTPClientTimeout = "JIRA_HTTP_CLIENT_TIMEOUT"
	envJiraMaxResponseBytes  = "JIRA_MAX_RESPONSE_BYTES"
	envJiraBaseURL           = "JIRA_BASE_URL"
	envJiraToken             = "JIRA_TOKEN"
	envJiraWorker            = "JIRA_WORKER"
	envJiraWorkerField       = "JIRA_WORKER_FIELD"
	envJiraBillableDefault   = "JIRA_BILLABLE_BY_DEFAULT"
)

var aliasEnvPattern = regexp.MustCompile(`[^A-Z0-9]+`)

type Config struct {
	defaultAlias      string
	instances         map[string]Instance
	httpClientTimeout time.Duration
	maxResponseBytes  int64
}

type Instance struct {
	Alias             string
	BaseURL           string
	Token             string
	Worker            string
	WorkerField       string
	BillableByDefault bool
}

func LoadDefault() (Config, error) {
	return Load(os.Environ())
}

func Load(environ []string) (Config, error) {
	env := parseEnv(environ)

	aliases := splitList(env[envJiraInstances])
	if len(aliases) == 0 {
		aliases = []string{DefaultAlias}
	}

	defaultAlias := strings.TrimSpace(env[envJiraDefaultInstance])
	if defaultAlias == "" {
		defaultAlias = aliases[0]
	}

	cfg := Config{
		defaultAlias:      defaultAlias,
		instances:         make(map[string]Instance, len(aliases)),
		httpClientTimeout: DefaultHTTPClientTimeout,
		maxResponseBytes:  DefaultMaxResponseBytes,
	}

	if value := strings.TrimSpace(env[envJiraHTTPClientTimeout]); value != "" {
		timeout, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("config: %s: %w", envJiraHTTPClientTimeout, err)
		}
		if timeout <= 0 {
			return Config{}, fmt.Errorf("config: %s must be greater than zero", envJiraHTTPClientTimeout)
		}
		cfg.httpClientTimeout = timeout
	}

	if value := strings.TrimSpace(env[envJiraMaxResponseBytes]); value != "" {
		maxBytes, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("config: %s: %w", envJiraMaxResponseBytes, err)
		}
		if maxBytes <= 0 {
			return Config{}, fmt.Errorf("config: %s must be greater than zero", envJiraMaxResponseBytes)
		}
		cfg.maxResponseBytes = maxBytes
	}

	for _, alias := range aliases {
		inst, err := loadInstance(env, alias)
		if err != nil {
			return Config{}, err
		}
		key := normalizeAlias(inst.Alias)
		if _, exists := cfg.instances[key]; exists {
			return Config{}, fmt.Errorf("config: duplicate Jira instance alias %q", inst.Alias)
		}
		cfg.instances[key] = inst
	}

	if _, ok := cfg.instances[normalizeAlias(cfg.defaultAlias)]; !ok {
		return Config{}, fmt.Errorf("config: default Jira instance %q is not configured", cfg.defaultAlias)
	}

	return cfg, nil
}

func (c Config) Instance(alias string) (Instance, bool) {
	if strings.TrimSpace(alias) == "" {
		alias = c.defaultAlias
	}
	inst, ok := c.instances[normalizeAlias(alias)]
	return inst, ok
}

func (c Config) HTTPClientTimeout() time.Duration {
	return c.httpClientTimeout
}

func (c Config) MaxResponseBytes() int64 {
	return c.maxResponseBytes
}

func (c Config) Instances() []Instance {
	out := make([]Instance, 0, len(c.instances))
	for _, inst := range c.instances {
		out = append(out, inst)
	}
	slices.SortFunc(out, func(a, b Instance) int {
		return strings.Compare(a.Alias, b.Alias)
	})
	return out
}

func loadInstance(env map[string]string, alias string) (Instance, error) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return Instance{}, errors.New("config: Jira instance alias is required")
	}

	baseURL := strings.TrimRight(strings.TrimSpace(instanceEnv(env, alias, "BASE_URL", envJiraBaseURL)), "/")
	if baseURL == "" {
		return Instance{}, fmt.Errorf("config: %s is required", instanceEnvName(alias, "BASE_URL", envJiraBaseURL))
	}
	if err := validateBaseURL(baseURL); err != nil {
		return Instance{}, fmt.Errorf("config: %s: %w", instanceEnvName(alias, "BASE_URL", envJiraBaseURL), err)
	}

	token := strings.TrimSpace(instanceEnv(env, alias, "TOKEN", envJiraToken))
	if token == "" {
		return Instance{}, fmt.Errorf("config: %s is required", instanceEnvName(alias, "TOKEN", envJiraToken))
	}

	workerField := strings.TrimSpace(instanceEnv(env, alias, "WORKER_FIELD", envJiraWorkerField))
	if workerField == "" {
		workerField = WorkerFieldKey
	}
	if workerField != WorkerFieldKey && workerField != WorkerFieldName {
		return Instance{}, fmt.Errorf("config: %s must be %q or %q", instanceEnvName(alias, "WORKER_FIELD", envJiraWorkerField), WorkerFieldKey, WorkerFieldName)
	}

	billableByDefault, err := parseOptionalBool(instanceEnv(env, alias, "BILLABLE_BY_DEFAULT", envJiraBillableDefault))
	if err != nil {
		return Instance{}, fmt.Errorf("config: %s: %w", instanceEnvName(alias, "BILLABLE_BY_DEFAULT", envJiraBillableDefault), err)
	}

	return Instance{
		Alias:             alias,
		BaseURL:           baseURL,
		Token:             token,
		Worker:            strings.TrimSpace(instanceEnv(env, alias, "WORKER", envJiraWorker)),
		WorkerField:       workerField,
		BillableByDefault: billableByDefault,
	}, nil
}

func instanceEnv(env map[string]string, alias, suffix, fallback string) string {
	name := aliasEnvName(alias, suffix)
	if value, ok := env[name]; ok {
		return value
	}
	if alias == DefaultAlias {
		return env[fallback]
	}
	return ""
}

func instanceEnvName(alias, suffix, fallback string) string {
	if alias == DefaultAlias {
		return fallback + " or " + aliasEnvName(alias, suffix)
	}
	return aliasEnvName(alias, suffix)
}

func aliasEnvName(alias, suffix string) string {
	normalized := aliasEnvPattern.ReplaceAllString(strings.ToUpper(alias), "_")
	normalized = strings.Trim(normalized, "_")
	return "JIRA_" + normalized + "_" + suffix
}

func normalizeAlias(alias string) string {
	return strings.ToLower(strings.TrimSpace(alias))
}

func parseEnv(environ []string) map[string]string {
	env := make(map[string]string, len(environ))
	for _, item := range environ {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			env[key] = value
		}
	}
	return env
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseOptionalBool(value string) (bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return false, nil
	}
	return strconv.ParseBool(value)
}

func validateBaseURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("must use http or https scheme")
	}
	if parsed.Host == "" {
		return errors.New("host is required")
	}
	return nil
}
