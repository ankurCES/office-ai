import './EditorToolbar.css'

interface ToolbarAction {
  id: string
  label: string
  icon?: string
  onClick: () => void
  disabled?: boolean
  active?: boolean
}

interface ToolbarGroup {
  id: string
  actions: ToolbarAction[]
}

interface EditorToolbarProps {
  groups: ToolbarGroup[]
  title?: string
}

export function EditorToolbar({ groups, title }: EditorToolbarProps) {
  return (
    <div className="editor-toolbar">
      {title && <span className="editor-toolbar-title">{title}</span>}
      {groups.map((group) => (
        <div key={group.id} className="editor-toolbar-group">
          {group.actions.map((action) => (
            <button
              key={action.id}
              className={`editor-toolbar-btn ${action.active ? 'active' : ''}`}
              onClick={action.onClick}
              disabled={action.disabled}
              title={action.label}
            >
              {action.icon || action.label}
            </button>
          ))}
        </div>
      ))}
    </div>
  )
}
