import { MessageCircle } from 'lucide-react'
import { t, type Language } from '../../i18n/translations'

interface TelegramConfigModalProps {
  onClose: () => void
  language: Language
}

export function TelegramConfigModal({ onClose, language }: TelegramConfigModalProps) {
  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4 overflow-y-auto backdrop-blur-sm">
      <div
        className="rounded-2xl w-full max-w-lg relative my-8 shadow-2xl"
        style={{ background: 'linear-gradient(180deg, #1E2329 0%, #181A20 100%)' }}
      >
        <div className="flex items-center justify-between p-6 border-b border-white/10">
          <div className="flex items-center gap-2">
            <MessageCircle className="w-6 h-6" style={{ color: '#2AABEE' }} />
            <h3 className="text-xl font-bold" style={{ color: '#EAECEF' }}>
              {t('telegram.botSetup', language)}
            </h3>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="p-2 rounded-lg hover:bg-white/10 transition-colors"
            style={{ color: '#848E9C' }}
          >
            ✕
          </button>
        </div>

        <div className="px-6 py-8 space-y-4">
          <div
            className="p-4 rounded-xl"
            style={{ background: 'rgba(42, 171, 238, 0.1)', border: '1px solid rgba(42, 171, 238, 0.25)' }}
          >
            <div className="text-sm font-semibold mb-2" style={{ color: '#2AABEE' }}>
              Telegram integration is frozen in Phase 1
            </div>
            <div className="text-xs leading-6" style={{ color: '#848E9C' }}>
              The Telegram bot setup flow remains in the repository for future reuse,
              but it is intentionally disabled from the active product path in this phase.
            </div>
          </div>

          <div className="flex justify-end">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 rounded-xl text-sm font-semibold"
              style={{ background: '#2B3139', color: '#EAECEF' }}
            >
              {t('close', language) || 'Close'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
