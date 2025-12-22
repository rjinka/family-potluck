import React, { createContext, useContext, useState, useCallback } from 'react';
import Notification from '../components/Notification';
import ConfirmationModal from '../components/ConfirmationModal';

const UIContext = createContext();

export const useUI = () => {
    const context = useContext(UIContext);
    if (!context) {
        throw new Error('useUI must be used within a UIProvider');
    }
    return context;
};

export const UIProvider = ({ children }) => {
    // Notification State
    const [notification, setNotification] = useState({
        isVisible: false,
        message: '',
        type: 'success'
    });

    // Confirmation Modal State
    const [modal, setModal] = useState({
        isOpen: false,
        title: '',
        message: '',
        confirmText: 'Confirm',
        cancelText: 'Cancel',
        isDestructive: false,
        onConfirm: () => { }
    });

    const showToast = useCallback((message, type = 'success') => {
        setNotification({ isVisible: true, message, type });
    }, []);

    const closeToast = useCallback(() => {
        setNotification(prev => ({ ...prev, isVisible: false }));
    }, []);

    const closeModal = useCallback(() => {
        setModal(prev => ({ ...prev, isOpen: false }));
    }, []);

    const confirm = useCallback(({ title, message, confirmText, cancelText, isDestructive, onConfirm }) => {
        setModal({
            isOpen: true,
            title,
            message,
            confirmText: confirmText || 'Confirm',
            cancelText: cancelText || 'Cancel',
            isDestructive: isDestructive || false,
            onConfirm: () => {
                if (onConfirm) onConfirm();
                closeModal();
            }
        });
    }, [closeModal]);

    return (
        <UIContext.Provider value={{ showToast, confirm }}>
            {children}
            <Notification
                message={notification.message}
                type={notification.type}
                isVisible={notification.isVisible}
                onClose={closeToast}
            />
            <ConfirmationModal
                isOpen={modal.isOpen}
                onClose={closeModal}
                onConfirm={modal.onConfirm}
                title={modal.title}
                message={modal.message}
                confirmText={modal.confirmText}
                cancelText={modal.cancelText}
                isDestructive={modal.isDestructive}
            />
        </UIContext.Provider>
    );
};
