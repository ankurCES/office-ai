// @ts-check
// Auto-generated Wails bindings for internal/shell.Service

export function GetTabs() { return window['go']['internal']['shell']['Service']['GetTabs'](); }
export function OpenTab(kind, filePath) { return window['go']['internal']['shell']['Service']['OpenTab'](kind, filePath); }
export function ActivateTab(id) { return window['go']['internal']['shell']['Service']['ActivateTab'](id); }
export function CloseTab(id) { return window['go']['internal']['shell']['Service']['CloseTab'](id); }
export function SetTabDirty(id, dirty) { return window['go']['internal']['shell']['Service']['SetTabDirty'](id, dirty); }
export function SetTabTitle(id, title) { return window['go']['internal']['shell']['Service']['SetTabTitle'](id, title); }
export function GetSettings() { return window['go']['internal']['shell']['Service']['GetSettings'](); }
export function UpdateSetting(key, value) { return window['go']['internal']['shell']['Service']['UpdateSetting'](key, value); }
export function GetRecentFiles() { return window['go']['internal']['shell']['Service']['GetRecentFiles'](); }
export function ToggleStarred(path) { return window['go']['internal']['shell']['Service']['ToggleStarred'](path); }
export function ClearRecents() { return window['go']['internal']['shell']['Service']['ClearRecents'](); }
