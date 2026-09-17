import { createContext, useCallback, useContext, useMemo, useState } from 'react'
import '../components/Toast.css'

const ToastContext = createContext(null)

export function ToastProvider({ children }) {
  const [toasts, setToasts] = useState([])

  const removeToast = useCallback((id) => {
    setToasts((current) => current.filter((item) => item.id !== id))
  }, [])

  const addToast = useCallback((message, type = 'info', duration = 3500) => {
    const id = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`
    setToasts((current) => [...current, { id, message, type }])
    if (duration > 0) {
      setTimeout(() => {
        removeToast(id)
      }, duration)
    }
  }, [removeToast])

  const toast = useMemo(() => ({
    success: (msg, dur) => addToast(msg, 'success', dur),
    error: (msg, dur) => addToast(msg, 'error', dur),
    info: (msg, dur) => addToast(msg, 'info', dur),
  }), [addToast])

  return (
    <ToastContext.Provider value={toast}>
      {children}
      <div className="toast-container" aria-live="polite">
        {toasts.map((item) => (
          <div
            key={item.id}
            className={`toast-item toast-item--${item.type}`}
            role={item.type === 'error' ? 'alert' : 'status'}
          >
            <span className="toast-item__indicator" />
            <p className="toast-item__msg">{item.message}</p>
            <button
              type="button"
              className="toast-item__close"
              onClick={() => removeToast(item.id)}
              aria-label="Fechar notificação"
            >
              ×
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}

export function useToast() {
  const context = useContext(ToastContext)
  if (!context) {
    return {
      success: () => {},
      error: () => {},
      info: () => {},
    }
  }
  return context
}
