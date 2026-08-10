// Auto-generated Wails bindings for internal/docs.Service
export function NewDocument(tabID: string): Promise<{
  file_path: string; is_dirty: boolean; title: string;
  word_count: number; page_count: number; paragraphs: string[];
}>;
export function OpenFile(tabID: string, filePath: string): Promise<{
  success: boolean; file_path: string; title: string; error?: string;
}>;
export function GetState(tabID: string): Promise<{
  file_path: string; is_dirty: boolean; title: string;
  word_count: number; page_count: number; paragraphs: string[];
} | null>;
export function UpdateContent(tabID: string, content: string): Promise<void>;
export function Save(tabID: string): Promise<{ success: boolean; file_path: string; error?: string }>;
export function SaveAs(tabID: string, path: string): Promise<{ success: boolean; file_path: string; error?: string }>;
export function Close(tabID: string): Promise<void>;
