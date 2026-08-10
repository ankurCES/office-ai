import { useState, useCallback, useEffect, useRef } from 'react'
import { SheetsService } from '../../services/wails-bridge'
import { EditorToolbar } from './EditorToolbar'
import './SheetsEditor.css'

interface SheetsEditorProps {
  tabId: string
  filePath?: string
  onTitleChange: (title: string) => void
  onDirtyChange: (dirty: boolean) => void
}

interface CellData {
  value: string
  formula?: string
}

const INITIAL_ROWS = 100
const INITIAL_COLS = 26

function colLabel(index: number): string {
  let label = ''
  let n = index
  while (n >= 0) {
    label = String.fromCharCode(65 + (n % 26)) + label
    n = Math.floor(n / 26) - 1
  }
  return label
}

export function SheetsEditor({ tabId, filePath, onTitleChange, onDirtyChange }: SheetsEditorProps) {
  const [cells, setCells] = useState<Record<string, CellData>>({})
  const [activeCell, setActiveCell] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')
  const [sheetNames, setSheetNames] = useState(['Sheet1'])
  const [activeSheet, setActiveSheet] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)

  // Open file or create blank via Go backend
  useEffect(() => {
    const init = async () => {
      try {
        let result: Record<string, unknown>
        if (filePath) {
          result = await SheetsService.openFile(tabId, filePath)
        } else {
          result = await SheetsService.newBlank(tabId)
        }
        if (result?.title) onTitleChange(result.title as string)
        else onTitleChange(filePath?.split(/[/\\]/).pop() || 'Untitled Spreadsheet')
        if (result?.sheet_names) setSheetNames(result.sheet_names as string[])

        // Load visible cell range
        const range = await SheetsService.getCellRange(tabId, 0, 0, 49, 25)
        if (range?.length) {
          const loaded: Record<string, CellData> = {}
          for (const cv of range) {
            const key = `${colLabel(cv.col)}${cv.row + 1}`
            loaded[key] = { value: cv.value, formula: cv.formula }
          }
          setCells(loaded)
        }
      } catch {
        onTitleChange(filePath?.split(/[/\\]/).pop() || 'Untitled Spreadsheet')
      }
    }
    init()
    return () => { SheetsService.close(tabId) }
  }, [filePath, tabId, onTitleChange])

  const cellKey = (row: number, col: number) => `${colLabel(col)}${row + 1}`

  const handleCellClick = useCallback((row: number, col: number) => {
    const key = cellKey(row, col)
    setActiveCell(key)
    const cell = cells[key]
    setEditValue(cell?.formula || cell?.value || '')
    inputRef.current?.focus()
  }, [cells])

  const commitCell = useCallback(async () => {
    if (!activeCell) return
    const isFormula = editValue.startsWith('=')
    setCells((prev) => ({
      ...prev,
      [activeCell]: {
        value: isFormula ? '...' : editValue,
        formula: isFormula ? editValue : undefined,
      },
    }))
    onDirtyChange(true)

    // Persist to Go backend
    const match = activeCell.match(/^([A-Z]+)(\d+)$/)
    if (match) {
      const colIdx = match[1].split('').reduce((acc, ch) => acc * 26 + ch.charCodeAt(0) - 64, 0) - 1
      const rowIdx = parseInt(match[2]) - 1
      try {
        await SheetsService.setCellValue(tabId, rowIdx, colIdx, editValue)
        if (isFormula) {
          await SheetsService.recalc(tabId)
          // Refresh cell range after recalc
          const range = await SheetsService.getCellRange(tabId, 0, 0, 49, 25)
          if (range?.length) {
            const loaded: Record<string, CellData> = {}
            for (const cv of range) {
              loaded[`${colLabel(cv.col)}${cv.row + 1}`] = { value: cv.value, formula: cv.formula }
            }
            setCells(loaded)
          }
        }
      } catch {
        // Fallback: keep local state
      }
    }
  }, [activeCell, editValue, onDirtyChange, tabId])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') {
        commitCell()
        if (activeCell) {
          const match = activeCell.match(/^([A-Z]+)(\d+)$/)
          if (match) {
            const col = match[1]
            const row = parseInt(match[2])
            const nextKey = `${col}${row + 1}`
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

  const toolbarGroups = [
    {
      id: 'file',
      actions: [{ id: 'save', label: 'Save', icon: '💾', onClick: handleSave }],
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
              {Array.from({ length: INITIAL_COLS }, (_, c) => (
                <th key={c} className="sheets-col-header">{colLabel(c)}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {Array.from({ length: INITIAL_ROWS }, (_, r) => (
              <tr key={r}>
                <td className="sheets-row-header">{r + 1}</td>
                {Array.from({ length: INITIAL_COLS }, (_, c) => {
                  const key = cellKey(r, c)
                  const isActive = activeCell === key
                  return (
                    <td
                      key={c}
                      className={`sheets-cell ${isActive ? 'active' : ''}`}
                      onClick={() => handleCellClick(r, c)}
                    >
                      {cells[key]?.value || ''}
                    </td>
                  )
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="sheets-sheet-tabs">
        {sheetNames.map((name, i) => (
          <button
            key={i}
            className={`sheets-sheet-tab ${i === activeSheet ? 'active' : ''}`}
            onClick={() => setActiveSheet(i)}
          >
            {name}
          </button>
        ))}
        <button className="sheets-add-sheet" title="Add sheet">+</button>
      </div>
    </div>
  )
}
