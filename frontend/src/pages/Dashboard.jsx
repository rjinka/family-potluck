import React, { useState, useEffect } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { useWebSocket } from '../context/WebSocketContext';
import { useUI } from '../context/UIContext';
import api from '../api/axios';
import {
    Calendar, MapPin, Users, Plus, ChevronRight,
    LogOut, Settings, Bell, Search, Filter,
    ChefHat, Clock, ArrowRight, SkipForward, CheckCircle, XCircle
} from 'lucide-react';

const Dashboard = () => {
    const { user, logout } = useAuth();
    const { lastMessage } = useWebSocket();
    const { showToast, confirm } = useUI();
    const navigate = useNavigate();
    const location = useLocation();
    const [groups, setGroups] = useState([]);
    const [selectedGroupId, setSelectedGroupId] = useState(location.state?.groupId || null);
    const [guestEvents, setGuestEvents] = useState([]);
    const [userGroups, setUserGroups] = useState([]);
    const [events, setEvents] = useState([]);
    const [loading, setLoading] = useState(true);
    const [showCreateModal, setShowCreateModal] = useState(false);
    const [newEvent, setNewEvent] = useState({
        type: 'Potluck',
        date: '',
        location: '',
        description: '',
        recurrence: ''
    });

    useEffect(() => {
        if (lastMessage) {
            if (lastMessage.type === 'event_created' || lastMessage.type === 'event_updated' || lastMessage.type === 'event_deleted') {
                // Refresh events if the event belongs to the current group
                if (selectedGroupId && lastMessage.data.group_id === selectedGroupId) {
                    fetchEvents();
                }

                // Refresh guest events if the updated/deleted event is in our guest list
                // For event_deleted, data is { event_id, group_id }
                // For event_updated, data is the event object (so data.id)
                const messageEventId = lastMessage.data.event_id || lastMessage.data.id;
                const isGuestEvent = guestEvents.some(e => e.id === messageEventId);

                if (isGuestEvent) {
                    fetchGuestEvents();
                }
            }
        }
    }, [lastMessage, selectedGroupId, guestEvents]);

    useEffect(() => {
        if (!user) return;

        // Fetch details of user's groups to display names
        if (user.group_ids && user.group_ids.length > 0) {
            fetchUserGroups();
            if (!selectedGroupId) {
                setSelectedGroupId(user.group_ids[0]);
            }
        }

        // Always fetch guest events
        fetchGuestEvents();
    }, [user, navigate]);

    const fetchGuestEvents = async () => {
        try {
            const response = await api.get(`/events/user?user_id=${user.id}`);
            const sortedEvents = (response.data || []).sort((a, b) => new Date(a.date) - new Date(b.date));
            setGuestEvents(sortedEvents);
        } catch (error) {
            console.error("Failed to fetch guest events", error);
        }
    };

    useEffect(() => {
        if (selectedGroupId) {
            fetchEvents();
        }
    }, [selectedGroupId]);

    const fetchUserGroups = async () => {
        try {
            const response = await api.get('/groups');
            // Filter groups that user is part of
            const myGroups = response.data.filter(g => user.group_ids.includes(g.id));
            setUserGroups(myGroups);
        } catch (error) {
            console.error("Failed to fetch groups", error);
        }
    };

    const fetchEvents = async () => {
        try {
            const response = await api.get(`/events?group_id=${selectedGroupId}`);
            const sortedEvents = (response.data || []).sort((a, b) => new Date(a.date) - new Date(b.date));
            setEvents(sortedEvents);
        } catch (error) {
            console.error("Failed to fetch events", error);
        }
    };

    const handleCreateEvent = async (e) => {
        e.preventDefault();
        try {
            await api.post('/events', {
                ...newEvent,
                group_id: selectedGroupId,
                host_id: user.id,
                date: new Date(newEvent.date).toISOString()
            });
            setShowCreateModal(false);
            fetchEvents();
            setNewEvent({ date: '', type: 'Dinner', location: '', description: '', recurrence: '' });
            showToast("Event created successfully!");
        } catch (error) {
            console.error("Failed to create event", error);
            showToast("Failed to create event", "error");
        }
    };

    const handleFinishEvent = async (eventId) => {
        confirm({
            title: "Complete Event",
            message: "Mark this event as complete? A new event will be created for the next occurrence.",
            confirmText: "Complete",
            onConfirm: async () => {
                try {
                    await api.post(`/events/${eventId}/finish?admin_id=${user.id}`);
                    fetchEvents();
                    showToast("Event completed and next occurrence created");
                } catch (error) {
                    console.error("Failed to complete event", error);
                    showToast("Failed to complete event", "error");
                }
            }
        });
    };

    const handleSkipEvent = async (eventId) => {
        confirm({
            title: "Skip Event",
            message: "Skip this event? It will be rescheduled to the next occurrence.",
            confirmText: "Skip",
            onConfirm: async () => {
                try {
                    await api.post(`/events/${eventId}/skip?admin_id=${user.id}`);
                    fetchEvents();
                    showToast("Event skipped");
                } catch (error) {
                    console.error("Failed to skip event", error);
                    showToast("Failed to skip event", "error");
                }
            }
        });
    };

    const handleDeleteEvent = async (eventId) => {
        confirm({
            title: "Cancel Event",
            message: "Are you sure you want to cancel this event? This will remove it from the calendar.",
            confirmText: "Cancel Event",
            isDestructive: true,
            onConfirm: async () => {
                try {
                    await api.delete(`/events/${eventId}?user_id=${user.id}`);
                    fetchEvents();
                    showToast("Event cancelled successfully");
                } catch (error) {
                    console.error("Failed to cancel event", error);
                    showToast("Failed to cancel event", "error");
                }
            }
        });
    };

    const handleCompleteNonRecurring = async (eventId) => {
        confirm({
            title: "Complete Event",
            message: "Mark this event as complete? It will be removed from the upcoming list.",
            confirmText: "Complete",
            onConfirm: async () => {
                try {
                    await api.delete(`/events/${eventId}?user_id=${user.id}`);
                    fetchEvents();
                    showToast("Event completed successfully");
                } catch (error) {
                    console.error("Failed to complete event", error);
                    showToast("Failed to complete event", "error");
                }
            }
        });
    };

    return (
        <div className="min-h-screen bg-gray-50 pb-20">
            {/* Header */}
            <header className="bg-white shadow-sm sticky top-0 z-10">
                <div className="max-w-5xl mx-auto px-4 py-4 flex justify-between items-center">
                    <h1 className="text-xl font-bold text-orange-600 flex items-center gap-2">
                        <ChefHat className="w-6 h-6" />
                        Gatherings
                    </h1>
                    <div className="flex items-center gap-4">
                        <span className="text-sm font-medium text-gray-600 hidden sm:block">{user?.name}</span>
                        <img src={user?.picture} alt="Profile" className="w-8 h-8 rounded-full border border-gray-200" />
                        <button onClick={logout} className="text-sm text-gray-500 hover:text-gray-800">Logout</button>
                    </div>
                </div>
            </header>

            <main className="max-w-5xl mx-auto px-4 py-8">
                {/* Group Selector */}
                {userGroups.length > 0 ? (
                    <div className="mb-8 flex items-center gap-4 bg-white p-4 rounded-xl shadow-sm border border-gray-100">
                        <div className="bg-orange-100 p-2 rounded-lg">
                            <Users className="w-5 h-5 text-orange-600" />
                        </div>
                        <div className="flex-1">
                            <label className="block text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Current Group</label>
                            <select
                                className="w-full md:w-auto bg-transparent font-bold text-gray-800 outline-none cursor-pointer hover:text-orange-600 transition"
                                value={selectedGroupId || ''}
                                onChange={(e) => setSelectedGroupId(e.target.value)}
                            >
                                {userGroups.map(group => (
                                    <option key={group.id} value={group.id}>{group.name}</option>
                                ))}
                            </select>
                        </div>
                        <button onClick={() => navigate('/groups')} className="text-sm text-orange-600 font-medium hover:underline">
                            Manage Groups
                        </button>
                    </div>
                ) : (
                    <div className="mb-8 bg-blue-50 p-4 rounded-xl border border-blue-100 flex justify-between items-center">
                        <div className="flex items-center gap-3">
                            <Users className="w-5 h-5 text-blue-600" />
                            <span className="text-blue-800 font-medium">You are not in any groups yet.</span>
                        </div>
                        <button onClick={() => navigate('/groups')} className="text-sm bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition">
                            Join or Create Group
                        </button>
                    </div>
                )}

                {/* Guest Events Section */}
                {guestEvents.length > 0 && (
                    <div className="mb-10">
                        <h2 className="text-2xl font-bold text-gray-800 mb-6 flex items-center gap-2">
                            <Calendar className="w-6 h-6 text-orange-500" />
                            Events You're Invited To
                        </h2>
                        <div className="space-y-4">
                            {guestEvents.map(event => (
                                <div key={event.id} className="bg-white p-6 rounded-xl shadow-sm border border-orange-100 hover:shadow-md transition group relative overflow-hidden">
                                    <div className="absolute top-0 right-0 bg-orange-500 text-white text-xs font-bold px-3 py-1 rounded-bl-lg">
                                        GUEST
                                    </div>
                                    <div className="flex justify-between items-start">
                                        <div className="flex-1">
                                            <div className="flex items-center gap-2 mb-2">
                                                <span className="bg-orange-100 text-orange-700 text-xs font-bold px-2 py-1 rounded-full uppercase tracking-wide">
                                                    {event.type}
                                                </span>
                                            </div>
                                            <h3 className="text-lg font-bold text-gray-800 mb-1 group-hover:text-orange-600 transition">{new Date(event.date).toLocaleDateString(undefined, { weekday: 'long', month: 'long', day: 'numeric' })}</h3>
                                            <div className="text-gray-500 text-sm flex items-center gap-4 mb-4">
                                                <span className="flex items-center gap-1">
                                                    <Clock className="w-4 h-4" />
                                                    {new Date(event.date).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                                                </span>
                                                <span className="flex items-center gap-1">
                                                    <MapPin className="w-4 h-4" />
                                                    {event.location}
                                                </span>
                                            </div>
                                            {event.host_name && (
                                                <p className="text-sm text-gray-600 mb-2">Hosted by <span className="font-semibold">{event.host_name}</span></p>
                                            )}
                                        </div>
                                        <div className="flex flex-col gap-2 mt-6">
                                            <button
                                                onClick={() => navigate(`/events/${event.id}`)}
                                                className="bg-orange-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-orange-700 transition flex items-center gap-2"
                                            >
                                                View Details
                                                <ArrowRight className="w-4 h-4" />
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                )}

                {selectedGroupId && userGroups.length > 0 && (
                    <>
                        <div className="flex justify-between items-center mb-6">
                            <h2 className="text-2xl font-bold text-gray-800">Group Events</h2>
                            <button
                                onClick={() => setShowCreateModal(!showCreateModal)}
                                className="bg-orange-600 text-white px-4 py-2 rounded-lg flex items-center gap-2 hover:bg-orange-700 transition shadow-sm"
                            >
                                <Plus className="w-4 h-4" />
                                {showCreateModal ? 'Cancel' : 'New Event'}
                            </button>
                        </div>

                        {/* Create Event Form */}
                        {showCreateModal && (
                            <div className="bg-white p-6 rounded-xl shadow-sm border border-gray-100 mb-8 animate-in fade-in slide-in-from-top-4">
                                <h3 className="text-lg font-semibold mb-4">Plan a New Gathering</h3>
                                <form onSubmit={handleCreateEvent} className="grid gap-4 md:grid-cols-2">
                                    <div>
                                        <label className="block text-sm font-medium text-gray-700 mb-1">Date & Time</label>
                                        <input
                                            type="datetime-local"
                                            required
                                            className="w-full p-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 outline-none"
                                            value={newEvent.date}
                                            onChange={e => setNewEvent({ ...newEvent, date: e.target.value })}
                                        />
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-gray-700 mb-1">Type</label>
                                        <select
                                            className="w-full p-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 outline-none"
                                            value={newEvent.type}
                                            onChange={e => setNewEvent({ ...newEvent, type: e.target.value })}
                                        >
                                            <option>Dinner</option>
                                            <option>Lunch</option>
                                            <option>Coffee Meet</option>
                                            <option>Picnic</option>
                                        </select>
                                    </div>

                                    <div>
                                        <label className="block text-sm font-medium text-gray-700 mb-1">Recurrence</label>
                                        <select
                                            className="w-full p-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 outline-none"
                                            value={newEvent.recurrence}
                                            onChange={e => setNewEvent({ ...newEvent, recurrence: e.target.value })}
                                        >
                                            <option value="">None (One-time)</option>
                                            <option value="Weekly">Weekly</option>
                                            <option value="Bi-Weekly">Bi-Weekly</option>
                                        </select>
                                    </div>

                                    <div className="md:col-span-2">
                                        <label className="block text-sm font-medium text-gray-700 mb-1">Location (Leave empty to use your household address)</label>
                                        <input
                                            type="text"
                                            placeholder="e.g. 123 Main St or Central Park"
                                            className="w-full p-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 outline-none"
                                            value={newEvent.location}
                                            onChange={e => setNewEvent({ ...newEvent, location: e.target.value })}
                                        />
                                    </div>
                                    <div className="md:col-span-2">
                                        <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
                                        <textarea
                                            placeholder="Any special notes? Theme?"
                                            className="w-full p-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 outline-none"
                                            rows="3"
                                            value={newEvent.description}
                                            onChange={e => setNewEvent({ ...newEvent, description: e.target.value })}
                                        />
                                    </div>
                                    <div className="md:col-span-2 flex justify-end">
                                        <button type="submit" className="bg-orange-600 text-white px-6 py-2 rounded-lg hover:bg-orange-700">
                                            Create Event
                                        </button>
                                    </div>
                                </form>
                            </div>
                        )}

                        {/* Events List */}
                        <div className="space-y-4">
                            {events.length === 0 ? (
                                <div className="text-center py-12 bg-white rounded-xl border border-gray-100 border-dashed">
                                    <Calendar className="w-12 h-12 text-gray-300 mx-auto mb-3" />
                                    <p className="text-gray-500">No upcoming events found.</p>
                                    <button
                                        onClick={() => setShowCreateModal(true)}
                                        className="text-orange-600 font-medium hover:underline mt-2"
                                    >
                                        Plan one now
                                    </button>
                                </div>
                            ) : (
                                events.map(event => (
                                    <div key={event.id} className="bg-white p-6 rounded-xl shadow-sm border border-gray-100 hover:shadow-md transition group">
                                        <div className="flex justify-between items-start">
                                            <div className="flex-1">
                                                <div className="flex items-center gap-2 mb-2">
                                                    <span className="bg-orange-100 text-orange-700 text-xs font-bold px-2 py-1 rounded-full uppercase tracking-wide">
                                                        {event.type}
                                                    </span>
                                                    {event.recurrence && (
                                                        <span className="bg-blue-50 text-blue-600 text-xs px-2 py-1 rounded-full flex items-center gap-1">
                                                            <Clock className="w-3 h-3" />
                                                            {event.recurrence}
                                                        </span>
                                                    )}
                                                </div>
                                                <h3 className="text-lg font-bold text-gray-800 mb-1 group-hover:text-orange-600 transition">{new Date(event.date).toLocaleDateString(undefined, { weekday: 'long', month: 'long', day: 'numeric' })}</h3>
                                                <div className="text-gray-500 text-sm flex items-center gap-4 mb-4">
                                                    <span className="flex items-center gap-1">
                                                        <Clock className="w-4 h-4" />
                                                        {new Date(event.date).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                                                    </span>
                                                    <span className="flex items-center gap-1">
                                                        <MapPin className="w-4 h-4" />
                                                        {event.location}
                                                    </span>
                                                </div>
                                                {event.description && (
                                                    <p className="text-gray-600 text-sm mb-4 line-clamp-2">{event.description}</p>
                                                )}
                                            </div>
                                            <div className="flex flex-col gap-2">
                                                <button
                                                    onClick={() => navigate(`/events/${event.id}`)}
                                                    className="bg-gray-50 text-gray-600 px-4 py-2 rounded-lg text-sm font-medium hover:bg-orange-50 hover:text-orange-600 transition flex items-center gap-2"
                                                >
                                                    Details & RSVP
                                                    <ArrowRight className="w-4 h-4" />
                                                </button>

                                                {/* Admin Controls for Recurring Events */}
                                                {/* Admin/Host Controls */}
                                                {(userGroups.find(g => g.id === selectedGroupId)?.admin_id === user?.id || event.host_id === user?.id) && (
                                                    <div className="flex gap-2 justify-end">
                                                        {event.recurrence && (
                                                            <button
                                                                onClick={() => handleSkipEvent(event.id)}
                                                                className="text-xs text-gray-400 hover:text-orange-600 flex items-center gap-1"
                                                                title="Skip to next occurrence"
                                                            >
                                                                <SkipForward className="w-3 h-3" /> Skip
                                                            </button>
                                                        )}

                                                        <button
                                                            onClick={() => event.recurrence ? handleFinishEvent(event.id) : handleCompleteNonRecurring(event.id)}
                                                            className="text-xs text-gray-400 hover:text-green-600 flex items-center gap-1"
                                                            title={event.recurrence ? "Mark as complete and create next occurrence" : "Mark as complete"}
                                                        >
                                                            <CheckCircle className="w-3 h-3" /> Complete
                                                        </button>

                                                        {(!event.recurrence || userGroups.find(g => g.id === selectedGroupId)?.admin_id === user?.id) && (
                                                            <button
                                                                onClick={() => handleDeleteEvent(event.id)}
                                                                className="text-xs text-gray-400 hover:text-red-600 flex items-center gap-1"
                                                                title="Cancel Event"
                                                            >
                                                                <XCircle className="w-3 h-3" /> Cancel
                                                            </button>
                                                        )}
                                                    </div>
                                                )}
                                            </div>
                                        </div>
                                    </div>
                                ))
                            )}
                        </div>
                    </>
                )}
            </main>
        </div>
    );
};

export default Dashboard;
