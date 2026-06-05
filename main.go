package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	tmpDir    string
	tmpDirMux sync.Mutex

	// Resolve root directory sekali saat startup
	rootDir string
)

func init() {
	var err error
	tmpDir, err = os.MkdirTemp("", "go-cli-*")
	if err != nil {
		log.Fatalf("Failed to create tmp dir: %v", err)
	}

	// Resolve root directory saat startup
	rootDir = resolveRootDir()
	log.Printf("📂 Project root: %s", rootDir)
	log.Printf("📁 Temp directory: %s", tmpDir)
}

// resolveRootDir mencari direktori project (tempat index.html ada)
func resolveRootDir() string {
	exe, err := os.Executable()
	if err != nil {
		log.Printf("⚠️  Could not determine executable path: %v", err)
		exe = ""
	}

	// Lokasi-lokasi yang perlu dicek
	candidates := []string{
		".",              // current directory
		"..",             // parent (jika exe dijalankan dari subdirektori bin/)
		"../..",          // grandparent
		"../../..",       // great-grandparent
		"../../../..",    // great-great-grandparent (Vercel depth)
		"../../../../..", // max depth untuk Vercel
	}

	// Jika kita tahu path executable, mulai dari direktori parentnya
	if exe != "" {
		exeDir := filepath.Dir(exe)
		// Vercel: exe ada di seperti .output/go/main atau /var/task/.output/go/main
		// index.html ada di root project, jadi perlu naik beberapa level
		for depth := 1; depth <= 6; depth++ {
			dir := exeDir
			for i := 0; i < depth; i++ {
				dir = filepath.Dir(dir)
			}
			candidates = append(candidates, dir)
		}
	}

	// Cek semua kandidat
	for _, dir := range candidates {
		indexPath := filepath.Join(dir, "index.html")
		staticPath := filepath.Join(dir, "static")

		if _, err := os.Stat(indexPath); err == nil {
			if _, err := os.Stat(staticPath); err == nil {
				log.Printf("✅ Found project root at: %s", dir)
				return dir
			}
		}
	}

	// Fallback: asumsikan current directory
	log.Printf("⚠️  Could not auto-detect project root, using '.'")
	return "."
}

// getFilePath mengembalikan absolute path relatif terhadap rootDir
func getFilePath(name string) string {
	return filepath.Join(rootDir, name)
}

// fileExists mengecek apakah file ada
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func main() {
	port := getEnvOrDefault("PORT", "8080")

	// Serve static files dari direktori static yang sudah di-resolve
	staticDir := getFilePath("static")
	if !fileExists(staticDir) {
		log.Printf("❌ Static directory not found at: %s", staticDir)
	} else {
		staticFS := http.FileServer(http.Dir(staticDir))
		http.Handle("/static/", http.StripPrefix("/static/", staticFS))
		log.Printf("📦 Serving static files from: %s", staticDir)
	}

	// Serve index.html
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Jika path /run, handle sebagai POST execution
		if r.URL.Path == "/run" && r.Method == http.MethodPost {
			handleRun(w, r)
			return
		}

		// Untuk GET / atau route lain, serve index.html
		indexPath := getFilePath("index.html")
		if !fileExists(indexPath) {
			log.Printf("❌ index.html not found at: %s", indexPath)
			// Coba berbagai fallback
			fallbacks := []string{
				filepath.Join(rootDir, "index.html"),
				"index.html",
				"../index.html",
				"../../index.html",
				"../../../index.html",
			}
			found := false
			for _, f := range fallbacks {
				if fileExists(f) {
					indexPath = f
					log.Printf("✅ Using fallback index.html: %s", f)
					found = true
					break
				}
			}
			if !found {
				http.Error(w, "index.html not found", http.StatusInternalServerError)
				return
			}
		}

		http.ServeFile(w, r, indexPath)
	})

	// Handle code execution (POST only)
	http.HandleFunc("/run", handleRun)

	log.Printf("🌟 Web CLI Simulator running at http://localhost:%s", port)
	log.Printf("🌐 Serving from project root: %s", rootDir)
	log.Printf("💡 PORT env=%s, fallback=8080", os.Getenv("PORT"))

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// ============================================================
// HANDLE /run
// ============================================================
func handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// Jika GET, return 404 (API only)
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

	// Execute dengan timeout 15 detik
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := executeCodeCtx(ctx, code)
	sendResponse(w, true, result.stdout, result.stderr)
}

type execResult struct {
	stdout string
	stderr string
}

// executeCodeCtx menjalankan kode dengan context untuk timeout
func executeCodeCtx(ctx context.Context, code string) execResult {
	// Tulis kode ke file temporary
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

	// Execute
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
