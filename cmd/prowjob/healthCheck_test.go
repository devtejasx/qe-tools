package prowjob

import (
	"strings"
	"testing"

	"github.com/konflux-ci/qe-tools/pkg/status"
	"github.com/konflux-ci/qe-tools/pkg/types"
	"github.com/spf13/viper"
	prowUtils "k8s.io/test-infra/prow/pod-utils/downwardapi"
)

func TestIsCriticalComponent(t *testing.T) {
	service := Service{
		Name:               "GitHub",
		CriticalComponents: []string{"Actions", "API Requests"},
	}

	tests := []struct {
		name      string
		component status.Component
		want      bool
	}{
		{
			name:      "component listed as critical",
			component: status.Component{Name: "Actions"},
			want:      true,
		},
		{
			name:      "another component listed as critical",
			component: status.Component{Name: "API Requests"},
			want:      true,
		},
		{
			name:      "component not listed as critical",
			component: status.Component{Name: "Packages"},
			want:      false,
		},
		{
			name:      "match is case sensitive",
			component: status.Component{Name: "actions"},
			want:      false,
		},
		{
			name:      "empty component name",
			component: status.Component{},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCriticalComponent(service, tt.component); got != tt.want {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestIsCriticalComponentWithoutCriticalComponents(t *testing.T) {
	service := Service{Name: "GitHub"}

	if isCriticalComponent(service, status.Component{Name: "Actions"}) {
		t.Error("a service declaring no critical components must not match anything")
	}
}

// withHealthCheckConfig swaps the package-level config that buildPRMessage
// reads, and restores it once the test is done.
func withHealthCheckConfig(t *testing.T, services []Service) {
	t.Helper()
	previous := healthCheckConfig
	healthCheckConfig = HealthCheckConfig{ExternalServices: services}
	t.Cleanup(func() { healthCheckConfig = previous })
}

func TestBuildPRMessage(t *testing.T) {
	withHealthCheckConfig(t, []Service{
		{Name: "GitHub", StatusPageURL: "https://www.githubstatus.com/api/v2/summary.json"},
		{Name: "Quay", StatusPageURL: "https://status.quay.io/api/v2/summary.json"},
	})

	hcStatus := &HealthCheckStatus{
		UnhealthyCriticalComponents: map[string][]string{
			"GitHub": {"Actions", "API Requests"},
		},
	}

	message := buildPRMessage(hcStatus, false)

	if !strings.Contains(message, "- GitHub: Actions, API Requests") {
		t.Errorf("expected the unhealthy components to be listed, got:\n%s", message)
	}
	// Only the status page of an affected service is worth pointing at.
	if !strings.Contains(message, "- https://www.githubstatus.com") {
		t.Errorf("expected the affected service status page, got:\n%s", message)
	}
	if strings.Contains(message, "status.quay.io") {
		t.Errorf("a healthy service status page must not be listed, got:\n%s", message)
	}
	// The URL is reduced to scheme and host, dropping the API path.
	if strings.Contains(message, "/api/v2/summary.json") {
		t.Errorf("expected only scheme and host, got:\n%s", message)
	}
	if !strings.Contains(message, "/retest-required") {
		t.Errorf("expected the retest instruction, got:\n%s", message)
	}
}

func TestBuildPRMessageConsequenceDependsOnFailIfUnhealthy(t *testing.T) {
	withHealthCheckConfig(t, []Service{
		{Name: "GitHub", StatusPageURL: "https://www.githubstatus.com/api/v2/summary.json"},
	})

	hcStatus := &HealthCheckStatus{
		UnhealthyCriticalComponents: map[string][]string{"GitHub": {"Actions"}},
	}

	failing := buildPRMessage(hcStatus, true)
	if !strings.Contains(failing, "E2E tests won") {
		t.Errorf("expected the blocking consequence, got:\n%s", failing)
	}

	warning := buildPRMessage(hcStatus, false)
	if !strings.Contains(warning, "E2E tests will probably fail") {
		t.Errorf("expected the non-blocking consequence, got:\n%s", warning)
	}
}

func TestBuildPRMessageWithMultipleUnhealthyServices(t *testing.T) {
	withHealthCheckConfig(t, []Service{
		{Name: "GitHub", StatusPageURL: "https://www.githubstatus.com/api/v2/summary.json"},
		{Name: "Quay", StatusPageURL: "https://status.quay.io/api/v2/summary.json"},
	})

	hcStatus := &HealthCheckStatus{
		UnhealthyCriticalComponents: map[string][]string{
			"GitHub": {"Actions"},
			"Quay":   {"Registry"},
		},
	}

	message := buildPRMessage(hcStatus, false)

	// The component lines are produced by ranging over a map, so assert on each
	// line rather than on a fixed order.
	for _, want := range []string{
		"- GitHub: Actions",
		"- Quay: Registry",
		"- https://www.githubstatus.com",
		"- https://status.quay.io",
	} {
		if !strings.Contains(message, want) {
			t.Errorf("expected %q in:\n%s", want, message)
		}
	}
}

func TestBuildPRMessageWithNoUnhealthyComponents(t *testing.T) {
	withHealthCheckConfig(t, []Service{
		{Name: "GitHub", StatusPageURL: "https://www.githubstatus.com/api/v2/summary.json"},
	})

	hcStatus := &HealthCheckStatus{UnhealthyCriticalComponents: map[string][]string{}}

	message := buildPRMessage(hcStatus, false)

	if strings.Contains(message, "githubstatus.com") {
		t.Errorf("no service is unhealthy, so no status page should be listed, got:\n%s", message)
	}
	if !strings.Contains(message, "/retest-required") {
		t.Errorf("expected the retest instruction, got:\n%s", message)
	}
}

func TestBuildPRMessageSkipsUnparsableStatusPageURL(t *testing.T) {
	withHealthCheckConfig(t, []Service{
		{Name: "Broken", StatusPageURL: "://not-a-url"},
		{Name: "GitHub", StatusPageURL: "https://www.githubstatus.com/api/v2/summary.json"},
	})

	hcStatus := &HealthCheckStatus{
		UnhealthyCriticalComponents: map[string][]string{
			"Broken": {"Everything"},
			"GitHub": {"Actions"},
		},
	}

	message := buildPRMessage(hcStatus, false)

	// A URL that cannot be parsed is logged and skipped; the remaining services
	// must still make it into the message.
	if !strings.Contains(message, "- https://www.githubstatus.com") {
		t.Errorf("expected the parsable status page to survive, got:\n%s", message)
	}
	if !strings.Contains(message, "- Broken: Everything") {
		t.Errorf("the component list is independent of URL parsing, got:\n%s", message)
	}
}

// withViperValue sets a viper key for the duration of a test.
func withViperValue(t *testing.T, key, value string) {
	t.Helper()
	previous := viper.Get(key)
	viper.Set(key, value)
	t.Cleanup(func() { viper.Set(key, previous) })
}

func TestHealthCheckNotifyRequiredEnvVars(t *testing.T) {
	// PreRunE names the missing variable in its error, so this list has to stay
	// in step with what the notify path actually reads.
	want := []string{
		types.GithubTokenEnv,
		prowUtils.RepoOwnerEnv,
		prowUtils.RepoNameEnv,
		prowUtils.PullNumberEnv,
	}

	if len(healthCheckNotifyRequiredEnvVars) != len(want) {
		t.Fatalf("expected %d required env vars, got %d", len(want), len(healthCheckNotifyRequiredEnvVars))
	}
	for i, name := range want {
		if healthCheckNotifyRequiredEnvVars[i] != name {
			t.Errorf("expected %q at index %d, got %q", name, i, healthCheckNotifyRequiredEnvVars[i])
		}
	}
}

func TestHealthCheckNotifyValidationReportsMissingEnvVar(t *testing.T) {
	// Mirrors the validation loop in PreRunE: with --notify-on-pr set, every
	// entry in healthCheckNotifyRequiredEnvVars must be present.
	withViperValue(t, notifyOnPRParamName, "true")
	for _, name := range healthCheckNotifyRequiredEnvVars {
		withViperValue(t, name, "set")
	}
	withViperValue(t, prowUtils.RepoNameEnv, "")

	var missing []string
	for _, e := range healthCheckNotifyRequiredEnvVars {
		if viper.GetString(e) == "" {
			missing = append(missing, e)
		}
	}

	if len(missing) != 1 || missing[0] != prowUtils.RepoNameEnv {
		t.Errorf("expected only %q to be reported missing, got %v", prowUtils.RepoNameEnv, missing)
	}
}
