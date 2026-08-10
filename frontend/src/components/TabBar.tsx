import { useRef } from 'react'
import type { TabInfo } from '../wails'
import { TAB_ICONS } from './icons/FileIcons'
import './TabBar.css'

interface TabBarProps {
  tabs: TabInfo[]
  onActivate: (id: string) => void
  onClose: (id: string) => void
  onNewTab: () => void
  onSettings: () => void
}

const KIND_COLORS: Record<string, string> = {
  docs: 'var(--color-docs)',
  sheets: 'var(--color-sheets)',
  slides: 'var(--color-slides)',
  pdf: 'var(--color-pdf)',
  markdown: 'var(--color-markdown)',
}

export function TabBar({ tabs, onActivate, onClose, onNewTab, onSettings }: TabBarProps) {
  const stripRef = useRef<HTMLDivElement>(null)

  return (
    <div className="tabbar" style={{ '--wails-draggable': 'drag' } as React.CSSProperties}>
      <div className="tabbar-strip" ref={stripRef}>
        {tabs.map((tab) => {
          const Icon = TAB_ICONS[tab.kind]
          const accentColor = KIND_COLORS[tab.kind]
          return (
            <button
              key={tab.id}
              className={`tabbar-tab ${tab.active ? 'active' : ''}`}
              style={tab.active && accentColor ? { borderBottomColor: accentColor } : undefined}
              onClick={() => onActivate(tab.id)}
              onAuxClick={(e) => {
                if (e.button === 1) onClose(tab.id)
              }}
            >
              {Icon && <Icon size={14} />}
              <span className="tabbar-tab-title">
                {tab.isDirty ? '● ' : ''}
                {tab.title}
              </span>
              {tab.id !== 'home' && (
                <span
                  className="tabbar-tab-close"
                  onClick={(e) => {
                    e.stopPropagation()
                    onClose(tab.id)
                  }}
                >
                  ×
                </span>
              )}
            </button>
          )
        })}
        <button className="tabbar-new" onClick={onNewTab} title="New tab">
          +
        </button>
      </div>
      <div className="tabbar-actions">
        <button className="tabbar-settings" onClick={onSettings} title="Settings">
          ⚙
        </button>
      </div>
    </div>
  )
}
