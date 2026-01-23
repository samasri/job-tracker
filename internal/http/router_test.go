package http_test

import (
	"testing"

	"jobtracker/internal/testharness"
)

func TestHealthEndpoint(t *testing.T) {
	env := testharness.NewTestEnv(t)

	resp := env.Get("/health")
	env.AssertStatus(resp, 200)

	var result map[string]string
	env.ReadJSON(resp, &result)

	if result["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", result["status"])
	}
}
