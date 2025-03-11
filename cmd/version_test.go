// Copyright 2020 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package cmd

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "sort"
    "strings"
    "testing"
    "io"
    "fmt"

    "github.com/open-policy-agent/opa/internal/report"
    "go.opentelemetry.io/otel"
    "sync"
)

func TestGenerateCmdOutputDisableCheckFlag(t *testing.T) {
    var stdout bytes.Buffer

    generateCmdOutput(&stdout, false)

    expectOutputKeys(t, stdout.String(), []string{
    "Version",
    "Build Commit",
    "Build Timestamp",
    "Build Hostname",
    "Go Version",
    "Platform",
    "WebAssembly",
    "Rego Version",
    })
}

func TestGenerateCmdOutputWithCheckFlagNoError(t *testing.T) {
    exp := &report.DataResponse{Latest: report.ReleaseDetails{
    Download:      "https://openpolicyagent.org/downloads/v100.0.0/opa_darwin_amd64",
    ReleaseNotes:  "https://github.com/open-policy-agent/opa/releases/tag/v100.0.0",
    LatestRelease: "v100.0.0",
    }}

    // test server
    baseURL, teardown := getTestServer(exp, http.StatusOK)
    defer teardown()

    t.Setenv("OPA_TELEMETRY_SERVICE_URL", baseURL)

    var stdout bytes.Buffer

    generateCmdOutput(&stdout, true)

    expectOutputKeys(t, stdout.String(), []string{
    "Version",
    "Build Commit",
    "Build Timestamp",
    "Build Hostname",
    "Go Version",
    "Platform",
    "WebAssembly",
    "Latest Upstream Version",
    "Release Notes",
    "Download",
    "Rego Version",
    })
}

func TestCheckOPAUpdateBadURL(t *testing.T) {
    url := "http://foo:8112"
    t.Setenv("OPA_TELEMETRY_SERVICE_URL", url)

    err := checkOPAUpdate(nil)
    if err == nil {
    t.Fatal("Expected error but got nil")
    }
}

func expectOutputKeys(t *testing.T, stdout string, expectedKeys []string) {
    t.Helper()

    lines := strings.Split(strings.Trim(stdout, "\n"), "\n")
    gotKeys := make([]string, 0, len(lines))

    for _, line := range lines {
    gotKeys = append(gotKeys, strings.Split(line, ":")[0])
    }

    sort.Strings(expectedKeys)
    sort.Strings(gotKeys)

    if len(expectedKeys) != len(gotKeys) {
    t.Fatalf("expected %v but got %v", expectedKeys, gotKeys)
    }

    for i, got := range gotKeys {
    if expectedKeys[i] != got {
    t.Fatalf("expected %v but got %v", expectedKeys, gotKeys)
    }
    }
}

func getTestServer(update interface{}, statusCode int) (baseURL string, teardownFn func()) {
    mux := http.NewServeMux()
    ts := httptest.NewServer(mux)

    mux.HandleFunc("/v1/version", func(w http.ResponseWriter, _ *http.Request) {
    w.WriteHeader(statusCode)
    bs, _ := json.Marshal(update)
    w.Header().Set("Content-Type", "application/json")
    _, _ = w.Write(bs)
    })
    return ts.URL, ts.Close
}

