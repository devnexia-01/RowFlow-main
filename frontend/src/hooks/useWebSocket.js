import { useState, useEffect, useRef, useCallback } from 'react'

const useWebSocket = () => {
  const [lastMessage, setLastMessage] = useState(null)
  const [connectionStatus, setConnectionStatus] = useState('disconnected')
  const ws = useRef(null)
  const reconnectTimeout = useRef(null)

  const connect = useCallback(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/ws`

    ws.current = new WebSocket(wsUrl)

    ws.current.onopen = () => {
      setConnectionStatus('connected')
    }

    ws.current.onmessage = (event) => {
      const message = JSON.parse(event.data)
      setLastMessage(message)
    }

    ws.current.onerror = () => {
      setConnectionStatus('error')
    }

    ws.current.onclose = () => {
      setConnectionStatus('disconnected')
      reconnectTimeout.current = setTimeout(() => {
        connect()
      }, 3000)
    }
  }, [])

  useEffect(() => {
    connect()

    return () => {
      if (reconnectTimeout.current) {
        clearTimeout(reconnectTimeout.current)
      }
      if (ws.current) {
        ws.current.close()
      }
    }
  }, [connect])

  const sendMessage = useCallback((message) => {
    if (ws.current && ws.current.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify(message))
    }
  }, [])

  return {
    sendMessage,
    lastMessage,
    connectionStatus
  }
}

export default useWebSocket
