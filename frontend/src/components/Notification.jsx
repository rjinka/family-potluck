import React, { useEffect } from 'react';
import { CheckCircle, XCircle, X } from 'lucide-react';

const Notification = ({ message, type = 'success', onClose, isVisible }) => {
    useEffect(() => {
        if (isVisible) {
            const timer = setTimeout(() => {
                onClose();
            }, 3000); // Auto close after 3 seconds
            return () => clearTimeout(timer);
        }
    }, [isVisible, onClose]);

    if (!isVisible) return null;

    return (
        <div className="fixed bottom-6 right-6 z-50 animate-in slide-in-from-bottom-5 duration-300">
            <div className={`flex items-center gap-3 px-4 py-3 rounded-lg shadow-lg border ${type === 'success'
                    ? 'bg-green-50 border-green-200 text-green-800'
                    : 'bg-red-50 border-red-200 text-red-800'
                }`}>
                {type === 'success' ? (
                    <CheckCircle className="w-5 h-5 text-green-600" />
                ) : (
                    <XCircle className="w-5 h-5 text-red-600" />
                )}
                <p className="font-medium text-sm">{message}</p>
                <button onClick={onClose} className={`ml-2 p-1 rounded-full hover:bg-black/5 ${type === 'success' ? 'text-green-600' : 'text-red-600'
                    }`}>
                    <X className="w-4 h-4" />
                </button>
            </div>
        </div>
    );
};

export default Notification;
