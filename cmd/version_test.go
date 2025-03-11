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
    "strconv"
    "strings"
    "testing"
    "io"
    "github.com/open-policy-agent/opa/internal/report"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
    "errors"
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


// TestOtelVersion verifies that the otel.Version() function returns the expected version.
func TestOtelVersion(t *testing.T) {
    v := otel.Version()
    expected := "1.34.0"
    if v != expected {
        t.Fatalf("expected version %q, got %q", expected, v)
    }
}

// TestOtlptraceVersion verifies that the otlptrace.Version() function returns the correct version string
func TestOtlptraceVersion(t *testing.T) {
    // Get the version from the otlptrace package
    version := otlptrace.Version()
    
    // The expected version should be "1.34.0" as defined in the source file
    expected := "1.34.0"
    
    // Verify the version matches the expected value
    if version != expected {
        t.Fatalf("expected otlptrace version %q, got %q", expected, version)
    }
}

// TestOtlptraceVersionConsistency verifies that the version reported by otlptrace.Version() 
// is consistent across multiple calls
func TestOtlptraceVersionConsistency(t *testing.T) {
    // Call Version() multiple times and ensure the result is consistent
    version1 := otlptrace.Version()
    version2 := otlptrace.Version()
    version3 := otlptrace.Version()
    
    // All three calls should return the same value
    if version1 != version2 || version2 != version3 {
        t.Fatalf("inconsistent version values returned: %q, %q, %q", 
                    version1, version2, version3)
    }
}

// TestOtlptraceVersionFormat verifies that the version string follows semantic versioning format
func TestOtlptraceVersionFormat(t *testing.T) {
    version := otlptrace.Version()
    
    // Basic pattern check for semantic versioning (MAJOR.MINOR.PATCH)
    matched := len(strings.Split(version, ".")) == 3
    
    if !matched {
        t.Fatalf("version string %q does not follow semantic versioning format (MAJOR.MINOR.PATCH)", 
                    version)
    }
    
    // Validate that each part is a number
    parts := strings.Split(version, ".")
    for i, part := range parts {
        _, err := strconv.Atoi(part)
        if err != nil {
            t.Fatalf("version component %d (%q) is not a valid integer", i+1, part)
        }
    }
}

// TestOtlptraceVersionIntegration verifies that the otlptrace version is properly 
// integrated in the command output
func TestOtlptraceVersionIntegration(t *testing.T) {
    var buf bytes.Buffer
    
    // Generate command output to the buffer
    generateCmdOutput(&buf, false)
    
    // Get the output as a string
    output := buf.String()
    
    // Check that the output contains the otlptrace.Version()
    if !strings.Contains(output, otlptrace.Version()) {
        t.Errorf("command output does not contain otlptrace version %q", otlptrace.Version())
    }
}
}

