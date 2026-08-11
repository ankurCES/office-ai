/**
 * Wails bridge: wraps window.go.* bindings with typed async functions.
 * In dev mode (no Wails runtime), calls fall through to mock implementations.
 *
 * Binding paths match Wails v2 conventions:
 *   window.go['package/path']['StructName']['MethodName']
 * For our project the services are bound at their package path.
 */

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type GoBinding = (...args: any[]) => Promise<any>

function getBinding(pkg: string, struct: string, method: string): GoBinding | null {
  try {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const w = window as any
    return w?.go?.[pkg]?.[struct]?.[method] ?? null
  } catch {
    return null
  }
}

async function call<T>(
  pkg: string,
  struct: string,
  method: string,
  args: unknown[] = [],
  fallback?: T,
): Promise<T> {
  const fn = getBinding(pkg, struct, method)
  if (fn) return fn(...args) as Promise<T>
  if (fallback !== undefined) return fallback
  throw new Error(`Wails binding not available: ${pkg}.${struct}.${method}`)
}

// ── Shell Service ──
// Go: internal/shell.Service
export const ShellService = {
  getTabs: () =>
    call<TabSummary[]>('github.com/ankurCES/office-ai/internal/shell', 'Service', 'GetTabs', [], []),
  openTab: (kind: string, filePath: string) =>
    call<string>('github.com/ankurCES/office-ai/internal/shell', 'Service', 'OpenTab', [kind, filePath], ''),
  activateTab: (id: string) =>
    call<void>('github.com/ankurCES/office-ai/internal/shell', 'Service', 'ActivateTab', [id]),
  closeTab: (id: string) =>
    call<boolean>('github.com/ankurCES/office-ai/internal/shell', 'Service', 'CloseTab', [id], true),
  setTabDirty: (id: string, dirty: boolean) =>
    call<void>('github.com/ankurCES/office-ai/internal/shell', 'Service', 'SetTabDirty', [id, dirty]),
  setTabTitle: (id: string, title: string) =>
    call<void>('github.com/ankurCES/office-ai/internal/shell', 'Service', 'SetTabTitle', [id, title]),
  getSettings: () =>
    call<AppSettings>('github.com/ankurCES/office-ai/internal/shell', 'Service', 'GetSettings', [], DEFAULT_SETTINGS),
  updateSetting: (key: string, value: unknown) =>
    call<void>('github.com/ankurCES/office-ai/internal/shell', 'Service', 'UpdateSetting', [key, value]),
  getRecentFiles: () =>
    call<RecentFile[]>('github.com/ankurCES/office-ai/internal/shell', 'Service', 'GetRecentFiles', [], []),
  toggleStarred: (path: string) =>
    call<void>('github.com/ankurCES/office-ai/internal/shell', 'Service', 'ToggleStarred', [path]),
  removeRecent: (path: string) =>
    call<void>('github.com/ankurCES/office-ai/internal/shell', 'Service', 'RemoveRecent', [path]),
}

