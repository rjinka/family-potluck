import React, { createContext, useContext, useEffect, useState } from 'react';

const WebSocketContext = createContext(null);

export const WebSocketProvider = ({ children }) => {
    const [socket, setSocket] = useState(null);
    const [lastMessage, setLastMessage] = useState(null);

    useEffect(() => {
        // Connect to WebSocket
        const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        // Assuming backend is running on port 8080 or same host/port if proxied
        // Use window.location.hostname to allow connection from other devices on the network
        const wsUrl = `ws://${window.location.hostname}:5000/ws`;

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
