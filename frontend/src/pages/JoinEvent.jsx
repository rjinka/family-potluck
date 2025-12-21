import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import api from '../api/axios';
import { Calendar, MapPin, User, ArrowRight, CheckCircle, Clock } from 'lucide-react';

const JoinEvent = () => {
    const { joinCode } = useParams();
    const { user, refreshUser } = useAuth();
    const navigate = useNavigate();
    const [event, setEvent] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [joining, setJoining] = useState(false);

    useEffect(() => {
        const fetchEvent = async () => {
            try {
                const response = await api.get(`/events/code/${joinCode}`);
                setEvent(response.data);
            } catch (err) {
                console.error("Failed to fetch event", err);
                if (err.response && err.response.status === 403) {
                    setError("This event has already finished.");
                } else {
                    setError("Event not found or invalid link.");
                }
            } finally {
                setLoading(false);
            }
        };

        if (joinCode) {
            fetchEvent();
        }
    }, [joinCode]);

    const handleJoin = async () => {
        if (!user) {
            // Redirect to login, preserving the return url (not implemented yet, just redirect)
            navigate('/login');
            return;
        }

        setJoining(true);
        try {
            await api.post('/events/join-by-code', {
                family_id: user.id,
                join_code: joinCode
            });
            // We don't necessarily need to refresh user here unless we store guest events in user object
            // But we might want to refresh to ensure latest state
            await refreshUser();
            navigate(`/events/${event.id}`);
        } catch (err) {
            console.error("Failed to join event", err);
            setError("Failed to join event. Please try again.");
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
                        onClick={() => navigate('/dashboard')}
                        className="text-orange-600 font-medium hover:underline"
                    >
                        Go to Dashboard
                    </button>
                </div>
            </div>
        );
    }

    // Check if user is already a guest or host or in the group (if we had that info)
    // For now, just check if they are in the guest list or host
    const isHost = user && event && event.host_id === user.id;
    const isGuest = user && event && event.guest_ids && event.guest_ids.includes(user.id);
    // We can't easily check if they are in the group without fetching group members or having group_ids in user
    // But if they are in the group, they can access the event anyway.
    // However, this page is specifically for the guest link.

    const alreadyJoined = isHost || isGuest;

    return (
        <div className="min-h-screen bg-gray-50 flex flex-col items-center justify-center p-4">
            <div className="bg-white p-8 rounded-2xl shadow-sm border border-gray-100 text-center max-w-md w-full">
                <div className="bg-orange-100 w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-6">
                    <Calendar className="w-8 h-8 text-orange-600" />
                </div>

                <h1 className="text-2xl font-bold text-gray-800 mb-2">You're Invited!</h1>
                <p className="text-gray-600 mb-6">
                    You have been invited to join the event:
                </p>

                <div className="bg-gray-50 p-4 rounded-xl border border-gray-100 mb-8 text-left">
                    <h2 className="font-bold text-lg text-gray-800 mb-2">{event.type}</h2>
                    <div className="space-y-2 text-sm text-gray-600">
                        <div className="flex items-center gap-2">
                            <Calendar className="w-4 h-4 text-orange-500" />
                            <span>{new Date(event.date).toLocaleDateString()}</span>
                        </div>
                        <div className="flex items-center gap-2">
                            <Clock className="w-4 h-4 text-orange-500" />
                            <span>{new Date(event.date).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
                        </div>
                        <div className="flex items-center gap-2">
                            <MapPin className="w-4 h-4 text-orange-500" />
                            <span>{event.location}</span>
                        </div>
                        {event.host_name && (
                            <div className="flex items-center gap-2">
                                <User className="w-4 h-4 text-orange-500" />
                                <span>Hosted by {event.host_name}</span>
                            </div>
                        )}
                    </div>
                </div>

                {alreadyJoined ? (
                    <div className="mb-6">
                        <div className="bg-green-50 text-green-700 p-4 rounded-lg mb-4 flex items-center justify-center gap-2">
                            <CheckCircle className="w-5 h-5" />
                            <span className="font-medium">You have access to this event</span>
                        </div>
                        <button
                            onClick={() => navigate(`/events/${event.id}`)}
                            className="w-full bg-orange-600 text-white py-3 rounded-lg font-medium hover:bg-orange-700 transition"
                        >
                            View Event Details
                        </button>
                    </div>
                ) : (
                    <button
                        onClick={handleJoin}
                        disabled={joining}
                        className="w-full bg-orange-600 text-white py-3 rounded-lg font-medium hover:bg-orange-700 transition flex items-center justify-center gap-2 disabled:opacity-70"
                    >
                        {joining ? 'Joining...' : 'Join Event'}
                        {!joining && <ArrowRight className="w-5 h-5" />}
                    </button>
                )}

                {!user && (
                    <button
                        onClick={() => navigate('/login')}
                        className="mt-4 text-gray-500 hover:text-gray-800 text-sm font-medium"
                    >
                        Login to Join
                    </button>
                )}
            </div>
        </div>
    );
};

export default JoinEvent;