// ── Docs Service ──
// Go: internal/docs.Service
export const DocsService = {
  openFile: (tabId: string, path: string) =>
    call<OpenFileResult>('github.com/ankurCES/office-ai/internal/docs', 'Service', 'OpenFile', [tabId, path]),
  newBlank: (tabId: string) =>
    call<OpenFileResult>('github.com/ankurCES/office-ai/internal/docs', 'Service', 'NewBlank', [tabId]),
  save: (tabId: string) =>
    call<SaveResult>('github.com/ankurCES/office-ai/internal/docs', 'Service', 'Save', [tabId]),
  saveAs: (tabId: string, path: string) =>
    call<SaveResult>('github.com/ankurCES/office-ai/internal/docs', 'Service', 'SaveAs', [tabId, path]),
  getState: (tabId: string) =>
    call<DocState | null>('github.com/ankurCES/office-ai/internal/docs', 'Service', 'GetState', [tabId], null),
  isDirty: (tabId: string) =>
    call<boolean>('github.com/ankurCES/office-ai/internal/docs', 'Service', 'IsDirty', [tabId], false),
  close: (tabId: string) =>
    call<void>('github.com/ankurCES/office-ai/internal/docs', 'Service', 'Close', [tabId]),
  updateParagraph: (tabId: string, index: number, text: string, bold: boolean, italic: boolean, align: string) =>
    call<boolean>('github.com/ankurCES/office-ai/internal/docs', 'Service', 'UpdateParagraph', [tabId, index, text, bold, italic, align]),
  insertParagraph: (tabId: string, index: number, text: string) =>
    call<boolean>('github.com/ankurCES/office-ai/internal/docs', 'Service', 'InsertParagraph', [tabId, index, text]),
  deleteParagraph: (tabId: string, index: number) =>
    call<boolean>('github.com/ankurCES/office-ai/internal/docs', 'Service', 'DeleteParagraph', [tabId, index]),
  findReplace: (tabId: string, search: string, replace: string, replaceAll: boolean) =>
    call<FindReplaceResult>('github.com/ankurCES/office-ai/internal/docs', 'Service', 'FindReplace', [tabId, search, replace, replaceAll]),
  find: (tabId: string, search: string) =>
    call<number[]>('github.com/ankurCES/office-ai/internal/docs', 'Service', 'Find', [tabId, search], []),
  undo: (tabId: string) =>
    call<boolean>('github.com/ankurCES/office-ai/internal/docs', 'Service', 'Undo', [tabId], false),
  redo: (tabId: string) =>
    call<boolean>('github.com/ankurCES/office-ai/internal/docs', 'Service', 'Redo', [tabId], false),
  getParagraphs: (tabId: string) =>
    call<ParaInfo[]>('github.com/ankurCES/office-ai/internal/docs', 'Service', 'GetParagraphs', [tabId], []),
}

