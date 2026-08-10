/**
 * Wails bridge: wraps window.go.* bindings with typed async functions.
 * In dev mode (no Wails runtime), calls fall through to mock implementations.
 */

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type GoBinding = (...args: any[]) => Promise<any>

function getBinding(pkg: string, method: string): GoBinding | null {
  try {
    // Wails v2 exposes Go bindings as window.go['package']['Service']['Method']
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const w = window as any
    return w?.go?.[pkg]?.Service?.[method] ?? w?.go?.[pkg]?.[method] ?? null
  } catch {
    return null
  }
}

// Safe call: tries Wails binding, falls back to provided default
async function call<T>(
  pkg: string,
  method: string,
  args: unknown[] = [],
  fallback?: T,
): Promise<T> {
  const fn = getBinding(pkg, method)
  if (fn) {
    return fn(...args) as Promise<T>
  }
  if (fallback !== undefined) return fallback
  throw new Error(`Wails binding not available: ${pkg}.${method}`)
}

// ── Shell Service ──
export const ShellService = {
  getTabs: () =>
    call<TabSummary[]>('internal/shell', 'GetTabs', [], []),
  openTab: (kind: string, filePath: string) =>
    call<string>('internal/shell', 'OpenTab', [kind, filePath], ''),
  activateTab: (id: string) =>
    call<void>('internal/shell', 'ActivateTab', [id]),
  closeTab: (id: string) =>
    call<boolean>('internal/shell', 'CloseTab', [id], true),
  setTabDirty: (id: string, dirty: boolean) =>
    call<void>('internal/shell', 'SetTabDirty', [id, dirty]),
  setTabTitle: (id: string, title: string) =>
    call<void>('internal/shell', 'SetTabTitle', [id, title]),
  getSettings: () =>
    call<AppSettings>('internal/shell', 'GetSettings', [], DEFAULT_SETTINGS),
  updateSetting: (key: string, value: unknown) =>
    call<void>('internal/shell', 'UpdateSetting', [key, value]),
  getRecentFiles: () =>
    call<RecentFile[]>('internal/shell', 'GetRecentFiles', [], []),
  toggleStarred: (path: string) =>
    call<void>('internal/shell', 'ToggleStarred', [path]),
  removeRecent: (path: string) =>
    call<void>('internal/shell', 'RemoveRecent', [path]),
}

// ── Docs Service ──
export const DocsService = {
  openFile: (tabId: string, path: string) =>
    call<OpenFileResult>('internal/docs', 'OpenFile', [tabId, path]),
  newBlank: (tabId: string) =>
    call<OpenFileResult>('internal/docs', 'NewBlank', [tabId]),
  save: (tabId: string) =>
    call<SaveResult>('internal/docs', 'Save', [tabId]),
  saveAs: (tabId: string, path: string) =>
    call<SaveResult>('internal/docs', 'SaveAs', [tabId, path]),
  getState: (tabId: string) =>
    call<DocState | null>('internal/docs', 'GetState', [tabId], null),
  isDirty: (tabId: string) =>
    call<boolean>('internal/docs', 'IsDirty', [tabId], false),
  close: (tabId: string) =>
    call<void>('internal/docs', 'Close', [tabId]),
}

// ── Sheets Service ──
export const SheetsService = {
  openFile: (tabId: string, path: string) =>
    call<Record<string, unknown>>('internal/sheets', 'OpenFile', [tabId, path]),
  newBlank: (tabId: string) =>
    call<Record<string, unknown>>('internal/sheets', 'NewBlank', [tabId]),
  getCellRange: (tabId: string, sr: number, sc: number, er: number, ec: number) =>
    call<CellValue[]>('internal/sheets', 'GetCellRange', [tabId, sr, sc, er, ec], []),
  setCellValue: (tabId: string, row: number, col: number, value: unknown) =>
    call<void>('internal/sheets', 'SetCellValue', [tabId, row, col, value]),
  recalc: (tabId: string) =>
    call<void>('internal/sheets', 'Recalc', [tabId]),
  save: (tabId: string) =>
    call<Record<string, unknown>>('internal/sheets', 'Save', [tabId]),
  close: (tabId: string) =>
    call<void>('internal/sheets', 'Close', [tabId]),
}

// ── Slides Service ──
export const SlidesService = {
  openFile: (tabId: string, path: string) =>
    call<Record<string, unknown>>('internal/slides', 'OpenFile', [tabId, path]),
  newBlank: (tabId: string) =>
    call<Record<string, unknown>>('internal/slides', 'NewBlank', [tabId]),
  addSlide: (tabId: string, afterIndex: number, layout: string) =>
    call<void>('internal/slides', 'AddSlide', [tabId, afterIndex, layout]),
  deleteSlide: (tabId: string, index: number) =>
    call<void>('internal/slides', 'DeleteSlide', [tabId, index]),
  addElement: (tabId: string, slideIndex: number, elem: unknown) =>
    call<void>('internal/slides', 'AddElement', [tabId, slideIndex, elem]),
  deleteElement: (tabId: string, slideIndex: number, elemId: string) =>
    call<void>('internal/slides', 'DeleteElement', [tabId, slideIndex, elemId]),
  save: (tabId: string) =>
    call<Record<string, unknown>>('internal/slides', 'Save', [tabId]),
  close: (tabId: string) =>
    call<void>('internal/slides', 'Close', [tabId]),
}

