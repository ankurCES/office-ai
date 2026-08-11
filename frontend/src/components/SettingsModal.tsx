import { useState, useEffect } from 'react'
import { ShellService, AIService } from '../services/wails-bridge'
import './SettingsModal.css'

interface SettingsModalProps {
  onClose: () => void
}

type Theme = 'system' | 'light' | 'dark'

interface Settings {
  theme: Theme
  language: string
  aiProvider: string
  apiKey: string
  model: string
}

const AI_PROVIDERS = [
  { id: 'anthropic', label: 'Anthropic (Claude)' },
  { id: 'openai', label: 'OpenAI (GPT)' },
  { id: 'google', label: 'Google (Gemini)' },
  { id: 'ollama', label: 'Ollama (Local)' },
]

const LANGUAGES = [
  { id: 'en', label: 'English' },
  { id: 'zh', label: '中文' },
  { id: 'ja', label: '日本語' },
  { id: 'ko', label: '한국어' },
  { id: 'fr', label: 'Français' },
  { id: 'de', label: 'Deutsch' },
  { id: 'es', label: 'Español' },
  { id: 'pt', label: 'Português' },
  { id: 'ru', label: 'Русский' },
  { id: 'ar', label: 'العربية' },
  { id: 'hi', label: 'हिन्दी' },
  { id: 'th', label: 'ไทย' },
  { id: 'id', label: 'Bahasa Indonesia' },
]

export function SettingsModal({ onClose }: SettingsModalProps) {
  const [activeTab, setActiveTab] = useState<'general' | 'ai' | 'about'>('general')
  const [settings, setSettings] = useState<Settings>({
    theme: 'system',
    language: 'en',
    aiProvider: 'anthropic',
    apiKey: '',
    model: 'claude-sonnet-4-20250514',
  })

  useEffect(() => {
    const load = async () => {
      try {
        const [appSettings, aiSettings] = await Promise.all([
          ShellService.getSettings(),
          AIService.getSettings(),
        ])
        setSettings({
          theme: ((appSettings as any)?.theme as Theme) || 'system',
          language: (appSettings as any)?.language || 'en',
          aiProvider: (aiSettings as any)?.provider || 'anthropic',
          apiKey: (aiSettings as any)?.api_key || '',
          model: (aiSettings as any)?.model || 'claude-sonnet-4-20250514',
        })
      } catch {
        // Use defaults
      }
    }
    load()
  }, [])

  const update = <K extends keyof Settings>(key: K, value: Settings[K]) => {
    setSettings((prev) => ({ ...prev, [key]: value }))
  }

  const handleSave = async () => {
    try {
      await Promise.all([
        ShellService.updateSetting('theme', settings.theme),
        ShellService.updateSetting('language', settings.language),
        AIService.updateSettings({
          provider: settings.aiProvider,
          api_key: settings.apiKey,
          model: settings.model,
        }),
      ])
    } catch {
      // Settings save failed
    }
    if (settings.theme !== 'system') {
      document.documentElement.setAttribute('data-theme', settings.theme)
    } else {
      document.documentElement.removeAttribute('data-theme')
    }
    onClose()
  }

  return (
    <div className="settings-overlay" onClick={onClose}>
      <div className="settings-modal" onClick={(e) => e.stopPropagation()}>
        <div className="settings-header">
          <h2>Settings</h2>
          <button className="settings-close" onClick={onClose}>×</button>
        </div>

        <div className="settings-body">
          <div className="settings-tabs">
            <button
              className={`settings-tab ${activeTab === 'general' ? 'active' : ''}`}
              onClick={() => setActiveTab('general')}
            >General</button>
            <button
              className={`settings-tab ${activeTab === 'ai' ? 'active' : ''}`}
              onClick={() => setActiveTab('ai')}
            >AI Provider</button>
            <button
              className={`settings-tab ${activeTab === 'about' ? 'active' : ''}`}
              onClick={() => setActiveTab('about')}
            >About</button>
          </div>

          <div className="settings-content">
            {activeTab === 'general' && (
              <div className="settings-section">
                <label className="settings-field">
                  <span className="settings-label">Theme</span>
                  <select
                    value={settings.theme}
                    onChange={(e) => update('theme', e.target.value as Theme)}
                    className="settings-select"
                  >
                    <option value="system">System</option>
                    <option value="light">Light</option>
                    <option value="dark">Dark</option>
                  </select>
                </label>
                <label className="settings-field">
                  <span className="settings-label">Language</span>
                  <select
                    value={settings.language}
                    onChange={(e) => update('language', e.target.value)}
                    className="settings-select"
                  >
                    {LANGUAGES.map((lang) => (
                      <option key={lang.id} value={lang.id}>{lang.label}</option>
                    ))}
                  </select>
                </label>
              </div>
            )}

            {activeTab === 'ai' && (
              <div className="settings-section">
                <label className="settings-field">
                  <span className="settings-label">AI Provider</span>
                  <select
                    value={settings.aiProvider}
                    onChange={(e) => update('aiProvider', e.target.value)}
                    className="settings-select"
                  >
                    {AI_PROVIDERS.map((p) => (
                      <option key={p.id} value={p.id}>{p.label}</option>
                    ))}
                  </select>
                </label>
                <label className="settings-field">
                  <span className="settings-label">API Key</span>
                  <input
                    type="password"
                    value={settings.apiKey}
                    onChange={(e) => update('apiKey', e.target.value)}
                    className="settings-input"
                    placeholder="sk-..."
                  />
                </label>
                <label className="settings-field">
                  <span className="settings-label">Model</span>
                  <input
                    type="text"
                    value={settings.model}
                    onChange={(e) => update('model', e.target.value)}
                    className="settings-input"
                    placeholder="e.g. claude-sonnet-4-20250514"
                  />
                </label>
              </div>
            )}

            {activeTab === 'about' && (
              <div className="settings-section settings-about">
                <h3>Quill</h3>
                <p>Version 0.1.0</p>
                <p>A modern office suite with AI assistance, built with Go + Wails.</p>
                <p className="settings-credits">Powered by Wails, React, and Anthropic Claude.</p>
              </div>
            )}
          </div>
        </div>

        <div className="settings-footer">
          <button className="settings-btn settings-btn-cancel" onClick={onClose}>Cancel</button>
          <button className="settings-btn settings-btn-save" onClick={handleSave}>Save</button>
        </div>
      </div>
    </div>
  )
}
