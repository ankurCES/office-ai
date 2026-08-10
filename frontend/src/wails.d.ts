// Type declarations for Wails runtime and Go bindings
// These map to the Go services bound in main.go

declare module '*/wailsjs/runtime/runtime' {
  export function EventsOn(eventName: string, callback: (...data: any[]) => void): () => void
  export function EventsEmit(eventName: string, ...data: any[]): void
  export function LogDebug(message: string): void
  export function LogInfo(message: string): void
  export function LogError(message: string): void
  export function WindowSetTitle(title: string): void
  export function WindowFullscreen(): void
  export function WindowUnfullscreen(): void
  export function WindowMinimise(): void
  export function WindowMaximise(): void
  export function WindowUnmaximise(): void
  export function WindowToggleMaximise(): void
  export function BrowserOpenURL(url: string): void
  export function Quit(): void
}

// Go service types
export interface DocState {
  file_path: string
  is_dirty: boolean
  title: string
  word_count: number
  page_count: number
  paragraphs?: string[]
}

export interface OpenFileResult {
  success: boolean
  file_path: string
  title: string
  error?: string
  state?: DocState
}

export interface SaveResult {
  success: boolean
  file_path: string
  error?: string
}

export interface SheetState {
  file_path: string
  is_dirty: boolean
  title: string
  sheet_names: string[]
  active_sheet: number
  row_count: number
  col_count: number
}

export interface SlideState {
  file_path: string
  is_dirty: boolean
  title: string
  slides: SlideData[]
  current_slide: number
}

export interface SlideData {
  index: number
  layout: string
  elements: SlideElement[]
}

export interface SlideElement {
  id: string
  kind: string
  x: number
  y: number
  w: number
  h: number
  text?: string
  src?: string
}

export interface PdfState {
  file_path: string
  title: string
  page_count: number
  current_page: number
}

export interface MarkdownState {
  file_path: string
  is_dirty: boolean
  title: string
  content: string
  word_count: number
}

export interface AIMessage {
  role: 'user' | 'assistant' | 'tool'
  text?: string
  tool_calls?: { id: string; name: string; input: Record<string, any> }[]
  tool_result?: { id: string; name: string; output: string; is_error?: boolean }
}

export interface AIRunResult {
  text: string
  cancelled: boolean
  turn_limit: boolean
}

export interface ProjectMeta {
  id: string
  name: string
  path: string
  file_type: string
  created_at: string
  updated_at: string
}

export interface TabInfo {
  id: string
  kind: 'home' | 'docs' | 'sheets' | 'slides' | 'pdf' | 'markdown'
  title: string
  filePath?: string
  isDirty?: boolean
  active: boolean
}