// ── Sheets Service ──
// Go: internal/sheets.Service
// API: GetCellRange(tabID, sheetName, startRow, startCol, rows, cols) → [][]CellData
//      SetCellValue(tabID, sheetName, cellRef, value) → SaveResult
//      NewWorkbook(tabID) → map
export const SheetsService = {
  openFile: (tabId: string, path: string) =>
    call<Record<string, unknown>>('github.com/ankurCES/office-ai/internal/sheets', 'Service', 'OpenFile', [tabId, path]),
  newWorkbook: (tabId: string) =>
    call<Record<string, unknown>>('github.com/ankurCES/office-ai/internal/sheets', 'Service', 'NewWorkbook', [tabId]),
  getState: (tabId: string) =>
    call<WorkbookState | null>('github.com/ankurCES/office-ai/internal/sheets', 'Service', 'GetState', [tabId], null),
  getCellRange: (tabId: string, sheetName: string, startRow: number, startCol: number, rows: number, cols: number) =>
    call<CellData[][]>('github.com/ankurCES/office-ai/internal/sheets', 'Service', 'GetCellRange', [tabId, sheetName, startRow, startCol, rows, cols], []),
  getAllRows: (tabId: string, sheetName: string) =>
    call<string[][]>('github.com/ankurCES/office-ai/internal/sheets', 'Service', 'GetAllRows', [tabId, sheetName], []),
  setCellValue: (tabId: string, sheetName: string, cellRef: string, value: string) =>
    call<SaveResult>('github.com/ankurCES/office-ai/internal/sheets', 'Service', 'SetCellValue', [tabId, sheetName, cellRef, value]),
  setCellFormula: (tabId: string, sheetName: string, cellRef: string, formula: string) =>
    call<SaveResult>('github.com/ankurCES/office-ai/internal/sheets', 'Service', 'SetCellFormula', [tabId, sheetName, cellRef, formula]),
  addSheet: (tabId: string, name: string) =>
    call<SaveResult>('github.com/ankurCES/office-ai/internal/sheets', 'Service', 'AddSheet', [tabId, name]),
  deleteSheet: (tabId: string, name: string) =>
    call<SaveResult>('github.com/ankurCES/office-ai/internal/sheets', 'Service', 'DeleteSheet', [tabId, name]),
  renameSheet: (tabId: string, oldName: string, newName: string) =>
    call<SaveResult>('github.com/ankurCES/office-ai/internal/sheets', 'Service', 'RenameSheet', [tabId, oldName, newName]),
  mergeCells: (tabId: string, sheetName: string, startCell: string, endCell: string) =>
    call<SaveResult>('github.com/ankurCES/office-ai/internal/sheets', 'Service', 'MergeCells', [tabId, sheetName, startCell, endCell]),
  insertRow: (tabId: string, sheetName: string, row: number) =>
    call<SaveResult>('github.com/ankurCES/office-ai/internal/sheets', 'Service', 'InsertRow', [tabId, sheetName, row]),
  insertCol: (tabId: string, sheetName: string, col: number) =>
    call<SaveResult>('github.com/ankurCES/office-ai/internal/sheets', 'Service', 'InsertCol', [tabId, sheetName, col]),
  deleteRow: (tabId: string, sheetName: string, row: number) =>
    call<SaveResult>('github.com/ankurCES/office-ai/internal/sheets', 'Service', 'DeleteRow', [tabId, sheetName, row]),
  setColumnWidth: (tabId: string, sheetName: string, startCol: string, endCol: string, width: number) =>
    call<SaveResult>('github.com/ankurCES/office-ai/internal/sheets', 'Service', 'SetColumnWidth', [tabId, sheetName, startCol, endCol, width]),
  setRowHeight: (tabId: string, sheetName: string, row: number, height: number) =>
    call<SaveResult>('github.com/ankurCES/office-ai/internal/sheets', 'Service', 'SetRowHeight', [tabId, sheetName, row, height]),
  setAutoFilter: (tabId: string, sheetName: string, rangeRef: string) =>
    call<SaveResult>('github.com/ankurCES/office-ai/internal/sheets', 'Service', 'SetAutoFilter', [tabId, sheetName, rangeRef]),
  save: (tabId: string) =>
    call<SaveResult>('github.com/ankurCES/office-ai/internal/sheets', 'Service', 'Save', [tabId]),
  close: (tabId: string) =>
    call<void>('github.com/ankurCES/office-ai/internal/sheets', 'Service', 'Close', [tabId]),
}

