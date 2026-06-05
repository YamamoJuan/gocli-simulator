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
	// Redirect ke stdout agar Vercel kategorikan sebagai info, bukan error
	log.SetOutput(os.Stdout)

	var err error
	tmpDir, err = os.MkdirTemp("", "go-cli-*")
	if err != nil {
		log.Fatalf("[INIT] Failed to create tmp dir: %v", err)
	}
	log.Printf("[INIT] Temp directory: %s", tmpDir)
	log.Printf("[INIT] index.html embedded: %d bytes", len(indexHTML))
	log.Printf("[INIT] terminal.css embedded: %d bytes", len(terminalCSS))
	log.Printf("[INIT] terminal.js embedded: %d bytes", len(terminalJS))
}

func main() {
	port := getEnvOrDefault("PORT", "8080")
	log.SetPrefix(fmt.Sprintf("[%s] ", port))

	mux := http.NewServeMux()

	// ── Root: GET / → index.html ──────────────────────────────
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[HANDLER] %s %s", r.Method, r.URL.Path)

		// Favicon
		if r.URL.Path == "/favicon.ico" || r.URL.Path == "/favicon.png" {
			w.Header().Set("Content-Type", "image/svg+xml")
			w.Write([]byte(faviconSVG))
			return
		}

		// Semua GET request lainnya → index.html
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		fmt.Fprint(w, indexHTML)
	})

	// ── Static CSS ─────────────────────────────────────────────
	mux.HandleFunc("/static/css/terminal.css", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[HANDLER] %s %s", r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		fmt.Fprint(w, terminalCSS)
	})

	// ── Static JS ─────────────────────────────────────────────
	mux.HandleFunc("/static/js/terminal.js", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[HANDLER] %s %s", r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		fmt.Fprint(w, terminalJS)
	})

	// ── Run: POST /run ────────────────────────────────────────
	mux.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[HANDLER] %s %s", r.Method, r.URL.Path)
		handleRun(w, r)
	})

	log.Printf("[SERVER] Go CLI Simulator starting on port %s", port)
	log.Printf("[SERVER] Open http://localhost:%s", port)
	log.Printf("[SERVER] All static files embedded in binary")

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("[SERVER] ListenAndServe error: %v", err)
	}
}

// ============================================================
// HANDLE /run
// ============================================================
func handleRun(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[ERROR] handleRun: failed to read body: %v", err)
		sendResponse(w, false, "", "Failed to read request body")
		return
	}
	defer r.Body.Close()

	code := string(body)
	if strings.TrimSpace(code) == "" {
		sendResponse(w, false, "", "No code provided")
		return
	}

	log.Printf("[EXEC] Starting (%d bytes, timeout 15s)", len(code))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := executeCodeCtx(ctx, code)

	if result.stderr != "" {
		if result.stderr == "TIMEOUT: Execution exceeded 15 seconds" {
			log.Printf("[EXEC] Result: TIMEOUT")
		} else {
			log.Printf("[EXEC] Result: ERROR — %s", truncate(result.stderr, 80))
		}
	} else {
		log.Printf("[EXEC] Result: OK — %s", truncate(result.stdout, 80))
	}

	sendResponse(w, true, result.stdout, result.stderr)
}

type execResult struct {
	stdout string
	stderr string
}

// ============================================================
// WRITE → CLOSE → DEFER REMOVE → EXECUTE
// ============================================================
func executeCodeCtx(ctx context.Context, code string) execResult {
	// 1. Buat temp file
	tmpDirMux.Lock()
	tmpFile, err := os.CreateTemp(tmpDir, "code-*.go")
	tmpDirMux.Unlock()
	if err != nil {
		return execResult{stderr: "Failed to create temp file: " + err.Error()}
	}
	filePath := tmpFile.Name()

	// 2. Tulis kode
	n, err := tmpFile.WriteString(code)
	if err != nil {
		tmpFile.Close()
		os.Remove(filePath)
		return execResult{stderr: "Failed to write code: " + err.Error()}
	}
	log.Printf("[EXEC] Wrote %d bytes to %s", n, filePath)

	// 3. Flush + tutup
	if err := tmpFile.Close(); err != nil {
		os.Remove(filePath)
		return execResult{stderr: "Failed to close temp file: " + err.Error()}
	}

	// 4. Cleanup setelah selesai
	defer os.Remove(filePath)

	// 5. Execute
	log.Printf("[EXEC] Running 'go run %s'...", filePath)
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "go", "run", filePath)
	cmd.Dir = tmpDir
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		log.Printf("[EXEC] TIMEOUT")
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

	return execResult{
		stdout: stdout.String(),
		stderr: strings.TrimSpace(stderr.String()),
	}
}

// ============================================================
// HELPERS
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

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

const faviconSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#3fb950"><polygon points="1,1 23,12 1,23"/></svg>`