// TestCheckOPAUpdateMalformedURL verifies that checkOPAUpdate returns an error when the telemetry URL is malformed.
func TestCheckOPAUpdateMalformedURL(t *testing.T) {
    // Set a malformed telemetry URL.
    t.Setenv("OPA_TELEMETRY_SERVICE_URL", "::::")

    err := checkOPAUpdate(nil)
    if err == nil {
        t.Fatal("expected error for malformed URL, got nil")
    }
// TestCheckOPAUpdateNonOKResponse verifies that checkOPAUpdate returns an error when the telemetry service returns a non-OK HTTP status.
func TestCheckOPAUpdateNonOKResponse(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusInternalServerError)
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"error": "server error"}`))
    }))
    defer ts.Close()

    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)
    err := checkOPAUpdate(nil)
    if err == nil {
        t.Fatal("expected error on non-OK response, got nil")
    }
}

// TestGetTestServer verifies that getTestServer returns a server that responds with the provided update data and status code.
func TestGetTestServer(t *testing.T) {
    expected := map[string]interface{}{"foo": "bar"}
    baseURL, teardown := getTestServer(expected, http.StatusCreated)
    defer teardown()

    resp, err := http.Get(baseURL + "/v1/version")
    if err != nil {
        t.Fatalf("failed to GET from test server: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        t.Fatalf("expected status code %d, got %d", http.StatusCreated, resp.StatusCode)
    }

    var got map[string]interface{}
    err = json.NewDecoder(resp.Body).Decode(&got)
    if err != nil {
        t.Fatalf("failed to decode response: %v", err)
    }

    if got["foo"] != "bar" {
        t.Fatalf("expected key 'foo' with value 'bar', got %v", got["foo"])
    }
}
// TestCheckOPAUpdateSuccess verifies that checkOPAUpdate successfully decodes valid telemetry data.
func TestCheckOPAUpdateSuccess(t *testing.T) {
    // expected update details
    expectedUpdate := &report.DataResponse{Latest: report.ReleaseDetails{
        Download:      "https://example.com/download",
        ReleaseNotes:  "https://example.com/release",
        LatestRelease: "v1.2.3",
    }}

    // create a test server that returns the expected update in JSON
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        responseBytes, _ := json.Marshal(expectedUpdate)
        w.Header().Set("Content-Type", "application/json")
        w.Write(responseBytes)
    }))
    defer ts.Close()

    // set the telemetry URL to the test server's URL
    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)

// TestGenerateCmdOutputWithCheckFlagPartialData verifies that even if some telemetry update fields are missing,
// generateCmdOutput still outputs the extended keys.
func TestGenerateCmdOutputWithCheckFlagPartialData(t *testing.T) {
    // Create a partial update: only LatestRelease is provided (Download and ReleaseNotes are omitted).
    partial := &report.DataResponse{Latest: report.ReleaseDetails{
        LatestRelease: "v2.0.0",
    }}

    // Create a test server that returns the partial update in JSON.
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        bs, _ := json.Marshal(partial)
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write(bs)
    }))
    defer ts.Close()

    // Set the telemetry URL to the test server's URL so that generateCmdOutput uses it.
    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)

    var stdout bytes.Buffer

    // Even with a partial update response, generateCmdOutput should output the extended set of keys.
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
    var update report.DataResponse
    err := checkOPAUpdate(&update)
    if err != nil {
        t.Fatalf("expected no error, got: %v", err)
    }

    if update.Latest.LatestRelease != "v1.2.3" {
        t.Fatalf("expected LatestRelease %q, got %q", "v1.2.3", update.Latest.LatestRelease)
    }
}
// TestGenerateCmdOutputWriterError verifies that generateCmdOutput does not panic even if the writer fails.
func TestGenerateCmdOutputWriterError(t *testing.T) {
    // errorWriter always returns an error on Write.
    type errorWriter struct{}
    func (e *errorWriter) Write(p []byte) (n int, err error) {
    return 0, errors.New("write error")
    }

    ew := &errorWriter{}
    // Ensure that generateCmdOutput does not panic with an erroring writer.
    defer func() {
    if r := recover(); r != nil {
    t.Fatalf("generateCmdOutput panicked: %v", r)
    }
    }()

    // We call generateCmdOutput with the error writer.
    // The check flag value is arbitrary here.
    generateCmdOutput(ew, true)
}

// TestCheckOPAUpdateEmptyBody verifies that checkOPAUpdate returns an error when the telemetry service returns a 200 response with an empty body.
func TestCheckOPAUpdateEmptyBody(t *testing.T) {
    // Create a test server that returns a 200 status and an empty response body.
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    // Do not write any response bytes.
    }))
    defer ts.Close()

    // Set the telemetry URL to point to our test server.
    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)

    var update report.DataResponse
    err := checkOPAUpdate(&update)
    if err == nil {
    t.Fatal("expected an error when server returns an empty body, got nil")
    }
}
// TestGenerateCmdOutputWithCheckFlagExtraFields verifies that generateCmdOutput handles telemetry responses with extra fields gracefully.
func TestGenerateCmdOutputWithCheckFlagExtraFields(t *testing.T) {
    extraData := map[string]interface{}{
        "foo": "bar",
        "Latest": map[string]interface{}{
            "LatestRelease": "v3.0.0",
            "Download":      "https://example.com/v3/download",
            "ReleaseNotes":  "https://example.com/v3/notes",
            "ExtraField":    "extra",
        },
    }
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        bs, _ := json.Marshal(extraData)
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write(bs)
    }))
    defer ts.Close()

    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)
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
// TestGetTestServerNilUpdate verifies that getTestServer correctly returns "null" when update is nil.
func TestGetTestServerNilUpdate(t *testing.T) {
    baseURL, teardown := getTestServer(nil, http.StatusOK)
    defer teardown()

    resp, err := http.Get(baseURL + "/v1/version")
    if err != nil {
        t.Fatalf("failed to GET from test server: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        t.Fatalf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        t.Fatalf("failed to read response body: %v", err)
    }

    if strings.TrimSpace(string(body)) != "null" {
        t.Fatalf("expected response body 'null', got %q", string(body))
    }
}

// TestCheckOPAUpdateInvalidType verifies that checkOPAUpdate returns an error when the JSON response is not an object.
func TestCheckOPAUpdateInvalidType(t *testing.T) {
    // Create a test server that returns a JSON array instead of an object.
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`[1,2,3]`))
    }))
    defer ts.Close()

    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)

    var update report.DataResponse
    err := checkOPAUpdate(&update)
    if err == nil {
        t.Fatal("expected error when JSON structure does not match expected object, got nil")
    }
}
// TestGenerateCmdOutputContent verifies that generateCmdOutput outputs the expected version information.
func TestGenerateCmdOutputContent(t *testing.T) {
    var buf bytes.Buffer

    // Call generateCmdOutput with the check flag disabled so that only basic keys are output.
    generateCmdOutput(&buf, false)

    output := buf.String()

    // Check that the output contains the "Version:" key.
    if !strings.Contains(output, "Version:") {
        t.Errorf("expected output to contain 'Version:', got %q", output)
    }

    // Check that the otel.Version() returned string ("1.34.0") is in the output.
    if !strings.Contains(output, otel.Version()) {
        t.Errorf("expected output to contain %q, got %q", otel.Version(), output)
    }
}
// TestCheckOPAUpdateEmptyJSON verifies that checkOPAUpdate behaves correctly when the telemetry service returns an empty JSON object.
func TestCheckOPAUpdateEmptyJSON(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte("{}"))
    }))
    defer ts.Close()

    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)
    var update report.DataResponse
    err := checkOPAUpdate(&update)
    if err != nil {
        t.Fatalf("expected no error, got: %v", err)
    }

    if update.Latest.LatestRelease != "" {
        t.Fatalf("expected LatestRelease to be empty, got %q", update.Latest.LatestRelease)
    }
}

// TestGenerateCmdOutputWithCheckFlagEmptyJSON verifies that generateCmdOutput outputs only basic keys
// when the telemetry service returns an empty JSON object.
func TestGenerateCmdOutputWithCheckFlagEmptyJSON(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte("{}"))
    }))
    defer ts.Close()

    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)
    var buf bytes.Buffer

    generateCmdOutput(&buf, true)

    expectOutputKeys(t, buf.String(), []string{
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
// TestCheckOPAUpdateTextPlainContentType verifies that checkOPAUpdate successfully decodes valid telemetry data
// even when the Content-Type header is "text/plain" instead of "application/json".
func TestCheckOPAUpdateTextPlainContentType(t *testing.T) {
    expectedUpdate := &report.DataResponse{Latest: report.ReleaseDetails{
        Download:      "https://example.com/download",
        ReleaseNotes:  "https://example.com/release",
        LatestRelease: "v1.0.0",
    }}

    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        responseBytes, _ := json.Marshal(expectedUpdate)
        w.Header().Set("Content-Type", "text/plain")
        _, _ = w.Write(responseBytes)
    }))
    defer ts.Close()

    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)

    var update report.DataResponse
    err := checkOPAUpdate(&update)
    if err != nil {
        t.Fatalf("expected no error, got: %v", err)
    }

    if update.Latest.LatestRelease != "v1.0.0" {
        t.Fatalf("expected LatestRelease %q, got %q", "v1.0.0", update.Latest.LatestRelease)
    }
}

// TestGenerateCmdOutputWithoutTelemetryEnv verifies that generateCmdOutput outputs only the basic keys
// when the OPA_TELEMETRY_SERVICE_URL environment variable is not set.
func TestGenerateCmdOutputWithoutTelemetryEnv(t *testing.T) {
    // Unset the telemetry service URL by setting it to an empty string.
    t.Setenv("OPA_TELEMETRY_SERVICE_URL", "")

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
        "Rego Version",
    })
}
// TestGenerateCmdOutputLineFormat verifies that generateCmdOutput outputs lines in "Key: Value" format.
func TestGenerateCmdOutputLineFormat(t *testing.T) {
    var buf bytes.Buffer
    generateCmdOutput(&buf, false)
    out := buf.String()
    lines := strings.Split(strings.TrimSpace(out), "\n")
    for _, line := range lines {
        if !strings.Contains(line, ":") {
            t.Errorf("line %q does not contain a colon", line)
        }
        parts := strings.SplitN(line, ":", 2)
        if strings.TrimSpace(parts[0]) == "" {
            t.Errorf("line %q has empty key", line)
        }
    }
}

// TestCheckOPAUpdateMissingContentType verifies that checkOPAUpdate can decode JSON responses even when the Content-Type header is missing.
func TestCheckOPAUpdateMissingContentType(t *testing.T) {
    expectedUpdate := &report.DataResponse{Latest: report.ReleaseDetails{
        Download:      "https://example.com/missing",
        ReleaseNotes:  "https://example.com/missing_notes",
        LatestRelease: "vMissing",
    }}

    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        responseBytes, _ := json.Marshal(expectedUpdate)
        // Do not set Content-Type header
        _, _ = w.Write(responseBytes)
    }))
    defer ts.Close()

    t.Setenv("OPA_TELEMETRY_SERVICE_URL", ts.URL)

    var update report.DataResponse
    err := checkOPAUpdate(&update)
    if err != nil {
        t.Fatalf("expected no error, got: %v", err)
    }

    if update.Latest.LatestRelease != "vMissing" {
        t.Fatalf("expected LatestRelease %q, got %q", "vMissing", update.Latest.LatestRelease)
    }
}
// TestGenerateCmdOutputNilWriter verifies that generateCmdOutput panics when a nil writer is provided.
func TestGenerateCmdOutputNilWriter(t *testing.T) {
    defer func() {
        if r := recover(); r == nil {
            t.Error("expected panic when writer is nil, but did not panic")
        }
    }()
    // Calling generateCmdOutput with nil should lead to a panic.
    generateCmdOutput(nil, false)
}
}