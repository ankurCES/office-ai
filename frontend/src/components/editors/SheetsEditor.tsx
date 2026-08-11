import { useState, useCallback, useEffect, useRef } from 'react'
import { SheetsService } from '../../services/wails-bridge'
import type { CellData } from '../../services/wails-bridge'
import { EditorToolbar } from './EditorToolbar'
import './SheetsEditor.css'

interface SheetsEditorProps {
  tabId: string
  filePath?: string
  onTitleChange: (title: string) => void
  onDirtyChange: (dirty: boolean) => void
}

const VISIBLE_ROWS = 50
const VISIBLE_COLS = 26

function colLabel(index: number): string {
  let label = ''
  let n = index
  while (n >= 0) {
    label = String.fromCharCode(65 + (n % 26)) + label
    n = Math.floor(n / 26) - 1
  }
  return label
}

function cellRef(row: number, col: number): string {
  return `${colLabel(col)}${row + 1}`
}

export function SheetsEditor({ tabId, filePath, onTitleChange, onDirtyChange }: SheetsEditorProps) {
  const [cells, setCells] = useState<Record<string, CellData>>({})
  const [activeCell, setActiveCell] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')
  const [sheetNames, setSheetNames] = useState<string[]>(['Sheet1'])
  const [activeSheet, setActiveSheet] = useState('Sheet1')
  const inputRef = useRef<HTMLInputElement>(null)

  // Load cells from Go backend for the active sheet
  const loadCells = useCallback(async (sheetName: string) => {
    try {
      const range = await SheetsService.getCellRange(tabId, sheetName, 0, 0, VISIBLE_ROWS, VISIBLE_COLS)
      if (range?.length) {
        const loaded: Record<string, CellData> = {}
        range.forEach((row, r) => {
          row?.forEach((cell, c) => {
            if (cell && (cell.value || cell.formula)) {
              loaded[cellRef(r, c)] = cell
            }
          })
        })
        setCells(loaded)
      }
    } catch {
      // Fallback: keep local state
    }
  }, [tabId])

  // Open file or create blank
  useEffect(() => {
    const init = async () => {
      try {
        let result: Record<string, unknown>
        if (filePath) {
          result = await SheetsService.openFile(tabId, filePath)
        } else {
          result = await SheetsService.newWorkbook(tabId)
        }
        if (result?.title) onTitleChange(result.title as string)
        else onTitleChange(filePath?.split(/[/\\]/).pop() || 'Untitled Spreadsheet')

        const sheets = result?.sheet_names as string[] | undefined
        if (sheets?.length) {
          setSheetNames(sheets)
          setActiveSheet(sheets[0])
          await loadCells(sheets[0])
        } else {
          await loadCells('Sheet1')
        }
      } catch {
        onTitleChange(filePath?.split(/[/\\]/).pop() || 'Untitled Spreadsheet')
      }
    }
    init()
    return () => { SheetsService.close(tabId) }
  }, [filePath, tabId, onTitleChange, loadCells])

  const handleCellClick = useCallback((row: number, col: number) => {
    const ref = cellRef(row, col)
    setActiveCell(ref)
    const cell = cells[ref]
    setEditValue(cell?.formula || cell?.value || '')
    inputRef.current?.focus()
  }, [cells])

  const commitCell = useCallback(async () => {
    if (!activeCell) return
    const isFormula = editValue.startsWith('=')
    setCells((prev) => ({
      ...prev,
      [activeCell]: {
        row: 0,
        col: 0,
        value: isFormula ? '...' : editValue,
        formula: isFormula ? editValue : '',
        type: isFormula ? 'formula' : 'string',
      },
    }))
    onDirtyChange(true)

    try {
      if (isFormula) {
        await SheetsService.setCellFormula(tabId, activeSheet, activeCell, editValue)
      } else {
        await SheetsService.setCellValue(tabId, activeSheet, activeCell, editValue)
      }
      // Refresh cells after formula evaluation
      if (isFormula) {
        await loadCells(activeSheet)
      }
    } catch {
      // Keep local state on error
    }
  }, [activeCell, editValue, onDirtyChange, tabId, activeSheet, loadCells])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') {
        commitCell()
        if (activeCell) {
          const match = activeCell.match(/^([A-Z]+)(\d+)$/)
          if (match) {
            const nextKey = `${match[1]}${parseInt(match[2]) + 1}`
            setActiveCell(nextKey)
            setEditValue(cells[nextKey]?.formula || cells[nextKey]?.value || '')
          }
        }
      } else if (e.key === 'Tab') {
        e.preventDefault()
        commitCell()
      } else if (e.key === 'Escape') {
        setEditValue(cells[activeCell!]?.value || '')
      }
    },
    [activeCell, cells, commitCell],
  )

  const handleSave = useCallback(async () => {
    try {
      const result = await SheetsService.save(tabId)
      if (result?.success) onDirtyChange(false)
    } catch (err) {
      console.error('Save failed:', err)
    }
  }, [tabId, onDirtyChange])

  const handleSwitchSheet = useCallback(async (name: string) => {
    setActiveSheet(name)
    setActiveCell(null)
    setEditValue('')
    await loadCells(name)
  }, [loadCells])

  const handleAddSheet = useCallback(async () => {
    const name = `Sheet${sheetNames.length + 1}`
    try {
      const result = await SheetsService.addSheet(tabId, name)
      if (result?.success) {
        setSheetNames(prev => [...prev, name])
        handleSwitchSheet(name)
        onDirtyChange(true)
      }
    } catch {
      // Fallback: add locally
      setSheetNames(prev => [...prev, name])
    }
  }, [tabId, sheetNames.length, handleSwitchSheet, onDirtyChange])

  const handleInsertRow = useCallback(async () => {
    if (!activeCell) return
    const match = activeCell.match(/^[A-Z]+(\d+)$/)
    if (match) {
      const row = parseInt(match[1])
      await SheetsService.insertRow(tabId, activeSheet, row)
      await loadCells(activeSheet)
      onDirtyChange(true)
    }
  }, [activeCell, tabId, activeSheet, loadCells, onDirtyChange])

  const handleInsertCol = useCallback(async () => {
    if (!activeCell) return
    const match = activeCell.match(/^([A-Z]+)/)
    if (match) {
      const col = match[1].split('').reduce((acc, ch) => acc * 26 + ch.charCodeAt(0) - 64, 0) - 1
      await SheetsService.insertCol(tabId, activeSheet, col)
      await loadCells(activeSheet)
      onDirtyChange(true)
    }
  }, [activeCell, tabId, activeSheet, loadCells, onDirtyChange])

  const toolbarGroups = [
    {
      id: 'file',
      actions: [{ id: 'save', label: 'Save', icon: '💾', onClick: handleSave }],
    },
    {
      id: 'edit',
      actions: [
        { id: 'insert-row', label: 'Insert Row', icon: '➕↔', onClick: handleInsertRow },
        { id: 'insert-col', label: 'Insert Column', icon: '➕↕', onClick: handleInsertCol },
      ],
    },
    {
      id: 'format',
      actions: [
        { id: 'bold', label: 'Bold', icon: 'B', onClick: () => {} },
        { id: 'italic', label: 'Italic', icon: 'I', onClick: () => {} },
      ],
    },
    {
      id: 'data',
      actions: [
        { id: 'sort-asc', label: 'Sort A→Z', icon: '↑', onClick: () => {} },
        { id: 'sort-desc', label: 'Sort Z→A', icon: '↓', onClick: () => {} },
        { id: 'filter', label: 'Filter', icon: '⏚', onClick: () => {} },
      ],
    },
  ]

  return (
    <div className="sheets-editor">
      <EditorToolbar groups={toolbarGroups} title="Spreadsheet" />
      <div className="sheets-formula-bar">
        <span className="sheets-cell-ref">{activeCell || ''}</span>
        <input
          ref={inputRef}
          className="sheets-formula-input"
          value={editValue}
          onChange={(e) => setEditValue(e.target.value)}
          onKeyDown={handleKeyDown}
          onBlur={commitCell}
          placeholder="Enter value or formula..."
        />
      </div>
      <div className="sheets-grid-wrapper">
        <table className="sheets-grid">
          <thead>
            <tr>
              <th className="sheets-row-header" />
              {Array.from({ length: VISIBLE_COLS }, (_, c) => (
                <th key={c} className="sheets-col-header">{colLabel(c)}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {Array.from({ length: VISIBLE_ROWS }, (_, r) => (
              <tr key={r}>
                <td className="sheets-row-header">{r + 1}</td>
                {Array.from({ length: VISIBLE_COLS }, (_, c) => {
                  const ref = cellRef(r, c)
                  const isActive = activeCell === ref
                  return (
                    <td
                      key={c}
                      className={`sheets-cell ${isActive ? 'active' : ''}`}
                      onClick={() => handleCellClick(r, c)}
                    >
                      {cells[ref]?.value || ''}
                    </td>
                  )
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="sheets-sheet-tabs">
        {sheetNames.map((name) => (
          <button
            key={name}
            className={`sheets-sheet-tab ${name === activeSheet ? 'active' : ''}`}
            onClick={() => handleSwitchSheet(name)}
          >
            {name}
          </button>
        ))}
        <button className="sheets-add-sheet" title="Add sheet" onClick={handleAddSheet}>+</button>
      </div>
    </div>
  )
}