// ── Slides Service ──
// Go: internal/slides.Service
export const SlidesService = {
  openFile: (tabId: string, path: string) =>
    call<SlideOpenResult>('github.com/ankurCES/office-ai/internal/slides', 'Service', 'OpenFile', [tabId, path]),
  newBlank: (tabId: string) =>
    call<SlideOpenResult>('github.com/ankurCES/office-ai/internal/slides', 'Service', 'NewBlank', [tabId]),
  addSlide: (tabId: string) =>
    call<SlideInfo>('github.com/ankurCES/office-ai/internal/slides', 'Service', 'AddSlide', [tabId]),
  deleteSlide: (tabId: string, index: number) =>
    call<boolean>('github.com/ankurCES/office-ai/internal/slides', 'Service', 'DeleteSlide', [tabId, index], false),
  duplicateSlide: (tabId: string, index: number) =>
    call<SlideInfo>('github.com/ankurCES/office-ai/internal/slides', 'Service', 'DuplicateSlide', [tabId, index]),
  moveSlide: (tabId: string, from: number, to: number) =>
    call<boolean>('github.com/ankurCES/office-ai/internal/slides', 'Service', 'MoveSlide', [tabId, from, to], false),
  updateElement: (tabId: string, slideIndex: number, elemId: string, text: string) =>
    call<boolean>('github.com/ankurCES/office-ai/internal/slides', 'Service', 'UpdateElement', [tabId, slideIndex, elemId, text], false),
  getSlides: (tabId: string) =>
    call<SlideInfo[]>('github.com/ankurCES/office-ai/internal/slides', 'Service', 'GetSlides', [tabId], []),
  getState: (tabId: string) =>
    call<DeckState | null>('github.com/ankurCES/office-ai/internal/slides', 'Service', 'GetState', [tabId], null),
  isDirty: (tabId: string) =>
    call<boolean>('github.com/ankurCES/office-ai/internal/slides', 'Service', 'IsDirty', [tabId], false),
  save: (tabId: string) =>
    call<SaveResult>('github.com/ankurCES/office-ai/internal/slides', 'Service', 'Save', [tabId]),
  saveAs: (tabId: string, path: string) =>
    call<SaveResult>('github.com/ankurCES/office-ai/internal/slides', 'Service', 'SaveAs', [tabId, path]),
  undo: (tabId: string) =>
    call<boolean>('github.com/ankurCES/office-ai/internal/slides', 'Service', 'Undo', [tabId], false),
  redo: (tabId: string) =>
    call<boolean>('github.com/ankurCES/office-ai/internal/slides', 'Service', 'Redo', [tabId], false),
  close: (tabId: string) =>
    call<void>('github.com/ankurCES/office-ai/internal/slides', 'Service', 'Close', [tabId]),
}

// ── PDF Service ──
// Go: internal/pdf.Service
export const PdfService = {
  openFile: (tabId: string, path: string) =>
    call<Record<string, unknown>>('github.com/ankurCES/office-ai/internal/pdf', 'Service', 'OpenFile', [tabId, path]),
  getState: (tabId: string) =>
    call<PDFState | null>('github.com/ankurCES/office-ai/internal/pdf', 'Service', 'GetState', [tabId], null),
  extractText: (tabId: string) =>
    call<string>('github.com/ankurCES/office-ai/internal/pdf', 'Service', 'ExtractText', [tabId], ''),
  extractPages: (tabId: string, pages: number[], outputPath: string) =>
    call<OpResult>('github.com/ankurCES/office-ai/internal/pdf', 'Service', 'ExtractPages', [tabId, pages, outputPath]),
  mergeFiles: (inputPaths: string[], outputPath: string) =>
    call<OpResult>('github.com/ankurCES/office-ai/internal/pdf', 'Service', 'MergeFiles', [inputPaths, outputPath]),
  splitFile: (tabId: string, outputDir: string) =>
    call<OpResult>('github.com/ankurCES/office-ai/internal/pdf', 'Service', 'SplitFile', [tabId, outputDir]),
  rotatePages: (tabId: string, rotation: number, pageNums: number[]) =>
    call<OpResult>('github.com/ankurCES/office-ai/internal/pdf', 'Service', 'RotatePages', [tabId, rotation, pageNums]),
  addWatermark: (tabId: string, text: string, outputPath: string) =>
    call<OpResult>('github.com/ankurCES/office-ai/internal/pdf', 'Service', 'AddWatermark', [tabId, text, outputPath]),
  validate: (path: string) =>
    call<OpResult>('github.com/ankurCES/office-ai/internal/pdf', 'Service', 'Validate', [path]),
  optimize: (tabId: string, outputPath: string) =>
    call<OpResult>('github.com/ankurCES/office-ai/internal/pdf', 'Service', 'Optimize', [tabId, outputPath]),
  save: (tabId: string) =>
    call<OpResult>('github.com/ankurCES/office-ai/internal/pdf', 'Service', 'Save', [tabId]),
  close: (tabId: string) =>
    call<void>('github.com/ankurCES/office-ai/internal/pdf', 'Service', 'Close', [tabId]),
}