func TestOtelVersion(t *testing.T) {
// TestGenerateCmdOutputWithCheckFlagAndError tests that when the update endpoint returns an error (HTTP 500),
// generateCmdOutput omits the update-specific keys from its output.
func TestGenerateCmdOutputWithCheckFlagAndError(t *testing.T) {
    // test server that returns HTTP 500 (error)
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusInternalServerError)
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"error": "internal error"}`))
    }))
    defer ts.Close()

    // Set the environment variable to point to our failing test server.
    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)

    var stdout bytes.Buffer

    // Call generateCmdOutput with the update check enabled.
    generateCmdOutput(&stdout, true)

    // Expect only the basic output keys (update keys are not printed if update check fails)
    expectOutputKeys(t, stdout.String(), []string{
        "Version",
        "Build Commit",
        "Build Timestamp",
        "Build Hostname",
        "Go Version",
        "Platform",
        "WebAssembly",
        "Rego Version",
    })
}

// TestCheckOPAUpdateEnvNotSet tests that when the OPA_TELEMETRY_SERVICE_URL environment variable
// is not set (or is empty), checkOPAUpdate does not return an error.
func TestCheckOPAUpdateEnvNotSet(t *testing.T) {
    // Set the telemetry service variable to an empty value.
    t.Setenv("OPA_TELEMETRY_SERVICE_URL", "")

    // Calling checkOPAUpdate should not produce an error due to missing telemetry URL.
    err := checkOPAUpdate(nil)
    if err != nil {
        t.Fatalf("expected no error when OPA_TELEMETRY_SERVICE_URL is unset, got: %v", err)
    }
}
    // Call the Version function from the OpenTelemetry package
    ver := otel.Version()
    if ver != "1.34.0" {
    t.Errorf("expected opentelemetry version to be %q, got %q", "1.34.0", ver)
    }
}

// TestCheckOPAUpdateInvalidJSON sets up a test HTTP server that returns invalid JSON,
// ensuring that checkOPAUpdate returns an error when the telemetry update endpoint returns bad data.
func TestCheckOPAUpdateInvalidJSON(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Header().Set("Content-Type", "application/json")
    // Write some invalid JSON data.
    _, _ = w.Write([]byte("invalid json"))
    }))
    defer ts.Close()

    // Use the new test server URL by updating the environment variable.
    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)

    err := checkOPAUpdate(nil)
    if err == nil {
    t.Fatal("expected error due to invalid JSON response, got nil")
    }
}

// TestCheckOPAUpdateSuccess tests that checkOPAUpdate successfully processes a valid update response and returns no error.
func TestCheckOPAUpdateSuccess(t *testing.T) {
    exp := &report.DataResponse{Latest: report.ReleaseDetails{
        Download:      "https://openpolicyagent.org/downloads/v1.0/opa_darwin_amd64",
        ReleaseNotes:  "https://github.com/open-policy-agent/opa/releases/tag/v1.0",
        LatestRelease: "v1.0.0",
    }}

    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        bs, err := json.Marshal(exp)
        if err != nil {
            t.Fatalf("failed to marshal exp: %v", err)
        }
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write(bs)
    }))
    defer ts.Close()

    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)

    err := checkOPAUpdate(nil)
    if err != nil {
        t.Fatalf("expected no error, got: %v", err)
    }
}

// TestGenerateCmdOutputWithEmptyUpdate tests that generateCmdOutput prints update output keys even when update details are empty.
func TestGenerateCmdOutputWithEmptyUpdate(t *testing.T) {
    // Setup a test server that returns an empty JSON object.
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("{}"))
    }))
    defer ts.Close()

    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)

    var stdout bytes.Buffer

    // Call generateCmdOutput with update check enabled.
    generateCmdOutput(&stdout, true)

    // Even if update details are empty, the update keys should be printed along with the basic keys.
    expectOutputKeys(t, stdout.String(), []string{
        "Version",
        "Build Commit",
        "Build Timestamp",
        "Build Hostname",
        "Go Version",
        "Platform",
        "WebAssembly",
        "Latest Upstream Version",
        "Release Notes",
        "Download",
        "Rego Version",
    })
}
// TestCheckOPAUpdateNotFound tests that checkOPAUpdate returns an error when the update endpoint returns a 404 status.
func TestCheckOPAUpdateNotFound(t *testing.T) {
    // Setup a test server that returns 404 for the /v1/version endpoint.
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusNotFound)
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"error": "not found"}`))
    }))
    defer ts.Close()

    // Set the telemetry service URL to point to our failing test server.
    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)

    // Invoke checkOPAUpdate and confirm that an error is returned.
    err := checkOPAUpdate(nil)
    if err == nil {
        t.Fatal("expected error due to 404 status code, got nil")
    }
}

