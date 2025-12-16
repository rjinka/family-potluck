import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import api from '../api/axios';
import { Users, ArrowRight, CheckCircle } from 'lucide-react';

const JoinGroup = () => {
    const { joinCode } = useParams();
    const { user, refreshUser } = useAuth();
    const navigate = useNavigate();
    const [group, setGroup] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [joining, setJoining] = useState(false);

    useEffect(() => {
        const fetchGroup = async () => {
            try {
                const response = await api.get(`/groups/code/${joinCode}`);
                setGroup(response.data);
            } catch (err) {
                console.error("Failed to fetch group", err);
                setError("Group not found or invalid link.");
            } finally {
                setLoading(false);
            }
        };

        if (joinCode) {
            fetchGroup();
        }
    }, [joinCode]);

    const handleJoin = async () => {
        if (!user) {
            // Redirect to login, preserving the return url
            // For now, simple redirect to login
            navigate('/login');
            return;
        }

        setJoining(true);
        try {
            await api.post('/groups/join-by-code', {
                family_id: user.id,
                join_code: joinCode
            });
            await refreshUser();
            navigate('/dashboard');
        } catch (err) {
            console.error("Failed to join group", err);
            setError("Failed to join group. Please try again.");
        } finally {
            setJoining(false);
        }
    };

    if (loading) return <div className="flex justify-center items-center h-screen">Loading...</div>;

    if (error) {
        return (
            <div className="min-h-screen bg-gray-50 flex flex-col items-center justify-center p-4">
                <div className="bg-white p-8 rounded-2xl shadow-sm border border-gray-100 text-center max-w-md w-full">
                    <div className="text-red-500 mb-4 text-xl font-semibold">Oops!</div>
                    <p className="text-gray-600 mb-6">{error}</p>
                    <button
                        onClick={() => navigate('/groups')}
                        className="text-orange-600 font-medium hover:underline"
                    >
                        Go to Groups
                    </button>
                </div>
            </div>
        );
    }

    const isMember = group && user?.group_ids?.includes(group.id);

    return (
        <div className="min-h-screen bg-gray-50 flex flex-col items-center justify-center p-4">
            <div className="bg-white p-8 rounded-2xl shadow-sm border border-gray-100 text-center max-w-md w-full">
                <div className="bg-blue-100 w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-6">
                    <Users className="w-8 h-8 text-blue-600" />
                </div>

                <h1 className="text-2xl font-bold text-gray-800 mb-2">You're Invited!</h1>
                <p className="text-gray-600 mb-8">
                    You have been invited to join the <span className="font-semibold text-gray-800">{group?.name}</span> gathering circle.
                </p>

                {isMember ? (
                    <div className="bg-green-50 text-green-700 p-4 rounded-lg mb-6 flex items-center justify-center gap-2">
                        <CheckCircle className="w-5 h-5" />
                        <span className="font-medium">You are already a member</span>
                    </div>
                ) : (
                    <button
                        onClick={handleJoin}
                        disabled={joining}
                        className="w-full bg-orange-600 text-white py-3 rounded-lg font-medium hover:bg-orange-700 transition flex items-center justify-center gap-2 disabled:opacity-70"
                    >
                        {joining ? 'Joining...' : 'Join Group'}
                        {!joining && <ArrowRight className="w-5 h-5" />}
                    </button>
                )}

                {(isMember || !user) && (
                    <button
                        onClick={() => navigate(isMember ? '/dashboard' : '/login')}
                        className="mt-4 text-gray-500 hover:text-gray-800 text-sm font-medium"
                    >
                        {isMember ? 'Go to Dashboard' : 'Login to Join'}
                    </button>
                )}
            </div>
        </div>
    );
};

export default JoinGroup;