// ── Markdown Service ──
// Go: internal/markdown.Service
export const MarkdownService = {
  openFile: (tabId: string, path: string) =>
    call<Record<string, unknown>>('github.com/ankurCES/office-ai/internal/markdown', 'Service', 'OpenFile', [tabId, path]),
  newBlank: (tabId: string) =>
    call<Record<string, unknown>>('github.com/ankurCES/office-ai/internal/markdown', 'Service', 'NewBlank', [tabId]),
  updateContent: (tabId: string, content: string) =>
    call<void>('github.com/ankurCES/office-ai/internal/markdown', 'Service', 'UpdateContent', [tabId, content]),
  save: (tabId: string) =>
    call<Record<string, unknown>>('github.com/ankurCES/office-ai/internal/markdown', 'Service', 'Save', [tabId]),
  saveAs: (tabId: string, path: string) =>
    call<Record<string, unknown>>('github.com/ankurCES/office-ai/internal/markdown', 'Service', 'SaveAs', [tabId, path]),
  close: (tabId: string) =>
    call<void>('github.com/ankurCES/office-ai/internal/markdown', 'Service', 'Close', [tabId]),
}

// ── AI Provider Service ──
// Go: pkg/aiprovider.Provider
export const AIProviderService = {
  chat: (req: ChatRequest) =>
    call<ChatResponse>('github.com/ankurCES/office-ai/pkg/aiprovider', 'Provider', 'Chat', [req]),
  chatStream: (req: ChatRequest) =>
    call<string>('github.com/ankurCES/office-ai/pkg/aiprovider', 'Provider', 'ChatStream', [req], ''),
  getSettings: () =>
    call<AISettings>('github.com/ankurCES/office-ai/pkg/aiprovider', 'Provider', 'GetSettings', [], DEFAULT_AI_SETTINGS),
  updateSettings: (settings: Partial<AISettings>) =>
    call<void>('github.com/ankurCES/office-ai/pkg/aiprovider', 'Provider', 'UpdateSettings', [settings]),
}

// ── Agent Service ──
// Go: pkg/agentcore.Agent
export const AgentService = {
  run: (prompt: string, opts?: { skill?: string }) =>
    call<RunResult>('github.com/ankurCES/office-ai/pkg/agentcore', 'Agent', 'Run', [prompt, opts]),
  getHistory: () =>
    call<AIMessage[]>('github.com/ankurCES/office-ai/pkg/agentcore', 'Agent', 'GetHistory', [], []),
  clearHistory: () =>
    call<void>('github.com/ankurCES/office-ai/pkg/agentcore', 'Agent', 'ClearHistory'),
}

// ── I18n Service ──
// Go: pkg/i18n.Service
export const I18nService = {
  getLang: () =>
    call<string>('github.com/ankurCES/office-ai/pkg/i18n', 'Service', 'GetLang', [], 'en'),
  setLang: (lang: string) =>
    call<void>('github.com/ankurCES/office-ai/pkg/i18n', 'Service', 'SetLang', [lang]),
  translate: (key: string) =>
    call<string>('github.com/ankurCES/office-ai/pkg/i18n', 'Service', 'Translate', [key], key),
}

// ── Config Service ──
// Go: pkg/config.Manager
export const ConfigService = {
  get: () =>
    call<AppConfig>('github.com/ankurCES/office-ai/pkg/config', 'Manager', 'Get', []),
  getAI: () =>
    call<AIProviderConfig>('github.com/ankurCES/office-ai/pkg/config', 'Manager', 'GetAI', []),
  update: (partial: Record<string, unknown>) =>
    call<void>('github.com/ankurCES/office-ai/pkg/config', 'Manager', 'Update', [partial]),
  setAIKey: (key: string) =>
    call<void>('github.com/ankurCES/office-ai/pkg/config', 'Manager', 'SetAIKey', [key]),
  setAIProvider: (provider: string, model: string, apiKey: string) =>
    call<void>('github.com/ankurCES/office-ai/pkg/config', 'Manager', 'SetAIProvider', [provider, model, apiKey]),
}