// TestGetTestServer tests the getTestServer helper to ensure it starts a server that correctly responds
// with the provided update content.
func TestGetTestServer(t *testing.T) {
    expectedUpdate := &report.DataResponse{
        Latest: report.ReleaseDetails{
            Download:      "https://example.com/download",
            ReleaseNotes:  "https://example.com/release-notes",
            LatestRelease: "v0.0.1",
        },
    }

    baseURL, teardown := getTestServer(expectedUpdate, http.StatusOK)
    defer teardown()

    resp, err := http.Get(baseURL + "/v1/version")
    if err != nil {
        t.Fatalf("unexpected error making GET request: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        t.Fatalf("expected status code 200, got: %d", resp.StatusCode)
    }

    var result report.DataResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        t.Fatalf("failed to decode JSON response: %v", err)
    }

    if result.Latest.LatestRelease != expectedUpdate.Latest.LatestRelease {
        t.Fatalf("expected latest release %q, got %q", expectedUpdate.Latest.LatestRelease, result.Latest.LatestRelease)
    }
}
// TestGenerateCmdOutputFaultyWriter tests generateCmdOutput with a faulty writer that always returns an error.
type faultyWriter struct{}

func (f *faultyWriter) Write(p []byte) (int, error) {
    return 0, fmt.Errorf("write error")
}

func TestGenerateCmdOutputFaultyWriter(t *testing.T) {
    // Use a faulty writer that returns error on Write.
    fw := &faultyWriter{}

    // Recover from any panic to ensure generateCmdOutput handles (or ignores) the write error.
    defer func() {
        if r := recover(); r != nil {
            t.Errorf("generateCmdOutput panicked with faulty writer: %v", r)
        }
    }()

    // Call generateCmdOutput with update check disabled (or enabled; the behavior is not guaranteed)
    generateCmdOutput(fw, false)
}
// TestGenerateCmdOutputFormat tests that generateCmdOutput outputs key-value pairs
// formatted as "Key: Value", ensuring both parts are non-empty.
func TestGenerateCmdOutputFormat(t *testing.T) {
    var buf bytes.Buffer

    // Call generateCmdOutput with update check disabled.
    generateCmdOutput(&buf, false)

    out := buf.String()
    if len(strings.TrimSpace(out)) == 0 {
        t.Fatal("expected non-empty output from generateCmdOutput")
    }

    // Verify each output line contains a colon separating a key from a value.
    lines := strings.Split(strings.TrimSpace(out), "\n")
    for _, line := range lines {
        parts := strings.SplitN(line, ":", 2)
        if len(parts) != 2 {
            t.Errorf("line %q does not contain a colon to separate key and value", line)
            continue
        }

        key := strings.TrimSpace(parts[0])
        val := strings.TrimSpace(parts[1])

        if key == "" {
            t.Errorf("found an empty key in line: %q", line)
        }
        if val == "" {
            t.Errorf("found an empty value in line: %q", line)
        }
    }
}

