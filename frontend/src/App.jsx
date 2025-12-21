import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import { WebSocketProvider } from './context/WebSocketContext';
import { UIProvider } from './context/UIContext';
import Login from './pages/Login';
import Groups from './pages/Groups';
import Dashboard from './pages/Dashboard';
import JoinGroup from './pages/JoinGroup';
import JoinEvent from './pages/JoinEvent';
import EventDetails from './pages/EventDetails';
import VersionFooter from './components/VersionFooter';
import { Toaster } from 'sonner';

const ProtectedRoute = ({ children }) => {
  const { user, loading } = useAuth();

  if (loading) return <div>Loading...</div>;
  if (!user) return <Navigate to="/login" />;

  return children;
};

function App() {
  return (
    <AuthProvider>
      <WebSocketProvider>
        <UIProvider>
          <Router>
            <Routes>
              <Route path="/login" element={<Login />} />
              <Route path="/groups" element={<ProtectedRoute><Groups /></ProtectedRoute>} />
              <Route path="/dashboard" element={<ProtectedRoute><Dashboard /></ProtectedRoute>} />
              <Route path="/join/:joinCode" element={<JoinGroup />} />
              <Route path="/join-event/:joinCode" element={<JoinEvent />} />
              <Route path="/events/:eventId" element={<ProtectedRoute><EventDetails /></ProtectedRoute>} />
              <Route path="/" element={<Navigate to="/dashboard" replace />} />
            </Routes>
          </Router>
          <Toaster position="top-center" richColors />
          <VersionFooter />
        </UIProvider>
      </WebSocketProvider>
    </AuthProvider>
  );
}

export default App;
