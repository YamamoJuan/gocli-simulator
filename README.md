# Go CLI Simulator

A web-based terminal simulator for running Go code directly in the browser.
Built with pure Golang (`net/http`) backend and vanilla HTML/CSS/JS frontend.
No external JS libraries required.

---

## 🚀 Features

### 🌐 Interactive Web Terminal
- Style Git Bash dengan dark theme
- macOS-style window controls (Close, Minimize, Maximize)
- Blinking cursor animation
- Command history dengan Arrow Up/Down
- Auto-scroll output

### 🐹 Built-in gonano Text Editor
- GNU nano-style text editor di dalam browser
- Line numbers dengan active line highlight
- Smart indent (Tab otomatis nambah level setelah `{`)
- Shortcut bar di bagian bawah
- `Ctrl+R` (double-press) — Execute code
- `Ctrl+X` (double-press) — Exit editor
- `Esc` — Cancel shortcut menu

### ⚡ Execute Go Code
- Ketik kode Go langsung, tekan Enter untuk jalankan
- Auto-import package `fmt` — tidak perlu import manual
- Support multi-line code
- Capture stdout dan stderr
- 15-second execution timeout

### 🎛️ Built-in Commands

| Command | Description |
|---------|-------------|
| `gonano` | Open nano text editor |
| `clear` | Clear terminal screen |
| `help` | Show help message |
| `about` | About this simulator |
| `date` | Show current date and time |
| `echo <text>` | Echo text to terminal |
| `go version` | Show Go version info |
| *Go code* | Execute Go code directly |

### ⌨️ Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Enter` | Execute code / run command |
| `Ctrl+L` | Clear terminal |
| `Ctrl+C` | Cancel current input |
| `↑` / `↓` | Navigate command history |
| `Ctrl+R` | Execute from editor (gonano) |
| `Ctrl+X` | Exit editor (gonano) |
| `Tab` | Smart indent (gonano) |
| `Esc` | Cancel menu (gonano) |

---

## 🛠️ Tech Stack

### Backend
```text
Go 1.21+
├── net/http          — HTTP server
├── context           — Timeout / deadline
├── os/exec           — Execute 'go run'
└── io / bytes / fmt  — I/O utilities
Frontend
Plaintext
HTML5                 — Semantic markup
├── CSS3              — Custom properties, flexbox
│   └── Solarized     — gonano dark color scheme
└── Vanilla JS        — No external dependencies
    ├── DOM manipulation
    ├── Fetch API     — HTTP requests
    └── ES6+ features — async/await, template literals
Dependencies:

Backend (Go): Zero external dependencies — hanya standard library (stdlib).

Frontend: Zero external dependencies — hanya browser native APIs.

🏁 Quick Start
Bash
# Clone atau cd ke project
cd gocli-simulator

# Install dependencies
go mod tidy

# Jalankan server
go run main.go

# Buka browser
# http://localhost:8080
📦 Deployment
Vercel (Serverless)
Bash
# Login Vercel
vercel login

# Deploy
vercel --prod
⚠️ Catatan: Vercel menggunakan model serverless yang memiliki limit waktu eksekusi. Untuk fitur go run secara penuh, disarankan menggunakan VPS atau Docker.

Docker
Bash
# Build image
docker build -t gocli-simulator .

# Jalankan container
docker run -p 8080:8080 --name gocli gocli-simulator

# Buka browser
# http://localhost:8080
VPS / Server
Bash
# Build binary
go build -o gocli-simulator main.go

# Jalankan
PORT=8080 ./gocli-simulator

# Atau dengan systemd (Linux)
sudo systemctl enable gocli
📁 Project Structure
Plaintext
gocli-simulator/
├── main.go                  # Go backend (net/http server)
├── go.mod                   # Go module definition
├── index.html               # Main HTML (terminal + gonano overlay)
├── static/
│   ├── css/
│   │   └── terminal.css     # All styling (terminal + gonano)
│   └── js/
│       └── terminal.js      # All client-side logic
├── README.md                # Project documentation
├── vercel.json              # Vercel serverless config
├── vercel.yaml              # Vercel CLI manifest
└── .gitignore               # Git ignore rules
🔌 API Reference
POST /run
Execute Go code and return output.

Request:

HTTP
Content-Type: text/plain

<Go source code as string>
Response (Success):

JSON
{
  "success": true,
  "stdout": "Hello, World!\n",
  "stderr": ""
}
Response (Error):

JSON
{
  "success": true,
  "stdout": "",
  "stderr": "command exited with code 1: ..."
}
GET /
Serve the main HTML page.

📄 License
MIT