// TestCheckOPAUpdatePopulateData tests that checkOPAUpdate populates update data when a non-nil pointer is provided.
// It sets up a test server that returns valid update JSON and verifies that the pointer passed is updated.
func TestCheckOPAUpdatePopulateData(t *testing.T) {
    // Expected update response.
    exp := &report.DataResponse{Latest: report.ReleaseDetails{
        Download:      "https://example.com/download",
        ReleaseNotes:  "https://example.com/release-notes",
        LatestRelease: "v2.0.0",
    }}

    // Set up a test server that returns the expected update response.
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        bs, err := json.Marshal(exp)
        if err != nil {
            t.Fatalf("failed to marshal update: %v", err)
        }
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write(bs)
    }))
    defer ts.Close()

    // Set the telemetry service URL to point to our test server.
    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)

    // Declare a variable to be populated by checkOPAUpdate.
    var updateData report.DataResponse

    // Call checkOPAUpdate with a non-nil pointer. Expect no error.
    err := checkOPAUpdate(&updateData)
    if err != nil {
        t.Fatalf("expected no error, got: %v", err)
    }

    // Verify that the update data was populated as expected.
    if updateData.Latest.LatestRelease != exp.Latest.LatestRelease {
        t.Errorf("expected LatestRelease %q, got %q", exp.Latest.LatestRelease, updateData.Latest.LatestRelease)
    }
}
// TestGenerateCmdOutputNilWriter tests that generateCmdOutput panics when provided a nil writer.
func TestGenerateCmdOutputNilWriter(t *testing.T) {
    defer func() {
        if r := recover(); r == nil {
            t.Errorf("expected panic with nil writer, but did not panic")
        }
    }()
    // Calling generateCmdOutput with a nil writer should cause a panic.
    generateCmdOutput(nil, false)
}
// TestGenerateCmdOutputConcurrency tests concurrent calls to generateCmdOutput to verify thread-safety.
func TestGenerateCmdOutputConcurrency(t *testing.T) {
    const numGoroutines = 10
    var wg sync.WaitGroup
    errCh := make(chan error, numGoroutines)

    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            var buf bytes.Buffer
            generateCmdOutput(&buf, false)
            output := buf.String()
            if strings.TrimSpace(output) == "" {
                errCh <- fmt.Errorf("output is empty")
            }
            // Ensure the basic "Version:" key is present in the output.
            if !strings.Contains(output, "Version:") {
                errCh <- fmt.Errorf("output missing 'Version:' key")
            }
        }()
    }
    wg.Wait()
    close(errCh)
    for err := range errCh {
        t.Error(err)
    }
}

// TestCheckOPAUpdateConcurrent tests concurrent calls to checkOPAUpdate to verify there are no race conditions.
func TestCheckOPAUpdateConcurrent(t *testing.T) {
    // Create a test server that returns a valid update response.
    exp := &report.DataResponse{Latest: report.ReleaseDetails{
        Download:      "https://example.com/download",
        ReleaseNotes:  "https://example.com/release-notes",
        LatestRelease: "v3.0.0",
    }}
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        bs, err := json.Marshal(exp)
        if err != nil {
            t.Fatalf("failed to marshal update: %v", err)
        }
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write(bs)
    }))
    defer ts.Close()

    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)

    const numGoroutines = 10
    var wg sync.WaitGroup
    errCh := make(chan error, numGoroutines)

    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            // Pass in a dummy update variable to be populated.
            var update report.DataResponse
            if err := checkOPAUpdate(&update); err != nil {
                errCh <- err
            }
        }()
    }
    wg.Wait()
    close(errCh)
    for err := range errCh {
        t.Error(err)
    }
}
// TestCheckOPAUpdateEmptyBody tests that checkOPAUpdate returns an error when the response body is empty.
func TestCheckOPAUpdateEmptyBody(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        // No body is written to simulate an empty response.
    }))
    defer ts.Close()

    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)
    err := checkOPAUpdate(nil)
    if err == nil {
        t.Fatal("expected error due to empty response body, got nil")
    }
}

// TestCheckOPAUpdateExtraFields tests that checkOPAUpdate properly parses update responses with extra, unexpected fields.
func TestCheckOPAUpdateExtraFields(t *testing.T) {
    exp := map[string]interface{}{
        "Latest": map[string]interface{}{
            "Download":      "https://example.com/download",
            "ReleaseNotes":  "https://example.com/release-notes",
            "LatestRelease": "v4.0.0",
            "ExtraField":    "extra value",
        },
        "Unexpected": "foo",
    }

    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        bs, err := json.Marshal(exp)
        if err != nil {
            t.Fatalf("failed to marshal expected response: %v", err)
        }
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write(bs)
    }))
    defer ts.Close()

    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)

    var updateData report.DataResponse
    err := checkOPAUpdate(&updateData)
    if err != nil {
        t.Fatalf("expected no error, got: %v", err)
    }
    if updateData.Latest.LatestRelease != "v4.0.0" {
        t.Errorf("expected LatestRelease 'v4.0.0', got: %q", updateData.Latest.LatestRelease)
    }
}