// ── Wails Runtime ──
export const WailsRuntime = {
  openFileDialog: async (): Promise<string> => {
    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const w = window as any
      if (w?.go?.main?.App?.OpenFileDialog) {
        return w.go.main.App.OpenFileDialog()
      }
      // Wails runtime API fallback
      if (w?.runtime?.OpenFileDialog) {
        return w.runtime.OpenFileDialog({})
      }
    } catch {}
    return ''
  },
  saveFileDialog: async (): Promise<string> => {
    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const w = window as any
      if (w?.go?.main?.App?.SaveFileDialog) {
        return w.go.main.App.SaveFileDialog()
      }
      if (w?.runtime?.SaveFileDialog) {
        return w.runtime.SaveFileDialog({})
      }
    } catch {}
    return ''
  },
  windowSetTitle: (title: string) => {
    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const w = window as any
      w?.runtime?.WindowSetTitle?.(title)
    } catch {}
  },
  eventEmit: (name: string, data?: unknown) => {
    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const w = window as any
      w?.runtime?.EventsEmit?.(name, data)
    } catch {}
  },
  eventOn: (name: string, callback: (...args: unknown[]) => void) => {
    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const w = window as any
      w?.runtime?.EventsOn?.(name, callback)
    } catch {}
  },
}

// ── Type Definitions ──
export interface TabSummary {
  id: string
  kind: string
  title: string
  file_path: string
  dirty: boolean
}

export interface AppSettings {
  theme: string
  language: string
  auto_save: boolean
  auto_save_interval: number
  show_recent: boolean
}

export interface RecentFile {
  path: string
  title: string
  kind: string
  opened_at: string
  starred: boolean
  recent: boolean
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

export interface ParaInfo {
  index: number
  text: string
  bold: boolean
  italic: boolean
  alignment: string
}

export interface FindReplaceResult {
  matches: number
  replaced: number
}

export interface WorkbookState {
  file_path: string
  is_dirty: boolean
  title: string
  sheets: WorksheetInfo[]
  active_sheet: string
}

export interface WorksheetInfo {
  name: string
  index: number
  row_count: number
  col_count: number
}

export interface CellData {
  value: string
  formula: string
  type: string
}

export interface SlideOpenResult {
  success: boolean
  title: string
  file_path: string
  slide_count: number
  slides: SlideInfo[]
  error?: string
}

export interface SlideInfo {
  index: number
  title: string
  elements: SlideElement[]
}

export interface SlideElement {
  id: string
  type: string
  text: string
  x: number
  y: number
  width: number
  height: number
}

export interface DeckState {
  file_path: string
  is_dirty: boolean
  title: string
  slide_count: number
  slides: SlideInfo[]
}

export interface PDFState {
  file_path: string
  title: string
  page_count: number
  metadata: Record<string, string>
}

export interface OpResult {
  success: boolean
  message: string
  error?: string
}

export interface AISettings {
  provider: string
  api_key: string
  model: string
}

export interface AIProviderConfig {
  provider: string
  api_key: string
  model: string
  base_url: string
  max_tokens: number
  temperature: number
}

export interface AppConfig {
  ai: AIProviderConfig
  editor: EditorConfig
  window: WindowConfig
  language: string
  theme: string
  data_dir: string
  log_level: string
}

export interface EditorConfig {
  font_family: string
  font_size: number
  tab_size: number
  word_wrap: boolean
  line_numbers: boolean
  auto_save: boolean
  auto_save_delay_ms: number
  spell_check: boolean
}

export interface WindowConfig {
  width: number
  height: number
  x: number
  y: number
  maximized: boolean
  fullscreen: boolean
}

const DEFAULT_SETTINGS: AppSettings = {
  theme: 'system',
  language: 'en',
  auto_save: true,
  auto_save_interval: 30,
  show_recent: true,
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
