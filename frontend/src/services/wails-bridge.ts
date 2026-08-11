/**
 * Wails Bridge — thin typed wrappers around Wails-generated bindings.
 *
 * Wails generates JS at frontend/wailsjs/go/<pkg>/Service.js that call
 * window['go']['<pkg>']['Service']['Method'](). We re-export those
 * with our domain types so components import from one place.
 */

// ── Wails-generated binding imports ────────────────────────────────
import * as DocsBinding from '../../wailsjs/go/docs/Service'
import * as SheetsBinding from '../../wailsjs/go/sheets/Service'
import * as SlidesBinding from '../../wailsjs/go/slides/Service'
import * as PdfBinding from '../../wailsjs/go/pdf/Service'
import * as MarkdownBinding from '../../wailsjs/go/markdown/Service'
import * as ShellBinding from '../../wailsjs/go/shell/Service'
import * as AppBinding from '../../wailsjs/go/main/App'
import * as AIBinding from '../../wailsjs/go/aiprovider/Service'

// ── Wails runtime ──────────────────────────────────────────────────
import { WindowSetTitle } from '../../wailsjs/runtime/runtime'

// ── Domain types ───────────────────────────────────────────────────

export interface TabSummary {
  id: string
  kind: string
  title: string
  file_path?: string
  is_dirty?: boolean
  active?: boolean
}

export interface RecentFile {
  path: string
  name: string
  title: string
  kind: string
  is_starred: boolean
  starred: boolean
  opened_at: any
}

export interface OpenFileResult {
  success: boolean
  file_path?: string
  title?: string
  error?: string
  word_count?: number
  char_count?: number
  page_count?: number
  paragraphs?: ParaInfo[]
}

export interface ParaInfo {
  index: number
  text: string
  style?: string
  alignment?: string
  is_bold?: boolean
  is_italic?: boolean
}

export interface DocState {
  file_path: string
  is_dirty: boolean
  title: string
  word_count: number
  char_count: number
  page_count: number
  paragraphs?: ParaInfo[]
}

export interface SaveResult {
  success: boolean
  file_path?: string
  error?: string
}

export interface FindReplaceResult {
  count: number
  success: boolean
  error?: string
}

export interface CellData {
  row: number
  col: number
  value: string
  formula?: string
  type: string
}

export interface WorksheetInfo {
  id: number
  name: string
  index: number
  row_count: number
  column_count: number
  hidden: boolean
}

export interface WorkbookState {
  file_path: string
  title: string
  is_dirty: boolean
  sheets: WorksheetInfo[]
  active_sheet: string
}

export interface SlideElement {
  id: string
  kind: string
  x: number
  y: number
  w: number
  h: number
  text?: string
  bold?: boolean
  italic?: boolean
  align?: string
  ph_type?: string
  image_id?: string
}

export interface SlideInfo {
  index: number
  title?: string
  elements: SlideElement[]
}

export interface DeckState {
  file_path: string
  is_dirty: boolean
  title: string
  slide_count: number
  slides: SlideInfo[]
  width: number
  height: number
}

export interface SlideOpenResult {
  success: boolean
  file_path?: string
  title?: string
  error?: string
  slide_count?: number
  slides?: SlideInfo[]
  width?: number
  height?: number
}

export interface PDFState {
  file_path: string
  title: string
  is_dirty: boolean
  page_count: number
  pages: any[]
  meta: any
}

export interface AppSettings {
  theme: string
  language: string
  onboard_done: boolean
  default_save_dir?: string
  update_channel: string
}

export interface AISettings {
  provider: string
  api_key: string
  model: string
}

