import React, { createContext, useContext, useEffect, useState } from 'react';

const WebSocketContext = createContext(null);

export const WebSocketProvider = ({ children }) => {
    const [socket, setSocket] = useState(null);
    const [lastMessage, setLastMessage] = useState(null);

    useEffect(() => {
        // Connect to WebSocket
        let wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        let wsHost = `${window.location.hostname}:5000`;
        const apiUrl = import.meta.env.VITE_API_URL;

        if (apiUrl) {
            try {
                const url = new URL(apiUrl);
                wsHost = url.host;
                wsProtocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
            } catch (e) {
                console.error("Invalid VITE_API_URL", e);
            }
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
