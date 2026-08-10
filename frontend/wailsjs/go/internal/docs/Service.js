export function NewDocument(tabID) { return window['go']['internal']['docs']['Service']['NewDocument'](tabID); }
export function OpenFile(tabID, filePath) { return window['go']['internal']['docs']['Service']['OpenFile'](tabID, filePath); }
export function GetState(tabID) { return window['go']['internal']['docs']['Service']['GetState'](tabID); }
export function UpdateContent(tabID, content) { return window['go']['internal']['docs']['Service']['UpdateContent'](tabID, content); }
export function Save(tabID) { return window['go']['internal']['docs']['Service']['Save'](tabID); }
export function SaveAs(tabID, path) { return window['go']['internal']['docs']['Service']['SaveAs'](tabID, path); }
export function Close(tabID) { return window['go']['internal']['docs']['Service']['Close'](tabID); }
