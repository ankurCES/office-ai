<p align="center">
  <img src="assets/banner.svg" alt="Quill Banner" width="100%"/>
</p>

<p align="center">
  <strong>Quill — AI-Powered Desktop Office Suite</strong>
</p>

<p align="center">
  <a href="https://github.com/ankurCES/office-ai/releases"><img src="https://img.shields.io/github/v/release/ankurCES/office-ai?style=flat-square&color=6366f1&label=release" alt="Release"/></a>
  <a href="https://github.com/ankurCES/office-ai/actions"><img src="https://img.shields.io/github/actions/workflow/status/ankurCES/office-ai/build.yml?style=flat-square&label=build" alt="Build"/></a>
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-8b5cf6?style=flat-square" alt="Platform"/>
  <img src="https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat-square&logo=go" alt="Go"/>
  <img src="https://img.shields.io/badge/Wails-2.x-a78bfa?style=flat-square" alt="Wails"/>
</p>

---

**Quill** is a lightweight, native desktop office suite built with Go + [Wails](https://wails.io) + React. It provides document editing, spreadsheets, presentations, PDF viewing, and Markdown — all enhanced with built-in AI from OpenAI, Anthropic, and Ollama.

### Why Quill?

| | Electron/Web Apps | **Quill** |
|---|---|---|
| **Memory** | 200–500 MB | ~30 MB |
| **Startup** | 3–8 seconds | < 1 second |
| **Binary** | 150+ MB | ~25 MB |
| **AI** | Cloud-only | Local + Cloud |
| **Privacy** | Data leaves device | Everything local-first |

---

## 🪶 The Quill Suite

| App | Description | Formats |
|-----|-------------|---------|
| **Quill Write** | Rich document editor with styles, paragraphs, formatting | `.docx`, `.txt`, `.html` |
| **Quill Calc** | Spreadsheet with formulas, multi-sheet, cell formatting | `.xlsx`, `.csv` |
| **Quill Present** | Slide deck editor with elements, layouts, notes | `.pptx` |
| **Quill View** | PDF viewer with page navigation, rotation, merge/split | `.pdf` |
| **Quill Note** | Markdown editor with live preview | `.md`, `.txt` |
| **Quill AI** | AI assistant panel — chat, summarize, translate, rewrite | — |

---

## ⚡ Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/ankurCES/office-ai/main/install.sh | bash
```

### Install Options

```bash
# Custom install prefix
curl -fsSL https://raw.githubusercontent.com/ankurCES/office-ai/main/install.sh | bash -s -- --prefix /opt/quill

# Specific version
curl -fsSL https://raw.githubusercontent.com/ankurCES/office-ai/main/install.sh | bash -s -- --version 0.2.0

# Build from source
curl -fsSL https://raw.githubusercontent.com/ankurCES/office-ai/main/install.sh | bash -s -- --from-source
```

### Build from Source

```bash
git clone https://github.com/ankurCES/office-ai.git
cd office-ai
./build.sh
# Binary → build/bin/quill
```

---

## 🏗️ Architecture

```
quill/
├── main.go                    # Wails bootstrap + service binding
├── app.go                     # App lifecycle, file dialogs
├── internal/
│   ├── docs/                  # Quill Write — docx parsing/editing/save
│   ├── sheets/                # Quill Calc — xlsx via excelize
│   ├── slides/                # Quill Present — pptx editing
│   ├── pdf/                   # Quill View — PDF via pdfcpu
│   ├── markdown/              # Quill Note — markdown editing
│   └── shell/                 # App shell — tabs, recents, settings
├── pkg/
│   ├── agentcore/             # AI orchestration engine
│   ├── aiprovider/            # OpenAI, Anthropic, Ollama providers
│   ├── config/                # Persistent config (~/.quill/config.json)
│   ├── fileparse/             # File type detection
│   ├── fileutil/              # Atomic writes, temp dirs, hashing
│   ├── i18n/                  # 7-language translations
│   ├── projectstore/          # Recent files + project persistence
│   └── shortcuts/             # Keyboard shortcut registry
├── frontend/
│   └── src/
│       ├── App.tsx            # Root layout, tab management
│       ├── services/
│       │   └── wails-bridge.ts # TypeScript ↔ Go binding layer
│       └── components/
│           ├── Home.tsx       # New document / recent files
│           ├── TabBar.tsx     # Multi-tab interface
│           ├── AiPanel.tsx    # Quill AI assistant
│           ├── SettingsModal.tsx
│           └── editors/       # DocsEditor, SheetsEditor, etc.
├── build.sh                   # Cross-platform build (macOS + Linux)
└── install.sh                 # One-line curl|bash installer
```

### Data Flow

```
┌─────────────────────────────────────────────────┐
│                   Quill (Wails)                  │
│  ┌──────────────┐          ┌──────────────────┐  │
│  │  React + TS   │◄─IPC──►│   Go Backend      │  │
│  │  Frontend     │         │                    │  │
│  │  • Editors    │         │  • internal/*      │  │
│  │  • AI Panel   │         │  • pkg/agentcore   │  │
│  │  • Settings   │         │  • pkg/aiprovider  │  │
│  └──────────────┘          └──────────────────┘  │
│                                    │              │
│                          ┌─────────┴─────────┐   │
│                          │  ~/.quill/         │   │
│                          │  config.json       │   │
│                          │  projects/         │   │
│                          │  recents.json      │   │
│                          └───────────────────┘   │
└─────────────────────────────────────────────────┘
```

---

## 🤖 AI Setup

Quill AI supports three providers. Set your API key in Settings (⚙️) or environment:

```bash
# OpenAI
export OPENAI_API_KEY="sk-..."

# Anthropic
export ANTHROPIC_API_KEY="sk-ant-..."

# Ollama (local, no key needed)
# Just run: ollama serve
```

### AI Capabilities

- **Chat** — conversational AI across all editors
- **Summarize** — condense documents, spreadsheet data, slides
- **Translate** — translate content to any language
- **Rewrite** — improve tone, grammar, style
- **Explain** — break down formulas, code blocks, complex text
- **Generate** — create content from prompts

---

## ⌨️ Keyboard Shortcuts

| Action | Shortcut |
|--------|----------|
| New document | `Ctrl+N` |
| Open file | `Ctrl+O` |
| Save | `Ctrl+S` |
| Save as | `Ctrl+Shift+S` |
| Close tab | `Ctrl+W` |
| Undo | `Ctrl+Z` |
| Redo | `Ctrl+Y` |
| Toggle AI panel | `Ctrl+Shift+A` |
| Settings | `Ctrl+,` |
| Bold | `Ctrl+B` |
| Italic | `Ctrl+I` |
| Underline | `Ctrl+U` |

---

## ⚙️ Configuration

Settings are stored in `~/.quill/config.json`:

```json
{
  "theme": "system",
  "language": "en",
  "font_size": 14,
  "auto_save_delay": 2000,
  "ai_provider": "anthropic",
  "ai_model": "claude-sonnet-4-20250514",
  "show_line_numbers": true,
  "spell_check": true,
  "word_wrap": true,
  "minimap": false
}
```

---

## 🌐 Languages

Quill supports 7 languages:

| Language | Code |
|----------|------|
| English | `en` |
| 中文 (Chinese) | `zh` |
| 日本語 (Japanese) | `ja` |
| 한국어 (Korean) | `ko` |
| Español (Spanish) | `es` |
| Français (French) | `fr` |
| Deutsch (German) | `de` |

---

## 📊 GenOffice → Quill Migration

Quill is a ground-up rewrite of [GenOffice](https://github.com/nicosql/genoffice):

| Component | GenOffice (Electron) | Quill (Wails) |
|-----------|---------------------|---------------|
| Runtime | Electron + Node.js | Go + native WebView |
| Frontend | React + TypeScript | React + TypeScript |
| Docs engine | Custom XML parser | Go docx (archive/zip) |
| Sheets engine | Rust sidecar (calamine) | excelize (pure Go) |
| Slides engine | Custom XML parser | Go pptx (archive/zip) |
| PDF engine | pdf.js | pdfcpu (pure Go) |
| AI | TypeScript agents | Go agentcore + streaming |
| Config | electron-store | JSON file (~/.quill/) |
| Install | .dmg / .AppImage | Single binary + curl installer |
| Memory | ~300 MB | ~30 MB |
| Binary size | ~150 MB | ~25 MB |

---

## 🛠️ Development

```bash
# Prerequisites
go install github.com/wailsapp/wails/v2/cmd/wails@latest
cd frontend && npm install && cd ..

# Dev mode (hot reload)
wails dev

# Build
./build.sh

# Run tests
go test ./...
```

### Contributing

```bash
git clone https://github.com/YOUR_USERNAME/office-ai.git
cd office-ai
wails dev   # starts dev server with hot reload
```

---

## 📄 License

MIT © [ankurCES](https://github.com/ankurCES)
