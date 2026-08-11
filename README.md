<p align="center">
  <img src="assets/banner.svg" alt="Office AI Banner" width="100%"/>
</p>

<p align="center">
  <strong>A native, AI-powered desktop office suite built with Go + Wails + React</strong>
</p>

<p align="center">
  <a href="#installation"><img src="https://img.shields.io/badge/install-curl%20%7C%20bash-blue?style=flat-square" alt="Install"/></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go"/></a>
  <a href="https://wails.io"><img src="https://img.shields.io/badge/Wails-v2.14-8b5cf6?style=flat-square" alt="Wails"/></a>
  <a href="https://react.dev"><img src="https://img.shields.io/badge/React-18-61DAFB?style=flat-square&logo=react&logoColor=black" alt="React"/></a>
  <a href="https://www.typescriptlang.org"><img src="https://img.shields.io/badge/TypeScript-5-3178C6?style=flat-square&logo=typescript&logoColor=white" alt="TypeScript"/></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="License"/></a>
  <a href="https://github.com/ankurCES/office-ai/actions"><img src="https://img.shields.io/github/actions/workflow/status/ankurCES/office-ai/build.yml?style=flat-square&label=build" alt="Build"/></a>
</p>

---

## Overview

**Office AI** is a lightweight, native desktop office suite that replaces heavyweight Electron apps with a fast Go backend and a modern React frontend, connected via [Wails](https://wails.io). It provides full document editing, spreadsheets, presentations, PDF viewing, and Markdown support — all enhanced with built-in AI assistance from OpenAI, Anthropic, and Ollama.

### Why Office AI?

| | Electron/Web Apps | **Office AI** |
|---|---|---|
| **Binary size** | 200–400 MB | ~15 MB |
| **RAM usage** | 300+ MB | ~50 MB |
| **Startup** | 3–8 seconds | < 1 second |
| **AI built-in** | Plugins/add-ons | Native, streaming |
| **Native OS feel** | Chromium wrapper | WebView2/WebKit |

---

## Features

### 📝 Documents (DOCX)
- Full OOXML parsing — paragraphs, runs, styles, fonts, spacing, indentation
- Rich text editing — bold, italic, underline, strikethrough, font size/family
- Section properties — page size, margins, columns, doc grid
- Paragraph numbering and list support
- Export to HTML and plain text
- Autosave with configurable delay

### 📊 Spreadsheets (XLSX)
- Real Excel parsing via [excelize](https://github.com/xuri/excelize)
- Multi-sheet workbooks with sheet management (add, rename, delete)
- Cell editing with type detection (string, number, formula, boolean)
- Cell range operations and merged cell support
- Formula evaluation engine
- Column/row insert and delete
- Create new blank workbooks
- Export to CSV

### 📽️ Presentations (PPTX)
- OOXML slide parsing — shapes, text frames, positioning
- Slide management — add, delete, reorder, duplicate
- Shape editing — text content, position, size
- Slide element manipulation
- Export individual slides to SVG
- Presentation state management

### 📄 PDF Viewer
- Full PDF parsing via [pdfcpu](https://github.com/pdfcpu/pdfcpu)
- Page-by-page viewing with page count
- Text extraction per page
- PDF metadata reading (author, title, subject, keywords)
- PDF merging — combine multiple PDFs into one
- Page extraction to new PDF
- Page rotation

### ✍️ Markdown Editor
- Live content editing with state tracking
- Save to `.md` files
- Integrated with AI assistant for content generation

### 🤖 AI Assistant
- **Multi-provider support**: OpenAI, Anthropic (Claude), Ollama (local)
- **Streaming responses** — real-time token-by-token output via SSE
- **Context-aware** — operates within the active document context
- **Model selection** — choose any model from your configured provider
- **Tool calls** — structured function calling support
- **Usage tracking** — token counts for input/output

### 🖥️ Shell & Tab Management
- Multi-tab interface — open multiple documents simultaneously
- Tab types: Documents, Sheets, Slides, PDF, Markdown
- Recent files with starring/favorites
- Persistent settings (theme, language, font, AI provider)
- Keyboard shortcuts with customization
- Internationalization (English, Chinese, Japanese, Korean, Spanish, French, German)

---

## Architecture

```
office-ai/
├── main.go                    # Wails app entry point
├── app.go                     # App lifecycle + file dialogs
├── internal/                  # Core services (bound to frontend via Wails)
│   ├── shell/shell.go         # Tab management, settings, recent files
│   ├── docs/                  # DOCX engine
│   │   ├── docs.go            # Full OOXML parser + editor
│   │   └── export.go          # HTML/text export
│   ├── sheets/sheets.go       # XLSX engine (excelize)
│   ├── slides/                # PPTX engine
│   │   ├── slides.go          # Slide parser + editor
│   │   └── export.go          # SVG export
│   ├── pdf/pdf.go             # PDF engine (pdfcpu)
│   └── markdown/markdown.go   # Markdown editor
├── pkg/                       # Shared packages
│   ├── aiprovider/            # OpenAI, Anthropic, Ollama with streaming
│   ├── agentcore/             # AI agent loop + tool execution
│   ├── config/                # App configuration persistence
│   ├── fileutil/              # File type detection, hashing, filters
│   ├── fileparse/             # Generic file content extraction
│   ├── i18n/                  # Internationalization (7 languages)
│   ├── projectstore/          # Project/file metadata store
│   └── shortcuts/             # Keyboard shortcut registry
├── frontend/                  # React + TypeScript UI
│   └── src/
│       ├── App.tsx            # Root component with routing
│       ├── services/
│       │   └── wails-bridge.ts  # Typed Go↔JS bindings (~550 lines)
│       ├── components/
│       │   ├── Home.tsx       # Welcome screen + recent files
│       │   ├── TabBar.tsx     # Multi-tab interface
│       │   ├── AiPanel.tsx    # AI assistant panel
│       │   ├── SettingsModal.tsx
│       │   ├── icons/FileIcons.tsx
│       │   └── editors/
│       │       ├── DocsEditor.tsx
│       │       ├── SheetsEditor.tsx
│       │       ├── SlidesEditor.tsx
│       │       ├── PdfViewer.tsx
│       │       ├── MarkdownEditor.tsx
│       │       └── EditorToolbar.tsx
│       └── styles/
├── build.sh                   # Cross-platform build script
├── install.sh                 # curl|bash installer
└── wails.json                 # Wails project config
```

### Data Flow

```
┌─────────────────────────────────────────────┐
│  Frontend (React + TypeScript)              │
│  ┌──────────┐ ┌──────────┐ ┌─────────────┐ │
│  │ Editors  │ │ TabBar   │ │  AI Panel   │ │
│  └────┬─────┘ └────┬─────┘ └──────┬──────┘ │
│       │             │              │        │
│  ┌────▼─────────────▼──────────────▼──────┐ │
│  │         wails-bridge.ts                │ │
│  │   Typed bindings for all Go services   │ │
│  └────────────────┬───────────────────────┘ │
└───────────────────┼─────────────────────────┘
                    │  Wails IPC (JSON-RPC)
┌───────────────────┼─────────────────────────┐
│  Backend (Go)     │                         │
│  ┌────────────────▼───────────────────────┐ │
│  │  Bound Services (shell, docs, sheets,  │ │
│  │  slides, pdf, markdown, aiprovider,    │ │
│  │  agentcore, shortcuts)                 │ │
│  └──┬─────────┬───────────┬───────────┬───┘ │
│     │         │           │           │     │
│  ┌──▼──┐  ┌──▼──┐  ┌─────▼───┐  ┌───▼───┐ │
│  │config│  │i18n │  │fileutil │  │store  │ │
│  └──────┘  └─────┘  └─────────┘  └───────┘ │
└─────────────────────────────────────────────┘
```

---

## Installation

### One-Line Install (macOS & Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/ankurCES/office-ai/main/install.sh | bash
```

The installer:
- Detects your OS (macOS, Linux) and architecture (amd64, arm64)
- Installs Go 1.25+ and Wails CLI if missing
- Installs platform dependencies (gtk3/webkit2gtk on Linux, Xcode tools on macOS)
- Builds the app from source
- Creates a `.desktop` entry (Linux) or `.app` bundle (macOS)
- Adds to your `$PATH`

### Install Options

```bash
# Custom install prefix
curl -fsSL https://raw.githubusercontent.com/ankurCES/office-ai/main/install.sh | bash -s -- --prefix /opt/office-ai

# Specific version
curl -fsSL https://raw.githubusercontent.com/ankurCES/office-ai/main/install.sh | bash -s -- --version 0.2.0

# Build from source (default)
curl -fsSL https://raw.githubusercontent.com/ankurCES/office-ai/main/install.sh | bash -s -- --from-source
```

### Build from Source

```bash
# Prerequisites
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Clone and build
git clone https://github.com/ankurCES/office-ai.git
cd office-ai
./build.sh              # Builds for current platform
./build.sh --all        # Builds for macOS + Linux (amd64 & arm64)
./build.sh --platform darwin/arm64   # Specific target
```

### Platform Requirements

| Platform | Requirements |
|----------|-------------|
| **macOS** | Xcode Command Line Tools, Go 1.25+ |
| **Linux** | `gcc`, `gtk3`, `webkit2gtk-4.0`, Go 1.25+ |

---

## Configuration

Settings are stored in `~/.office-ai/config.json`:

```json
{
  "theme": "system",
  "language": "en",
  "fontSize": 14,
  "fontFamily": "Inter",
  "autoSaveDelay": 2000,
  "provider": "anthropic",
  "showLineNumbers": true,
  "tabSize": 4,
  "wordWrap": true,
  "spellCheck": true,
  "recentFilesMax": 50
}
```

### AI Provider Setup

Configure your AI provider in Settings (`Ctrl+,`):

**OpenAI**
```json
{
  "provider": "openai",
  "apiKey": "sk-...",
  "model": "gpt-4o"
}
```

**Anthropic (Claude)**
```json
{
  "provider": "anthropic",
  "apiKey": "sk-ant-...",
  "model": "claude-sonnet-4-20250514"
}
```

**Ollama (Local)**
```json
{
  "provider": "ollama",
  "baseUrl": "http://localhost:11434",
  "model": "llama3"
}
```

---

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+N` | New document |
| `Ctrl+O` | Open file |
| `Ctrl+S` | Save |
| `Ctrl+Shift+S` | Save as |
| `Ctrl+W` | Close tab |
| `Ctrl+Tab` | Next tab |
| `Ctrl+Shift+Tab` | Previous tab |
| `Ctrl+,` | Settings |
| `Ctrl+B` | Bold |
| `Ctrl+I` | Italic |
| `Ctrl+U` | Underline |
| `Ctrl+Z` | Undo |
| `Ctrl+Shift+Z` | Redo |
| `Ctrl+Shift+A` | Toggle AI panel |

All shortcuts are customizable via the shortcuts registry.

---

## Development

```bash
# Live development mode (hot reload)
wails dev

# Run Go tests
go test ./...

# Run frontend type check
cd frontend && npx tsc --noEmit

# Build for production
wails build

# Lint
go vet ./...
cd frontend && npx eslint src/
```

### Test Coverage

```bash
# Run with coverage
go test -cover ./...

# Generate HTML report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

Tests cover: config persistence, i18n translations, keyboard shortcuts, project store, file utilities, AI provider request building, sheets operations, and markdown state management.

---

## Supported File Formats

| Format | Read | Write | Export |
|--------|------|-------|--------|
| `.docx` | ✅ | ✅ | HTML, TXT |
| `.xlsx` | ✅ | ✅ | CSV |
| `.pptx` | ✅ | ✅ | SVG |
| `.pdf` | ✅ | — | Text, Merge, Extract |
| `.md` | ✅ | ✅ | — |

---

## Internationalization

Office AI supports 7 languages out of the box:

| Code | Language |
|------|----------|
| `en` | English |
| `zh` | 中文 (Chinese) |
| `ja` | 日本語 (Japanese) |
| `ko` | 한국어 (Korean) |
| `es` | Español (Spanish) |
| `fr` | Français (French) |
| `de` | Deutsch (German) |

Switch via Settings → Language, or set `"language"` in config.

---

## Comparison with GenOffice (Electron)

Office AI is a ground-up rewrite of [GenOffice](https://github.com/nicosql/genoffice), replacing the Electron + TypeScript stack with Go + Wails:

| Component | GenOffice (Electron) | Office AI (Wails) |
|-----------|---------------------|-------------------|
| Runtime | Electron + Node.js | Native WebView |
| Backend | TypeScript | Go |
| Frontend | React + TypeScript | React + TypeScript |
| Doc engine | `docx-engine` (TS) | `internal/docs` (Go, encoding/xml) |
| Sheet engine | `xlsx-engine` (Rust sidecar) | `internal/sheets` (Go, excelize) |
| Slide engine | `pptx-engine` (TS) | `internal/slides` (Go, encoding/xml) |
| PDF | pdf.js | pdfcpu (Go native) |
| AI | `agent-core` (TS) | `pkg/aiprovider` + `pkg/agentcore` (Go) |
| Shell/tabs | `@nicosql/shell` (React) | `internal/shell` (Go) + React TabBar |
| Config | electron-store | `pkg/config` (JSON file) |
| i18n | i18next | `pkg/i18n` (Go native) |
| Build size | ~350 MB | ~15 MB |
| Memory | ~300 MB | ~50 MB |

---

## Contributing

```bash
# Fork and clone
git clone https://github.com/YOUR_USERNAME/office-ai.git
cd office-ai

# Install dependencies
go mod tidy
cd frontend && npm install && cd ..

# Run in dev mode
wails dev

# Before submitting a PR
go test ./...
go vet ./...
cd frontend && npx tsc --noEmit
```

---

## License

MIT — see [LICENSE](LICENSE) for details.

---

<p align="center">
  <sub>Built with ❤️ using <a href="https://go.dev">Go</a>, <a href="https://wails.io">Wails</a>, and <a href="https://react.dev">React</a></sub>
</p>
