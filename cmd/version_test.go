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
    "regexp"

    "github.com/open-policy-agent/opa/internal/report"
    "go.opentelemetry.io/otel"
    "sync"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
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
// TestOtlptraceVersion tests that the Version function in otlptrace returns the expected version string.
func TestOtlptraceVersion(t *testing.T) {
    got := otlptrace.Version()
    if got != "1.34.0" {
        t.Errorf("expected version %q, got %q", "1.34.0", got)
    }
}

// TestOtlptraceVersionFormat verifies that the Version string has a valid semantic version format.
func TestOtlptraceVersionFormat(t *testing.T) {
    ver := otlptrace.Version()
    if ver == "" {
        t.Fatal("expected non-empty version string")
    }
    matched, err := regexp.MatchString(`^\d+\.\d+\.\d+$`, ver)
    if err != nil {
        t.Fatalf("failed to compile regex: %v", err)
    }
    if !matched {
        t.Errorf("version %q does not match semantic version format", ver)
    }
}

// TestCheckOPAUpdateWhitespaceURL tests that when the OPA_TELEMETRY_SERVICE_URL environment variable
// is set to a string containing only whitespace, checkOPAUpdate does not return an error.
func TestCheckOPAUpdateWhitespaceURL(t *testing.T) {
    t.Setenv("OPA_TELEMETRY_SERVICE_URL", "   ")
    err := checkOPAUpdate(nil)
    if err != nil {
        t.Fatalf("expected no error for whitespace OPA_TELEMETRY_SERVICE_URL, got: %v", err)
    }
}
// TestCheckOPAUpdateWrongJSONType tests that checkOPAUpdate returns an error
// when the update endpoint returns a JSON value of the wrong type (e.g. a JSON array).
func TestCheckOPAUpdateWrongJSONType(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Return a JSON array instead of an object.
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(`[{"unexpected": "value"}]`))
    }))
    defer ts.Close()

    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)

    err := checkOPAUpdate(nil)
    if err == nil {
        t.Fatal("expected error due to wrong JSON type, got nil")
    }
// TestOtlptraceVersionConcurrency tests that otlptrace.Version can be called concurrently without issues.
func TestOtlptraceVersionConcurrency(t *testing.T) {
    const numGoroutines = 20
    var wg sync.WaitGroup
    errCh := make(chan error, numGoroutines)
    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            ver := otlptrace.Version()
            if ver != "1.34.0" {
                errCh <- fmt.Errorf("expected \"1.34.0\", got %q", ver)
            }
        }()
    }
    wg.Wait()
    close(errCh)
    for err := range errCh {
        t.Error(err)
    }
// TestCheckOPAUpdateMissingLatest tests that checkOPAUpdate works correctly when the update JSON is missing the "Latest" field.
func TestCheckOPAUpdateMissingLatest(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Header().Set("Content-Type", "application/json")
        // Return an empty JSON object (missing "Latest")
        _, _ = w.Write([]byte(`{}`))
    }))
    defer ts.Close()

    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)
    var update report.DataResponse
    err := checkOPAUpdate(&update)
    if err != nil {
        t.Fatalf("expected no error when response lacks Latest, got: %v", err)
    }
    if update.Latest.Download != "" || update.Latest.ReleaseNotes != "" || update.Latest.LatestRelease != "" {
        t.Errorf("expected empty Latest fields, got: %+v", update.Latest)
    }
}

// TestCheckOPAUpdateMalformedURL tests that checkOPAUpdate returns an error when provided with a malformed telemetry URL.
func TestCheckOPAUpdateMalformedURL(t *testing.T) {
    // Intentionally set a malformed URL.
    t.Setenv("OPA_TELEMETRY_SERVICE_URL", "http://%")
    err := checkOPAUpdate(nil)
    if err == nil {
        t.Fatal("expected error due to malformed URL, got nil")
    }
}
}

