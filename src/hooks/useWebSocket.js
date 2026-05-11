import { useState, useEffect, useCallback, useRef } from 'react';

const useWebSocket = (url) => {
  const [connectionStatus, setConnectionStatus] = useState('disconnected');
  const [lastMessage, setLastMessage] = useState(null);
  const ws = useRef(null);

  useEffect(() => {
    if (!url) return;
    
    try {
      ws.current = new WebSocket(url);

      ws.current.onopen = () => {
        setConnectionStatus('connected');
        console.log('WebSocket connected');
      };

      ws.current.onclose = () => {
        setConnectionStatus('disconnected');
        console.log('WebSocket disconnected');
      };

      ws.current.onerror = (error) => {
        setConnectionStatus('error');
        console.error('WebSocket error:', error);
      };

      ws.current.onmessage = (event) => {
        try {
          setLastMessage(event.data);
        } catch (error) {
          console.error('Message parse error:', error);
        }
      };

      return () => {
        if (ws.current && ws.current.readyState === WebSocket.OPEN) {
          ws.current.close();
        }
      };
    } catch (error) {
      console.error('WebSocket connection error:', error);
      setConnectionStatus('error');
    }
  }, [url]);

  const sendMessage = useCallback((message) => {
    if (ws.current?.readyState === WebSocket.OPEN) {
      const request = {
        ...message,
        id: message.id || `${message.action}_${Date.now()}`
      };
      ws.current.send(JSON.stringify(request));
    } else {
      console.error('WebSocket is not connected');
    }
  }, []);

  return { connectionStatus, lastMessage, sendMessage };
};

export default useWebSocket;