package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	tmpDir    string
	tmpDirMux sync.Mutex
)

//go:embed index.html
var indexHTML string

//go:embed static/css/terminal.css
var terminalCSS string

//go:embed static/js/terminal.js
var terminalJS string

func init() {
	var err error
	tmpDir, err = os.MkdirTemp("", "go-cli-*")
	if err != nil {
		log.Fatalf("Failed to create tmp dir: %v", err)
	}
	log.Printf("📁 Temp directory: %s", tmpDir)
	log.Printf("✅ index.html embedded (%d bytes)", len(indexHTML))
	log.Printf("✅ terminal.css embedded (%d bytes)", len(terminalCSS))
	log.Printf("✅ terminal.js embedded (%d bytes)", len(terminalJS))
}

func main() {
	port := getEnvOrDefault("PORT", "8080")

	// ── Serve index.html ───────────────────────────────────────
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Jika POST /run, handle execute
		if r.URL.Path == "/run" && r.Method == http.MethodPost {
			handleRun(w, r)
			return
		}

		// Favicon → serve inline SVG
		if r.URL.Path == "/favicon.ico" || r.URL.Path == "/favicon.png" {
			w.Header().Set("Content-Type", "image/svg+xml")
			w.Write([]byte(faviconSVG))
			return
		}

		// Serve index.html
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		fmt.Fprint(w, indexHTML)
	})

	// ── Serve CSS ──────────────────────────────────────────────
	http.HandleFunc("/static/css/terminal.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		fmt.Fprint(w, terminalCSS)
	})

	// ── Serve JS ───────────────────────────────────────────────
	http.HandleFunc("/static/js/terminal.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		fmt.Fprint(w, terminalJS)
	})

	// ── Handle code execution ──────────────────────────────────
	http.HandleFunc("/run", handleRun)

	log.Printf("🌟 Web CLI Simulator running at http://localhost:%s", port)
	log.Printf("💡 All static files embedded in binary — no external files needed")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// ============================================================
// HANDLE /run
// ============================================================
func handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Not Found — use POST to execute code", http.StatusNotFound)
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

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := executeCodeCtx(ctx, code)
	sendResponse(w, true, result.stdout, result.stderr)
}

type execResult struct {
	stdout string
	stderr string
}

func executeCodeCtx(ctx context.Context, code string) execResult {
	tmpDirMux.Lock()
	tmpFile, err := os.CreateTemp(tmpDir, "code-*.go")
	tmpDirMux.Unlock()
	if err != nil {
		return execResult{stderr: "Failed to create temp file: " + err.Error()}
	}
	filePath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(filePath)

	_, err = tmpFile.WriteString(code)
	if err != nil {
		return execResult{stderr: "Failed to write code: " + err.Error()}
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "go", "run", filePath)
	cmd.Dir = tmpDir
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return execResult{
			stdout: stdout.String(),
			stderr: "TIMEOUT: Execution exceeded 15 seconds",
		}
	}

	if err != nil {
		errStr := strings.TrimSpace(stderr.String())
		if errStr == "" {
			errStr = err.Error()
		}
		return execResult{stdout: stdout.String(), stderr: errStr}
	}

	return execResult{stdout: stdout.String(), stderr: strings.TrimSpace(stderr.String())}
}

// ============================================================
// RESPONSE HELPERS
// ============================================================
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

// ============================================================
// FAVICON SVG
// ============================================================
const faviconSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#3fb950"><polygon points="1,1 23,12 1,23"/></svg>`