// TestGetTestServerCustomStatus tests that getTestServer returns a server responding with a custom status code.
func TestGetTestServerCustomStatus(t *testing.T) {
    // Create a dummy update response.
    dummyUpdate := map[string]string{"dummy": "value"}

    // Use custom status code 418 (I'm a teapot).
    baseURL, teardown := getTestServer(dummyUpdate, http.StatusTeapot)
    defer teardown()

    resp, err := http.Get(baseURL + "/v1/version")
    if err != nil {
        t.Fatalf("failed to get from test server: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusTeapot {
        t.Fatalf("expected status code %d, got %d", http.StatusTeapot, resp.StatusCode)
    }

    // Read the response body and verify the dummy update content.
    var result map[string]string
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        t.Fatalf("failed to decode JSON: %v", err)
    }
    if val, ok := result["dummy"]; !ok || val != "value" {
        t.Fatalf("expected dummy update value, got %v", result)
    }
}
}
// TestCheckOPAUpdateMethodNotAllowed tests that checkOPAUpdate returns an error when the update endpoint returns HTTP 405 (Method Not Allowed).
func TestCheckOPAUpdateMethodNotAllowed(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusMethodNotAllowed)
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"error": "method not allowed"}`))
    }))
    defer ts.Close()

    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)

    err := checkOPAUpdate(nil)
    if err == nil {
        t.Fatal("expected error due to 405 status code, got nil")
    }
}

// TestCheckOPAUpdateWrongContentType tests that checkOPAUpdate successfully parses JSON even when the Content-Type header is not "application/json".
func TestCheckOPAUpdateWrongContentType(t *testing.T) {
    expected := &report.DataResponse{Latest: report.ReleaseDetails{
        Download:      "http://example.com/download",
        ReleaseNotes:  "http://example.com/notes",
        LatestRelease: "v5.0.0",
    }}

    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        bs, _ := json.Marshal(expected)
        w.Header().Set("Content-Type", "text/plain")
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

    if updateData.Latest.LatestRelease != expected.Latest.LatestRelease {
        t.Errorf("expected LatestRelease %q, got %q", expected.Latest.LatestRelease, updateData.Latest.LatestRelease)
    }
}
// TestCheckOPAUpdateRedirect tests that checkOPAUpdate returns an error when encountering a redirect loop.
func TestCheckOPAUpdateRedirect(t *testing.T) {
    // Create a handler that always redirects to the same URL (creating a loop)
    var redirectURL string
    handler := func(w http.ResponseWriter, r *http.Request) {
        http.Redirect(w, r, redirectURL, http.StatusFound)
    }
    ts := httptest.NewServer(http.HandlerFunc(handler))
    redirectURL = ts.URL
    defer ts.Close()

    // Set the telemetry service URL to the test server's URL.
    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)

    // Call checkOPAUpdate and expect an error (due to redirect loop or exceeded redirects).
    err := checkOPAUpdate(nil)
    if err == nil {
        t.Fatal("expected error due to redirect loop, got nil")
    }
}
// TestCheckOPAUpdateTypeMismatch tests that checkOPAUpdate returns an error when the update response contains a type mismatch.
func TestCheckOPAUpdateTypeMismatch(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        // Note: "Download" is an integer instead of the expected string.
        _, _ = w.Write([]byte(`{"Latest": {"Download": 123, "ReleaseNotes": "http://example.com", "LatestRelease": "v1.1.1"}}`))
    }))
    defer ts.Close()
    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)

    var updateData report.DataResponse
    err := checkOPAUpdate(&updateData)
    if err == nil {
        t.Fatal("expected error due to type mismatch in update response, got nil")
    }
}

// TestCheckOPAUpdateNoContentType tests that checkOPAUpdate successfully parses JSON content even if the Content-Type header is missing.
func TestCheckOPAUpdateNoContentType(t *testing.T) {
    expected := &report.DataResponse{Latest: report.ReleaseDetails{
        Download:      "http://example.com/download",
        ReleaseNotes:  "http://example.com/notes",
        LatestRelease: "v6.0.0",
    }}
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        bs, _ := json.Marshal(expected)
        // Do not set any Content-Type header.
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write(bs)
    }))
    defer ts.Close()
    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)

    var updateData report.DataResponse
    err := checkOPAUpdate(&updateData)
    if err != nil {
        t.Fatalf("expected no error when Content-Type header is missing, got: %v", err)
    }
    if updateData.Latest.LatestRelease != expected.Latest.LatestRelease {
        t.Errorf("expected LatestRelease %q, got %q", expected.Latest.LatestRelease, updateData.Latest.LatestRelease)
    }
// TestGetTestServerInvalidPath tests that getTestServer returns 404 for paths not registered.
func TestGetTestServerInvalidPath(t *testing.T) {
    dummyUpdate := map[string]string{"test": "value"}
    baseURL, teardown := getTestServer(dummyUpdate, http.StatusOK)
    defer teardown()

    // Call an invalid path that is not registered.
    resp, err := http.Get(baseURL + "/invalid")
    if err != nil {
        t.Fatalf("unexpected error making GET request: %v", err)
    }
    defer resp.Body.Close()

    // Expect a 404 Not Found response for the invalid path.
    if resp.StatusCode != http.StatusNotFound {
        t.Fatalf("expected status code 404 for invalid path, got: %d", resp.StatusCode)
    }
}
// TestGenerateCmdOutputCustomBuildInfo verifies that when custom build-info environment variables
// are set, generateCmdOutput prints them correctly in its output.
func TestGenerateCmdOutputCustomBuildInfo(t *testing.T) {
    // Set custom build-info environment variables.
    t.Setenv("OPA_BUILD_COMMIT", "abc123")
    t.Setenv("OPA_BUILD_TIMESTAMP", "2023-10-01T00:00:00Z")
    t.Setenv("OPA_BUILD_HOSTNAME", "build-host")

    // Use a buffer to capture the output.
    var buf bytes.Buffer

    // Call generateCmdOutput with update check disabled.
    generateCmdOutput(&buf, false)

    output := buf.String()

    // Check that the custom build info appears in the output.
    if !strings.Contains(output, "abc123") {
        t.Errorf("expected build commit 'abc123' to appear in output, got: %q", output)
    }
    if !strings.Contains(output, "2023-10-01T00:00:00Z") {
        t.Errorf("expected build timestamp '2023-10-01T00:00:00Z' to appear in output, got: %q", output)
    }
    if !strings.Contains(output, "build-host") {
        t.Errorf("expected build hostname 'build-host' to appear in output, got: %q", output)
    }
}

// TestGenerateCmdOutputCustomUpdateKeys verifies that generateCmdOutput prints update keys with custom
// update data when a non-empty update is returned from the telemetry service.
func TestGenerateCmdOutputCustomUpdateKeys(t *testing.T) {
    // Create a custom update payload with additional values.
    exp := &report.DataResponse{Latest: report.ReleaseDetails{
        Download:      "https://custom.example.com/download",
        ReleaseNotes:  "https://custom.example.com/release-notes",
        LatestRelease: "v9.9.9",
    }}

    // Set up a test server that returns the custom update data.
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        bs, _ := json.Marshal(exp)
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write(bs)
    }))
    defer ts.Close()

    // Point the telemetry update URL env variable to our custom test server.
    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)

    var buf bytes.Buffer
    // Call generateCmdOutput with update check enabled.
    generateCmdOutput(&buf, true)

    output := buf.String()

    // Verify that the output includes the update-specific keys with the custom values.
    if !strings.Contains(output, "Latest Upstream Version:") || !strings.Contains(output, "v9.9.9") {
        t.Errorf("expected update version 'v9.9.9' in output, got: %q", output)
    }
    if !strings.Contains(output, "Download:") || !strings.Contains(output, "https://custom.example.com/download") {
        t.Errorf("expected custom download URL in output, got: %q", output)
    }
    if !strings.Contains(output, "Release Notes:") || !strings.Contains(output, "https://custom.example.com/release-notes") {
        t.Errorf("expected custom release notes URL in output, got: %q", output)
    }
}
}