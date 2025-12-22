import React, { createContext, useState, useContext, useEffect } from 'react';
import api from '../api/axios';

const AuthContext = createContext();

export const AuthProvider = ({ children }) => {
    const [user, setUser] = useState(null);
    const [loading, setLoading] = useState(true);

    const checkAuth = async () => {
        try {
            const response = await api.get('/auth/me');
            setUser(response.data);
        } catch (error) {
            console.error("Auth check failed", error);
            // Only clear user if it's a 401 Unauthorized
            if (error.response && error.response.status === 401) {
                setUser(null);
            }
            // For other errors (like network issues), we keep the cached user
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        checkAuth();
    }, []);

    const login = async (idToken) => {
        try {
            const response = await api.post('/auth/google', { id_token: idToken });
            setUser(response.data);
            return response.data;
        } catch (error) {
            console.error("Login failed", error);
            throw error;
        }
    };

    const refreshUser = async () => {
        await checkAuth();
    };

    const logout = async () => {
        try {
            await api.post('/auth/logout');
        } catch (error) {
            console.error("Logout failed", error);
        }
        setUser(null);
    };

    return (
        <AuthContext.Provider value={{ user, login, logout, loading, refreshUser }}>
            {children}
        </AuthContext.Provider>
    );
};

export const useAuth = () => useContext(AuthContext);
