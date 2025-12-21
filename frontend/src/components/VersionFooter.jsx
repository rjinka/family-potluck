import React, { useEffect, useState } from 'react';
import api from '../api/axios';
import pkg from '../../package.json';

const VersionFooter = () => {
    const [backendVersion, setBackendVersion] = useState('Loading...');
    const frontendVersion = pkg.version;

    useEffect(() => {
        const fetchVersion = async () => {
            try {
                const response = await api.get('/version');
                setBackendVersion(response.data.version);
            } catch (error) {
                console.error('Failed to fetch backend version:', error);
                setBackendVersion('Unknown');
            }
        };

        fetchVersion();
    }, []);

    return (
        <footer className="fixed bottom-0 left-0 right-0 p-2 bg-white/80 backdrop-blur-sm border-t border-gray-100 flex justify-center gap-4 text-[10px] text-gray-400 z-50">
            <span>Frontend: v{frontendVersion}</span>
            <span className="text-gray-200">|</span>
            <span>Backend: v{backendVersion}</span>
        </footer>
    );
};

export default VersionFooter;
