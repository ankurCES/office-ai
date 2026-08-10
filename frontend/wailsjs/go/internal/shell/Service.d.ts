// Auto-generated Wails bindings for internal/shell.Service
export function GetTabs(): Promise<Array<{
  id: string;
  kind: string;
  title: string;
  file_path?: string;
  is_dirty: boolean;
  is_active: boolean;
}>>;

export function OpenTab(kind: string, filePath: string): Promise<string>;
export function ActivateTab(id: string): Promise<void>;
export function CloseTab(id: string): Promise<boolean>;
export function SetTabDirty(id: string, dirty: boolean): Promise<void>;
export function SetTabTitle(id: string, title: string): Promise<void>;
export function GetSettings(): Promise<{
  language: string;
  theme: string;
  onboard_done: boolean;
  default_save_dir?: string;
  update_channel: string;
}>;
export function UpdateSetting(key: string, value: any): Promise<void>;
export function GetRecentFiles(): Promise<Array<{
  path: string;
  name: string;
  kind: string;
  opened_at: string;
  is_starred: boolean;
}>>;
export function ToggleStarred(path: string): Promise<void>;
export function ClearRecents(): Promise<void>;
