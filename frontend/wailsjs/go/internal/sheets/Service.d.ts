// Auto-generated Wails bindings for internal/sheets.Service
export function NewSpreadsheet(tabID: string): Promise<any>;
export function OpenFile(tabID: string, filePath: string): Promise<{ success: boolean; error?: string }>;
export function GetState(tabID: string): Promise<any>;
export function SetCell(tabID: string, sheet: number, row: number, col: number, value: string): Promise<void>;
export function GetCell(tabID: string, sheet: number, row: number, col: number): Promise<string>;
export function AddSheet(tabID: string, name: string): Promise<void>;
export function RemoveSheet(tabID: string, index: number): Promise<void>;
export function SetActiveSheet(tabID: string, index: number): Promise<void>;
export function Save(tabID: string): Promise<{ success: boolean; file_path: string; error?: string }>;
export function SaveAs(tabID: string, path: string): Promise<{ success: boolean; file_path: string; error?: string }>;
export function Close(tabID: string): Promise<void>;
