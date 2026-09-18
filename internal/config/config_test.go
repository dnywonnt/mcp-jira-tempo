package config

import "testing"

func TestLoadDefaultInstanceFromGenericEnv(t *testing.T) {
	cfg, err := Load([]string{
		"JIRA_BASE_URL=https://jira.example.test/",
		"JIRA_TOKEN=secret",
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	inst, ok := cfg.Instance("")
	if !ok {
		t.Fatal("default instance not found")
	}
	if inst.Alias != DefaultAlias {
		t.Fatalf("Alias = %q, want %q", inst.Alias, DefaultAlias)
	}
	if inst.BaseURL != "https://jira.example.test" {
		t.Fatalf("BaseURL = %q", inst.BaseURL)
	}
	if inst.Token != "secret" {
		t.Fatalf("Token = %q", inst.Token)
	}
	if inst.WorkerField != WorkerFieldKey {
		t.Fatalf("WorkerField = %q, want %q", inst.WorkerField, WorkerFieldKey)
	}
	if cfg.HTTPClientTimeout() != DefaultHTTPClientTimeout {
		t.Fatalf("HTTPClientTimeout = %s, want %s", cfg.HTTPClientTimeout(), DefaultHTTPClientTimeout)
	}
	if cfg.MaxResponseBytes() != DefaultMaxResponseBytes {
		t.Fatalf("MaxResponseBytes = %d, want %d", cfg.MaxResponseBytes(), DefaultMaxResponseBytes)
	}
}

func TestLoadMultipleInstances(t *testing.T) {
	cfg, err := Load([]string{
		"JIRA_INSTANCES=Corp,client",
		"JIRA_DEFAULT_INSTANCE=CLIENT",
		"JIRA_HTTP_CLIENT_TIMEOUT=45s",
		"JIRA_MAX_RESPONSE_BYTES=2097152",
		"JIRA_CORP_BASE_URL=https://corp-jira.example.test/",
		"JIRA_CORP_TOKEN=corp-secret",
		"JIRA_CORP_WORKER=corp-worker",
		"JIRA_CORP_WORKER_FIELD=name",
		"JIRA_CLIENT_BASE_URL=https://client-jira.example.test",
		"JIRA_CLIENT_TOKEN=client-secret",
		"JIRA_CLIENT_BILLABLE_BY_DEFAULT=true",
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	defaultInst, ok := cfg.Instance("")
	if !ok {
		t.Fatal("default instance not found")
	}
	if defaultInst.Alias != "client" {
		t.Fatalf("default alias = %q, want client", defaultInst.Alias)
	}
	if !defaultInst.BillableByDefault {
		t.Fatal("client BillableByDefault = false, want true")
	}
	if cfg.HTTPClientTimeout().String() != "45s" {
		t.Fatalf("HTTPClientTimeout = %s, want 45s", cfg.HTTPClientTimeout())
	}
	if cfg.MaxResponseBytes() != 2097152 {
		t.Fatalf("MaxResponseBytes = %d, want 2097152", cfg.MaxResponseBytes())
	}

	corp, ok := cfg.Instance("CORP")
	if !ok {
		t.Fatal("corp instance not found")
	}
	if corp.Alias != "Corp" {
		t.Fatalf("corp Alias = %q, want Corp", corp.Alias)
	}
	if corp.Worker != "corp-worker" {
		t.Fatalf("corp Worker = %q", corp.Worker)
	}
	if corp.WorkerField != WorkerFieldName {
		t.Fatalf("corp WorkerField = %q, want %q", corp.WorkerField, WorkerFieldName)
	}
}

func TestLoadReturnsErrorWhenDefaultIsMissing(t *testing.T) {
	_, err := Load([]string{
		"JIRA_INSTANCES=corp",
		"JIRA_DEFAULT_INSTANCE=missing",
		"JIRA_CORP_BASE_URL=https://corp-jira.example.test",
		"JIRA_CORP_TOKEN=corp-secret",
	})
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestLoadReturnsErrorWhenRequiredValueIsMissing(t *testing.T) {
	_, err := Load([]string{
		"JIRA_BASE_URL=https://jira.example.test",
	})
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}
