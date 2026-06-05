package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"context"
	"os"
	"os/exec"
	// "path/filepath"
	// "runtime"
	"strings"
	"sync"
	"time"
)

var (
	tmpDir    string
	tmpDirMux sync.Mutex
)

func init() {
	var err error
	tmpDir, err = os.MkdirTemp("", "go-cli-*")
	if err != nil {
		log.Fatalf("Failed to create tmp dir: %v", err)
	}
}

func main() {
	// Serve static files
	staticFS := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", staticFS))

	// Serve index.html
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	// Handle code execution
	http.HandleFunc("/run", handleRun)

	port := getEnvOrDefault("PORT", "8080")
	log.Printf("🌟 Web CLI Simulator running at http://localhost:%s", port)
	log.Printf("📁 Temp directory: %s", tmpDir)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendResponse(w, false, "", "Failed to read request body")
		return
	}
	defer r.Body.Close()

	code := string(body)
	if strings.TrimSpace(code) == "" {
		sendResponse(w, false, "", "No code provided")
		return
	}

	// Inject timeout handling - wrap code in a goroutine with select
	// so it can be killed after timeout
	wrappedCode := injectTimeout(code, 10*time.Second)

	result := executeCode(wrappedCode)
	sendResponse(w, true, result.stdout, result.stderr)
}

func injectTimeout(code string, timeout time.Duration) string {
	// We'll write the code directly and rely on process timeout
	// The execution function will handle killing the process
	return code
}

type execResult struct {
	stdout string
	stderr string
}

func executeCode(code string) execResult {
	tmpDirMux.Lock()
	tmpFile, err := os.CreateTemp(tmpDir, "code-*.go")
	tmpDirMux.Unlock()
	if err != nil {
		return execResult{stderr: "Failed to create temp file: " + err.Error()}
	}
	defer os.Remove(tmpFile.Name())

	// Write code to temp file
	_, err = tmpFile.WriteString(code)
	tmpFile.Close()
	if err != nil {
		return execResult{stderr: "Failed to write code: " + err.Error()}
	}

	// Execute with timeout using context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "go", "run", tmpFile.Name())
	cmd.Dir = tmpDir
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return execResult{
			stdout: stdout.String(),
			stderr: "TIMEOUT: Code execution exceeded 10 seconds",
		}
	}

	if err != nil {
		// If there's still output, include it with the error context
		out := stdout.String()
		errOut := stderr.String()
		if errOut == "" {
			errOut = err.Error()
		}
		return execResult{stdout: out, stderr: errOut}
	}

	return execResult{stdout: stdout.String(), stderr: stderr.String()}
}

func sendResponse(w http.ResponseWriter, success bool, stdout, stderr string) {
	w.Header().Set("Content-Type", "application/json")
	resp := fmt.Sprintf(`{"success":%v,"stdout":%s,"stderr":%s}`,
		success,
		escapeJSON(stdout),
		escapeJSON(stderr))
	fmt.Fprint(w, resp)
}

func escapeJSON(s string) string {
	var buf bytes.Buffer
	for _, r := range s {
		switch {
		case r == '"':
			buf.WriteString(`\"`)
		case r == '\\':
			buf.WriteString(`\\`)
		case r == '\n':
			buf.WriteString(`\n`)
		case r == '\r':
			buf.WriteString(`\r`)
		case r == '\t':
			buf.WriteString(`\t`)
		case r < ' ' || r == 127:
			// skip control chars
		default:
			buf.WriteRune(r)
		}
	}
	return fmt.Sprintf(`"%s"`, buf.String())
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