// ── PDF Service ──
export const PdfService = {
  openFile: (tabId: string, path: string) =>
    call<Record<string, unknown>>('internal/pdf', 'OpenFile', [tabId, path]),
  getPagePreview: (tabId: string, pageIndex: number) =>
    call<string>('internal/pdf', 'GetPagePreview', [tabId, pageIndex], ''),
  extractPages: (tabId: string, pages: number[], outputPath: string) =>
    call<Record<string, unknown>>('internal/pdf', 'ExtractPages', [tabId, pages, outputPath]),
  save: (tabId: string) =>
    call<Record<string, unknown>>('internal/pdf', 'Save', [tabId]),
  close: (tabId: string) =>
    call<void>('internal/pdf', 'Close', [tabId]),
}

// ── Markdown Service ──
export const MarkdownService = {
  openFile: (tabId: string, path: string) =>
    call<Record<string, unknown>>('internal/markdown', 'OpenFile', [tabId, path]),
  newBlank: (tabId: string) =>
    call<Record<string, unknown>>('internal/markdown', 'NewBlank', [tabId]),
  updateContent: (tabId: string, content: string) =>
    call<void>('internal/markdown', 'UpdateContent', [tabId, content]),
  save: (tabId: string) =>
    call<Record<string, unknown>>('internal/markdown', 'Save', [tabId]),
  saveAs: (tabId: string, path: string) =>
    call<Record<string, unknown>>('internal/markdown', 'SaveAs', [tabId, path]),
  close: (tabId: string) =>
    call<void>('internal/markdown', 'Close', [tabId]),
}

// ── AI Provider Service ──
export const AIProviderService = {
  getSettings: () =>
    call<AISettings>('pkg/aiprovider', 'GetSettings', [], DEFAULT_AI_SETTINGS),
  updateSettings: (settings: AISettings) =>
    call<void>('pkg/aiprovider', 'UpdateSettings', [settings]),
  chat: (req: ChatRequest) =>
    call<ChatResponse>('pkg/aiprovider', 'Chat', [req]),
}

// ── Agent Core ──
export const AgentService = {
  run: (userText: string, opts: { skill?: string; contextText?: string }) =>
    call<RunResult>('pkg/agentcore', 'Run', [userText, opts]),
  clearHistory: () =>
    call<void>('pkg/agentcore', 'ClearHistory'),
  getHistory: () =>
    call<AIMessage[]>('pkg/agentcore', 'GetHistory', [], []),
}

// ── I18n Service ──
export const I18nService = {
  setLang: (lang: string) =>
    call<void>('pkg/i18n', 'SetLang', [lang]),
  getLang: () =>
    call<string>('pkg/i18n', 'GetLang', [], 'en'),
  t: (key: string) =>
    call<string>('pkg/i18n', 'T', [key], key),
}

// ── Wails Runtime ──
export const WailsRuntime = {
  openFileDialog: async (): Promise<string> => {
    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const w = window as any
      if (w?.runtime?.OpenFileDialog) {
        return await w.runtime.OpenFileDialog({})
      }
      if (w?.go?.main?.App?.OpenFileDialog) {
        return await w.go.main.App.OpenFileDialog()
      }
      return ''
    } catch {
      return ''
    }
  },
  saveFileDialog: async (): Promise<string> => {
    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const w = window as any
      if (w?.runtime?.SaveFileDialog) {
        return await w.runtime.SaveFileDialog({})
      }
      return ''
    } catch {
      return ''
    }
  },
  eventsOn: (event: string, callback: (...args: unknown[]) => void) => {
    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const w = window as any
      return w?.runtime?.EventsOn?.(event, callback)
    } catch {
      return () => {}
    }
  },
  eventsEmit: (event: string, ...data: unknown[]) => {
    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const w = window as any
      w?.runtime?.EventsEmit?.(event, ...data)
    } catch {
      // no-op
    }
  },
  windowSetTitle: (title: string) => {
    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const w = window as any
      w?.runtime?.WindowSetTitle?.(title)
    } catch {
      document.title = title
    }
  },
  quit: () => {
    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const w = window as any
      w?.runtime?.Quit?.()
    } catch {
      // no-op
    }
  },
}

// ── Shared Types (match wails.d.ts) ──
export interface TabSummary {
  id: string
  kind: string
  title: string
  file_path: string
  is_dirty: boolean
}

export interface RecentFile {
  name: string
  path: string
  kind: string
  opened_at: number
  starred: boolean
}

export interface AppSettings {
  theme: string
  language: string
  auto_save: boolean
  auto_save_interval: number
  show_recent: boolean
}

const DEFAULT_SETTINGS: AppSettings = {
  theme: 'system',
  language: 'en',
  auto_save: true,
  auto_save_interval: 30,
  show_recent: true,
}

export interface OpenFileResult {
  success: boolean
  file_path: string
  title: string
  error?: string
}

export interface SaveResult {
  success: boolean
  file_path: string
  error?: string
}

export interface DocState {
  file_path: string
  is_dirty: boolean
  title: string
  word_count: number
  page_count: number
  paragraphs?: string[]
}

export interface CellValue {
  row: number
  col: number
  value: string
  formula?: string
}

export interface AISettings {
  provider: string
  api_key: string
  model: string
}

const DEFAULT_AI_SETTINGS: AISettings = {
  provider: 'anthropic',
  api_key: '',
  model: 'claude-sonnet-4-20250514',
}

export interface ChatRequest {
  messages: AIMessage[]
  system?: string
  max_tokens?: number
}

export interface ChatResponse {
  text: string
  usage?: { input_tokens: number; output_tokens: number }
}

export interface AIMessage {
  role: 'user' | 'assistant' | 'tool'
  text?: string
}

export interface RunResult {
  text: string
  cancelled: boolean
  turn_limit: boolean
}
