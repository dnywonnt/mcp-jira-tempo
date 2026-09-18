package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		environ []string
		wantErr bool
		check   func(t *testing.T, cfg Config)
	}{
		{
			name: "default instance from generic env",
			environ: []string{
				"JIRA_BASE_URL=https://jira.example.test/",
				"JIRA_TOKEN=secret",
			},
			check: func(t *testing.T, cfg Config) {
				t.Helper()

				inst, ok := cfg.Instance("")
				require.True(t, ok)
				require.Equal(t, DefaultAlias, inst.Alias)
				require.Equal(t, "https://jira.example.test", inst.BaseURL)
				require.Equal(t, "secret", inst.Token)
				require.Equal(t, WorkerFieldKey, inst.WorkerField)
				require.Equal(t, DefaultHTTPClientTimeout, cfg.HTTPClientTimeout())
				require.Equal(t, DefaultMaxResponseBytes, cfg.MaxResponseBytes())
			},
		},
		{
			name: "multiple instances",
			environ: []string{
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
			},
			check: func(t *testing.T, cfg Config) {
				t.Helper()

				defaultInst, ok := cfg.Instance("")
				require.True(t, ok)
				require.Equal(t, "client", defaultInst.Alias)
				require.True(t, defaultInst.BillableByDefault)
				require.Equal(t, "45s", cfg.HTTPClientTimeout().String())
				require.EqualValues(t, 2097152, cfg.MaxResponseBytes())

				corp, ok := cfg.Instance("CORP")
				require.True(t, ok)
				require.Equal(t, "Corp", corp.Alias)
				require.Equal(t, "corp-worker", corp.Worker)
				require.Equal(t, WorkerFieldName, corp.WorkerField)
			},
		},
		{
			name: "default instance is missing",
			environ: []string{
				"JIRA_INSTANCES=corp",
				"JIRA_DEFAULT_INSTANCE=missing",
				"JIRA_CORP_BASE_URL=https://corp-jira.example.test",
				"JIRA_CORP_TOKEN=corp-secret",
			},
			wantErr: true,
		},
		{
			name: "required value is missing",
			environ: []string{
				"JIRA_BASE_URL=https://jira.example.test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(tt.environ)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			tt.check(t, cfg)
		})
	}
}