export interface AIMessage {
  role: 'user' | 'assistant' | 'tool'
  text?: string
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

// ── Shell Service ──────────────────────────────────────────────────

export const ShellService = {
  getTabs: () => ShellBinding.GetTabs() as any as Promise<TabSummary[]>,
  openTab: (kind: string, filePath: string) => ShellBinding.OpenTab(kind, filePath),
  activateTab: (id: string) => ShellBinding.ActivateTab(id),
  closeTab: (id: string) => ShellBinding.CloseTab(id),
  setTabDirty: (id: string, dirty: boolean) => ShellBinding.SetTabDirty(id, dirty),
  setTabTitle: (id: string, title: string) => ShellBinding.SetTabTitle(id, title),
  getSettings: () => ShellBinding.GetSettings() as any as Promise<AppSettings>,
  updateSetting: (key: string, value: unknown) => ShellBinding.UpdateSetting(key, value),
  getRecentFiles: () => ShellBinding.GetRecentFiles() as any as Promise<RecentFile[]>,
  toggleStarred: (path: string) => ShellBinding.ToggleStarred(path),
  removeRecent: (path: string) => ShellBinding.RemoveRecent(path),
}

// ── Docs Service ───────────────────────────────────────────────────

export const DocsService = {
  openFile: (tabId: string, path: string) =>
    DocsBinding.OpenFile(tabId, path) as any as Promise<OpenFileResult>,
  newBlank: (tabId: string) =>
    DocsBinding.NewBlank(tabId) as any as Promise<OpenFileResult>,
  save: (tabId: string) =>
    DocsBinding.Save(tabId) as any as Promise<SaveResult>,
  saveAs: (tabId: string, path: string) =>
    DocsBinding.SaveAs(tabId, path) as any as Promise<SaveResult>,
  getState: (tabId: string) =>
    DocsBinding.GetState(tabId) as any as Promise<DocState | null>,
  isDirty: (tabId: string) =>
    DocsBinding.IsDirty(tabId) as any as Promise<boolean>,
  close: (tabId: string) =>
    DocsBinding.Close(tabId),
  updateParagraph: (tabId: string, index: number, text: string, bold: boolean, italic: boolean, align: string) =>
    DocsBinding.UpdateParagraph(tabId, index, text, bold, italic, align),
  insertParagraph: (tabId: string, index: number, text: string) =>
    DocsBinding.InsertParagraph(tabId, index, text),
  deleteParagraph: (tabId: string, index: number) =>
    DocsBinding.DeleteParagraph(tabId, index),
  findReplace: (tabId: string, search: string, replace: string, replaceAll: boolean) =>
    DocsBinding.FindReplace(tabId, search, replace, replaceAll) as any as Promise<FindReplaceResult>,
  find: (tabId: string, search: string) =>
    DocsBinding.Find(tabId, search) as any as Promise<number[]>,
  undo: (tabId: string) =>
    DocsBinding.Undo(tabId),
  redo: (tabId: string) =>
    DocsBinding.Redo(tabId),
  getParagraphs: (tabId: string) =>
    DocsBinding.GetParagraphs(tabId) as any as Promise<ParaInfo[]>,
  getHTMLPreview: (tabId: string) =>
    DocsBinding.GetHTMLPreview(tabId) as any as Promise<string>,
  exportHTML: (tabId: string, path: string) =>
    DocsBinding.ExportHTML(tabId, path) as any as Promise<SaveResult>,
  exportText: (tabId: string, path: string) =>
    DocsBinding.ExportText(tabId, path) as any as Promise<SaveResult>,
  startAutosave: (tabId: string, intervalSec: number) =>
    DocsBinding.StartAutosave(tabId, intervalSec),
}

// ── Sheets Service ─────────────────────────────────────────────────

export const SheetsService = {
  openFile: (tabId: string, path: string) =>
    SheetsBinding.OpenFile(tabId, path) as any as Promise<Record<string, unknown>>,
  newWorkbook: (tabId: string) =>
    SheetsBinding.NewWorkbook(tabId) as any as Promise<Record<string, unknown>>,
  getState: (tabId: string) =>
    SheetsBinding.GetState(tabId) as any as Promise<WorkbookState | null>,
  getCellRange: (tabId: string, sheetName: string, startRow: number, startCol: number, rows: number, cols: number) =>
    SheetsBinding.GetCellRange(tabId, sheetName, startRow, startCol, rows, cols) as any as Promise<CellData[][]>,
  getAllRows: (tabId: string, sheetName: string) =>
    SheetsBinding.GetAllRows(tabId, sheetName) as any as Promise<string[][]>,
  setCellValue: (tabId: string, sheetName: string, cellRef: string, value: string) =>
    SheetsBinding.SetCellValue(tabId, sheetName, cellRef, value) as any as Promise<SaveResult>,
  setCellFormula: (tabId: string, sheetName: string, cellRef: string, formula: string) =>
    SheetsBinding.SetCellFormula(tabId, sheetName, cellRef, formula) as any as Promise<SaveResult>,
  addSheet: (tabId: string, name: string) =>
    SheetsBinding.AddSheet(tabId, name) as any as Promise<SaveResult>,
  deleteSheet: (tabId: string, name: string) =>
    SheetsBinding.DeleteSheet(tabId, name) as any as Promise<SaveResult>,
  renameSheet: (tabId: string, oldName: string, newName: string) =>
    SheetsBinding.RenameSheet(tabId, oldName, newName) as any as Promise<SaveResult>,
  insertRow: (tabId: string, sheetName: string, row: number) =>
    SheetsBinding.InsertRow(tabId, sheetName, row) as any as Promise<SaveResult>,
  insertCol: (tabId: string, sheetName: string, col: number) =>
    SheetsBinding.InsertCol(tabId, sheetName, col) as any as Promise<SaveResult>,
  deleteRow: (tabId: string, sheetName: string, row: number) =>
    SheetsBinding.DeleteRow(tabId, sheetName, row) as any as Promise<SaveResult>,
  mergeCells: (tabId: string, sheetName: string, startCell: string, endCell: string) =>
    SheetsBinding.MergeCells(tabId, sheetName, startCell, endCell) as any as Promise<SaveResult>,
  getMergedCells: (tabId: string, sheetName: string) =>
    SheetsBinding.GetMergedCells(tabId, sheetName),
  setColumnWidth: (tabId: string, sheetName: string, startCol: string, endCol: string, width: number) =>
    SheetsBinding.SetColumnWidth(tabId, sheetName, startCol, endCol, width) as any as Promise<SaveResult>,
  setRowHeight: (tabId: string, sheetName: string, row: number, height: number) =>
    SheetsBinding.SetRowHeight(tabId, sheetName, row, height) as any as Promise<SaveResult>,
  setAutoFilter: (tabId: string, sheetName: string, range: string) =>
    SheetsBinding.SetAutoFilter(tabId, sheetName, range) as any as Promise<SaveResult>,
  exportCSV: (tabId: string, sheetName: string, path: string) =>
    SheetsBinding.ExportCSV(tabId, sheetName, path) as any as Promise<SaveResult>,
  save: (tabId: string) =>
    SheetsBinding.Save(tabId) as any as Promise<SaveResult>,
  saveAs: (tabId: string, path: string) =>
    SheetsBinding.SaveAs(tabId, path) as any as Promise<SaveResult>,
  close: (tabId: string) =>
    SheetsBinding.Close(tabId),
}

// ── Slides Service ─────────────────────────────────────────────────

export const SlidesService = {
  openFile: (tabId: string, path: string) =>
    SlidesBinding.OpenFile(tabId, path) as any as Promise<SlideOpenResult>,
  newBlank: (tabId: string) =>
    SlidesBinding.NewBlank(tabId) as any as Promise<SlideOpenResult>,
  getState: (tabId: string) =>
    SlidesBinding.GetState(tabId) as any as Promise<DeckState | null>,
  getSlides: (tabId: string) =>
    SlidesBinding.GetSlides(tabId) as any as Promise<SlideInfo[]>,
  addSlide: (tabId: string) =>
    SlidesBinding.AddSlide(tabId) as any as Promise<SlideInfo>,
  deleteSlide: (tabId: string, index: number) =>
    SlidesBinding.DeleteSlide(tabId, index) as any as Promise<boolean>,
  duplicateSlide: (tabId: string, index: number) =>
    SlidesBinding.DuplicateSlide(tabId, index) as any as Promise<SlideInfo>,
  moveSlide: (tabId: string, fromIdx: number, toIdx: number) =>
    SlidesBinding.MoveSlide(tabId, fromIdx, toIdx) as any as Promise<boolean>,
  updateElement: (tabId: string, slideIdx: number, elemId: string, text: string) =>
    SlidesBinding.UpdateElement(tabId, slideIdx, elemId, text) as any as Promise<boolean>,
  getSlideData: (tabId: string, index: number) =>
    SlidesBinding.GetSlideData(tabId, index) as any as Promise<string>,
  getSlideSVG: (tabId: string, index: number) =>
    SlidesBinding.GetSlideSVG(tabId, index) as any as Promise<string>,
  isDirty: (tabId: string) =>
    SlidesBinding.IsDirty(tabId) as any as Promise<boolean>,
  undo: (tabId: string) =>
    SlidesBinding.Undo(tabId) as any as Promise<boolean>,
  redo: (tabId: string) =>
    SlidesBinding.Redo(tabId) as any as Promise<boolean>,
  save: (tabId: string) =>
    SlidesBinding.Save(tabId) as any as Promise<SaveResult>,
  saveAs: (tabId: string, path: string) =>
    SlidesBinding.SaveAs(tabId, path) as any as Promise<SaveResult>,
  close: (tabId: string) =>
    SlidesBinding.Close(tabId),
  exportHTML: (tabId: string, path: string) =>
    SlidesBinding.ExportHTML(tabId, path) as any as Promise<SaveResult>,
}

// ── PDF Service ────────────────────────────────────────────────────

export const PdfService = {
  openFile: (tabId: string, path: string) =>
    PdfBinding.OpenFile(tabId, path),
  getState: (tabId: string) =>
    PdfBinding.GetState(tabId) as any as Promise<PDFState | null>,
  extractText: (tabId: string) =>
    PdfBinding.ExtractText(tabId) as any as Promise<string>,
  extractPages: (tabId: string, pages: number[], outPath: string) =>
    PdfBinding.ExtractPages(tabId, pages, outPath),
  mergeFiles: (paths: string[], outPath: string) =>
    PdfBinding.MergeFiles(paths, outPath),
  addWatermark: (tabId: string, text: string, outPath: string) =>
    PdfBinding.AddWatermark(tabId, text, outPath),
  rotatePage: (tabId: string, page: number, degrees: number) =>
    PdfBinding.RotatePages(tabId, degrees, [page]),
  close: (tabId: string) =>
    PdfBinding.Close(tabId),
}

// ── Markdown Service ───────────────────────────────────────────────

export const MarkdownService = {
  openFile: (tabId: string, path: string) =>
    MarkdownBinding.OpenFile(tabId, path) as any as Promise<Record<string, unknown>>,
  newBlank: (tabId: string) =>
    MarkdownBinding.NewBlank(tabId) as any as Promise<Record<string, unknown>>,
  updateContent: (tabId: string, content: string) =>
    MarkdownBinding.UpdateContent(tabId, content),
  save: (tabId: string) =>
    MarkdownBinding.Save(tabId) as any as Promise<Record<string, unknown>>,
  saveAs: (tabId: string, path: string) =>
    MarkdownBinding.SaveAs(tabId, path) as any as Promise<Record<string, unknown>>,
  close: (tabId: string) =>
    MarkdownBinding.Close(tabId),
}

// ── AI Provider Service ────────────────────────────────────────────

export const AIService = {
  chat: (request: ChatRequest) =>
    AIBinding.Chat(null as any, request as any) as any as Promise<ChatResponse>,
  getSettings: () =>
    AIBinding.GetSettings() as any as Promise<AISettings>,
  updateSettings: (settings: AISettings) =>
    AIBinding.UpdateSettings(settings as any),
  listModels: () =>
    AIBinding.ListModels(null as any) as any as Promise<string[]>,
}

// ── Wails Runtime ──────────────────────────────────────────────────

export const WailsRuntime = {
  openFileDialog: () => AppBinding.OpenFileDialog(),
  saveFileDialog: () => AppBinding.SaveFileDialog(),
  windowSetTitle: (title: string) => WindowSetTitle(title),
}
