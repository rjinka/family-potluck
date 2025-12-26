import axios from 'axios';

const api = axios.create({
    baseURL: import.meta.env.VITE_API_URL || 'http://localhost:5000',
    withCredentials: true,
});

api.interceptors.response.use(
    (response) => response,
    (error) => {
        // We handle 401s in the AuthContext/ProtectedRoute
        // rather than doing a hard window.location redirect here
        return Promise.reject(error);
    }
);

export default api;
