import React, { createContext, useContext, useEffect, useState } from 'react';

const WebSocketContext = createContext(null);

export const WebSocketProvider = ({ children }) => {
    const [socket, setSocket] = useState(null);
    const [lastMessage, setLastMessage] = useState(null);

    useEffect(() => {
        // Connect to WebSocket
        const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';

        let wsHost;
        const apiUrl = import.meta.env.VITE_API_URL;

        if (apiUrl) {
            // Extract host from VITE_API_URL (e.g., https://api.example.com -> api.example.com)
            try {
                const url = new URL(apiUrl);
                wsHost = url.host;
            } catch (e) {
                console.error("Invalid VITE_API_URL", e);
                wsHost = `${window.location.hostname}:5000`;
            }
        } else {
            wsHost = `${window.location.hostname}:5000`;
        }

        const wsUrl = `${wsProtocol}//${wsHost}/ws`;

        const ws = new WebSocket(wsUrl);

        ws.onopen = () => {
            console.log('Connected to WebSocket');
        };

        ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                setLastMessage(message);
            } catch (e) {
                console.error("Failed to parse websocket message", e);
            }
        };

        ws.onclose = () => {
            console.log('Disconnected from WebSocket');
            // Implement reconnection logic if needed
        };

        setSocket(ws);

        return () => {
            ws.close();
        };
    }, []);

    return (
        <WebSocketContext.Provider value={{ socket, lastMessage }}>
            {children}
        </WebSocketContext.Provider>
    );
};

export const useWebSocket = () => useContext(WebSocketContext);
