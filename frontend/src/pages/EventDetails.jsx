import React, { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { useWebSocket } from '../context/WebSocketContext';
import { toast } from 'sonner';
import api from '../api/axios';
import {
    Calendar, MapPin, ChefHat, ArrowLeft, CheckCircle,
    HelpCircle, XCircle, Trash2, Plus, User, RefreshCw, Share2, Copy, BarChart2, Edit, Sparkles
} from 'lucide-react';

import { useUI } from '../context/UIContext';
import EventStatsModal from '../components/EventStatsModal';
import EventRSVPModal from '../components/EventRSVPModal';
import HostSummary from '../components/HostSummary';
import DietaryPreferencesModal from '../components/DietaryPreferencesModal';

const DIETARY_TAGS = ["Vegan", "Vegetarian", "Gluten-Free", "Dairy-Free", "Nut-Free", "Spicy", "Halal", "Kosher"];

const EventDetails = () => {
    const { eventId } = useParams();
    const { user } = useAuth();
    const { lastMessage } = useWebSocket();
    const { showToast, confirm } = useUI();
    const navigate = useNavigate();

    const [event, setEvent] = useState(null);
    const [dishes, setDishes] = useState([]);
    const [rsvps, setRsvps] = useState([]);
    const [swapRequests, setSwapRequests] = useState([]);
    const [groupMembers, setGroupMembers] = useState([]);
    const [loading, setLoading] = useState(true);
    const [addingDish, setAddingDish] = useState(false);
    const [showRSVPModal, setShowRSVPModal] = useState(false);
    const [showHostAcceptModal, setShowHostAcceptModal] = useState(false);
    const [showStatsModal, setShowStatsModal] = useState(false);
    const [showRSVPListModal, setShowRSVPListModal] = useState(false);
    const [showDietaryModal, setShowDietaryModal] = useState(false);
    const [rsvpStatus, setRsvpStatus] = useState(null);
    const [newDish, setNewDish] = useState({ name: '', description: '', isRequest: false, dietary_tags: [] });
    const [customTag, setCustomTag] = useState('');
    const [copiedGuestLink, setCopiedGuestLink] = useState(false);
    const [eventStats, setEventStats] = useState(null);
    const [rsvpData, setRsvpData] = useState({ count: 1, kidsCount: 0 });
    const [hostUpdateData, setHostUpdateData] = useState({ date: '', time: '', location: '' });
    const [isAiThinking, setIsAiThinking] = useState(false);



    const fetchEventDetails = useCallback(async () => {
        try {
            const response = await api.get(`/events/${eventId}`);
            setEvent(response.data);
        } catch (error) {
            console.error("Failed to fetch event details", error);
        } finally {
            setLoading(false);
        }
    }, [eventId]);

    const fetchDishes = useCallback(async () => {
        try {
            const response = await api.get(`/dishes?event_id=${eventId}`);
            setDishes(response.data || []);
        } catch (error) {
            console.error("Failed to fetch dishes", error);
        }
    }, [eventId]);

    const fetchRSVPs = useCallback(async () => {
        try {
            const response = await api.get(`/rsvps?event_id=${eventId}`);
            const rsvpList = response.data || [];
            setRsvps(rsvpList);
            // Update current user's RSVP status
            const myRsvp = rsvpList.find(r => r.family_id === user.id);
            if (myRsvp) {
                setRsvpStatus(myRsvp.status);
                setRsvpData({
                    count: myRsvp.count || 1,
                    kidsCount: myRsvp.kids_count || 0
                });
            }
        } catch (error) {
            console.error("Failed to fetch RSVPs", error);
        }
    }, [eventId, user.id]);

    const fetchEventStats = useCallback(async () => {
        try {
            const response = await api.get(`/events/stats/${eventId}`);
            setEventStats(response.data);
        } catch (error) {
            console.error("Failed to fetch event stats", error);
        }
    }, [eventId]);

    const fetchGroupMembers = useCallback(async (groupId) => {
        try {
            const response = await api.get(`/groups/members?group_id=${groupId}`);
            setGroupMembers(response.data || { families: [], households: [] });
        } catch (error) {
            console.error("Failed to fetch group members", error);
        }
    }, []);


    const handleRSVP = (status) => {
        if (status === 'Yes') {
            setShowRSVPModal(true);
        } else {
            submitRSVP(status, 0, 0);
        }
    };

    const submitRSVP = async (status, count, kidsCount) => {
        try {
            await api.post('/rsvps', {
                event_id: eventId,
                family_id: user.id,
                status: status,
                count: parseInt(count) || 0,
                kids_count: parseInt(kidsCount) || 0
            });
            setRsvpStatus(status);
            setShowRSVPModal(false);
            setRsvpData({ count: 1, kidsCount: 0 }); // Reset form
            fetchRSVPs(); // Refresh list
            showToast(`RSVP updated: ${status}`, 'success');
        } catch (error) {
            console.error("Failed to update RSVP", error);
            showToast("Failed to update RSVP", "error");
        }
    };

    const [groups, setGroups] = useState([]);
    const [isAdmin, setIsAdmin] = useState(false);
    const [isHostHousehold, setIsHostHousehold] = useState(false);



    const fetchGroups = useCallback(async () => {
        try {
            const response = await api.get('/groups');
            setGroups(response.data || []);
        } catch (error) {
            console.error("Failed to fetch groups", error);
        }
    }, []);

    const fetchSwapRequests = useCallback(async () => {
        try {
            const response = await api.get(`/swaps?event_id=${eventId}`);
            setSwapRequests(response.data || []);
        } catch (error) {
            console.error("Failed to fetch swap requests", error);
        }
    }, [eventId]);

    const handleRequestSwap = (dish) => {
        const isOwnDish = dish.bringer_id === user.id;
        confirm({
            title: isOwnDish ? "Offer Dish" : "Request Swap",
            message: isOwnDish
                ? `Offer ${dish.name} for swap? Anyone can accept it.`
                : `Request to swap for ${dish.name}?`,
            onConfirm: () => executeRequestSwap(dish, isOwnDish),
            isDestructive: false
        });
    };

    const executeRequestSwap = async (dish, isOwnDish) => {
        try {
            await api.post('/swaps', {
                event_id: eventId,
                dish_id: dish.id,
                requesting_family_id: user.id,
                target_family_id: isOwnDish ? null : dish.bringer_id,
            });
            showToast(isOwnDish ? "Dish offered for swap!" : "Swap request sent!", "success");
            fetchSwapRequests();
        } catch (error) {
            console.error("Failed to request swap", error);
            showToast("Failed to request swap", "error");
        }
    };

    const handleRequestHostSwap = () => {
        confirm({
            title: "Request Host Swap",
            message: "Are you sure you want to request a host swap? This will ask other members to volunteer.",
            onConfirm: executeRequestHostSwap,
            isDestructive: false
        });
    };

    const executeRequestHostSwap = async () => {
        try {
            await api.post('/swaps', {
                event_id: eventId,
                type: 'host',
                requesting_family_id: user.id,
                // target_family_id is null for open request
            });
            showToast("Host swap request sent!", "success");
            fetchSwapRequests();
        } catch (error) {
            console.error("Failed to request host swap", error);
            showToast("Failed to request host swap", "error");
        }
    };

    const [selectedSwapRequestId, setSelectedSwapRequestId] = useState(null);

    const handleAcceptHostSwapClick = (requestId) => {
        setSelectedSwapRequestId(requestId);
        // Pre-fill with current event data
        const eventDate = new Date(event.date);

        // Find user's household address to overwrite location
        let userAddress = event.location;
        if (user.household_id && groupMembers.households) {
            const userHousehold = groupMembers.households.find(h => h.id === user.household_id);
            if (userHousehold && userHousehold.address) {
                userAddress = userHousehold.address;
            }
        }

        setHostUpdateData({
            date: eventDate.toISOString().split('T')[0],
            time: eventDate.toTimeString().slice(0, 5),
            location: userAddress
        });
        setShowHostAcceptModal(true);
    };

    const confirmAcceptHostSwap = async () => {
        // No need for window.confirm here as the modal itself is the confirmation step
        // But if we want double confirmation, we can use the new modal. 
        // However, the "Host Accept Modal" is already a form of confirmation.
        // Let's just proceed.

        try {
            // Combine date and time
            const dateTime = new Date(`${hostUpdateData.date}T${hostUpdateData.time}`);

            await api.patch(`/swaps/${selectedSwapRequestId}`, {
                status: 'approved',
                target_family_id: user.id,
                event_updates: {
                    date: dateTime.toISOString(),
                    location: hostUpdateData.location
                }
            });

            setShowHostAcceptModal(false);
            fetchEventDetails();
            fetchSwapRequests();
            showToast("You are now the host!", "success");
        } catch (error) {
            console.error("Failed to accept host swap", error);
            showToast("Failed to accept host swap", "error");
        }
    };

    const [showEditEventModal, setShowEditEventModal] = useState(false);
    const [editEventData, setEditEventData] = useState({ name: '', type: '', date: '', time: '', location: '', description: '', recurrence: '' });
    const [isEditCustomType, setIsEditCustomType] = useState(false);

    const handleEditEventClick = () => {
        const eventDate = new Date(event.date);
        setEditEventData({
            name: event.name || '',
            date: eventDate.toISOString().split('T')[0],
            time: eventDate.toTimeString().slice(0, 5),
            location: event.location,
            description: event.description || '',
            recurrence: event.recurrence || '',
            type: event.type || ''
        });
        const standardTypes = ['Dinner', 'Lunch', 'Coffee Meet', 'Picnic'];
        setIsEditCustomType(!standardTypes.includes(event.type));
        setShowEditEventModal(true);
    };

    const handleUpdateEvent = async (e) => {
        e.preventDefault();
        try {
            const dateTime = new Date(`${editEventData.date}T${editEventData.time}`);
            await api.patch(`/events/${eventId}`, {
                name: editEventData.name,
                type: editEventData.type,
                date: dateTime.toISOString(),
                location: editEventData.location,
                description: editEventData.description,
                recurrence: editEventData.recurrence,
                user_id: user.id
            });
            setShowEditEventModal(false);
            fetchEventDetails();
            showToast("Event details updated!", "success");
        } catch (error) {
            console.error("Failed to update event", error);
            showToast("Failed to update event", "error");
        }
    };

    const handleUpdateSwap = async (requestId, status) => {
        try {
            const payload = { status };
            if (status === 'approved') {
                payload.target_family_id = user.id;
            }
            await api.patch(`/swaps/${requestId}`, payload);
            fetchSwapRequests();
        } catch (error) {
            console.error("Failed to update swap request", error);
        }
    };

    const handlePledgeDish = async (dishId) => {
        try {
            await api.post(`/dishes/${dishId}/pledge`, { family_id: user.id });
            fetchDishes();
        } catch (error) {
            console.error("Failed to pledge dish", error);
        }
    };

    const handleUnpledgeDish = (dishId) => {
        confirm({
            title: "Unpledge Dish",
            message: "Are you sure you want to unpledge this dish?",
            onConfirm: () => executeUnpledgeDish(dishId),
            isDestructive: true
        });
    };

    const executeUnpledgeDish = async (dishId) => {
        try {
            await api.post(`/dishes/${dishId}/unpledge`);
            fetchDishes();
        } catch (error) {
            console.error("Failed to unpledge dish", error);
        }
    };

    const handleDeleteDish = (dishId) => {
        confirm({
            title: "Delete Dish",
            message: "Are you sure you want to delete this dish?",
            onConfirm: () => executeDeleteDish(dishId),
            isDestructive: true
        });
    };

    const executeDeleteDish = async (dishId) => {
        try {
            await api.delete(`/dishes/${dishId}?user_id=${user.id}`);
            fetchDishes();
        } catch (error) {
            console.error("Failed to delete dish", error);
        }
    };

    const handleAddDish = async (e) => {
        e.preventDefault();
        if (!newDish.name) return;
        setAddingDish(true);
        try {
            // Determine if this is a request (no bringer) or a pledge (current user)
            // If admin checks "Request this dish", bringer_id is null, is_requested is true
            // Otherwise, bringer_id is user.id, is_requested is false
            const isRequest = newDish.isRequest;

            await api.post('/dishes', {
                event_id: eventId,
                name: newDish.name,
                description: newDish.description,
                dietary_tags: newDish.dietary_tags,
                bringer_id: isRequest ? null : user.id,
                is_host_dish: false,
                is_requested: isRequest
            });
            setNewDish({ name: '', description: '', isRequest: false, dietary_tags: [] });
            fetchDishes();
            showToast("Dish added successfully!", "success");
        } catch (error) {
            console.error("Failed to add dish", error);
            showToast("Failed to add dish", "error");
        } finally {
            setAddingDish(false);
        }
    };

    useEffect(() => {
        fetchEventDetails();
        fetchDishes();
        fetchSwapRequests();
        fetchRSVPs();
        fetchEventStats();
    }, [eventId, fetchEventDetails, fetchDishes, fetchSwapRequests, fetchRSVPs, fetchEventStats]);

    useEffect(() => {
        if (event?.group_id) {
            fetchGroupMembers(event.group_id);
        }
    }, [event, fetchGroupMembers]);

    useEffect(() => {
        if (lastMessage) {
            console.log("WebSocket Message Received:", lastMessage);
            if (lastMessage.type === 'rsvp_updated') {
                const msgEventId = String(lastMessage.data.event_id);
                const currentEventId = String(eventId);

                if (msgEventId === currentEventId) {
                    console.log("Refreshing RSVPs...");
                    setTimeout(() => fetchRSVPs(), 100);

                    // Show toast
                    const familyName = lastMessage.data.family_name || "Someone";
                    const status = lastMessage.data.status;
                    if (status === 'going') {
                        toast.success(`${familyName} is going!`);
                    } else if (status === 'maybe') {
                        toast.info(`${familyName} might come.`);
                    } else if (status === 'not_going') {
                        toast.info(`${familyName} can't make it.`);
                    }
                }
            }
            if (['dish_added', 'dish_pledged', 'dish_unpledged', 'dish_deleted'].includes(lastMessage.type)) {
                const msgEventId = String(lastMessage.data.event_id);
                const currentEventId = String(eventId);

                if (msgEventId === currentEventId) {
                    if (lastMessage.type === 'dish_added') {
                        setDishes(prev => {
                            if (prev.some(d => d.id === lastMessage.data.id)) return prev;
                            return [...prev, lastMessage.data];
                        });
                        toast.success(`New dish added: ${lastMessage.data.name}`);
                    } else {
                        fetchDishes();
                    }

                    if (lastMessage.type === 'dish_pledged') {
                        const bringerName = lastMessage.data.bringer_name || "Someone";
                        const dishName = lastMessage.data.dish_name || "a dish";
                        toast.success(`${bringerName} is bringing ${dishName}!`);
                    } else if (lastMessage.type === 'dish_unpledged') {
                        const dishName = lastMessage.data.dish_name || "A dish";
                        toast.warning(`${dishName} is available again.`);
                    }
                }
            }
            if (lastMessage.type === 'suggestions_started') {
                const msgEventId = String(lastMessage.data.event_id);
                const currentEventId = String(eventId);
                if (msgEventId === currentEventId) {
                    setIsAiThinking(true);
                }
            }
            if (lastMessage.type === 'suggestions_finished') {
                const msgEventId = String(lastMessage.data.event_id);
                const currentEventId = String(eventId);
                if (msgEventId === currentEventId) {
                    setIsAiThinking(false);
                    fetchDishes();
                }
            }
            if (['swap_created', 'swap_updated'].includes(lastMessage.type)) {
                const msgEventId = String(lastMessage.data.event_id);
                const currentEventId = String(eventId);
                if (msgEventId === currentEventId) {
                    fetchSwapRequests();
                    fetchDishes();
                    if (lastMessage.type === 'swap_created') {
                        toast.info("New swap request received.");
                    } else if (lastMessage.type === 'swap_updated') {
                        toast.info(`Swap request ${lastMessage.data.status}.`);
                    }
                }
            }
            if (lastMessage.type === 'event_updated') {
                const msgEventId = String(lastMessage.data.id);
                const currentEventId = String(eventId);
                if (msgEventId === currentEventId) {
                    fetchEventDetails();
                    toast.info("Event details updated.");
                }
            }
        }
    }, [lastMessage, eventId, fetchRSVPs, fetchDishes, fetchSwapRequests, fetchEventDetails]);

    useEffect(() => {
        fetchGroups();
    }, [user, fetchGroups]);

    useEffect(() => {
        if (event && groups.length > 0) {
            const group = groups.find(g => g.id === event.group_id);
            setIsAdmin(group?.admin_ids?.includes(user.id) || group?.admin_id === user.id);
        }
    }, [event, groups, user]);

    useEffect(() => {
        if (event && groupMembers.families) {
            const host = groupMembers.families.find(m => m.id === event.host_id);
            if (host && user.household_id && host.household_id === user.household_id) {
                setIsHostHousehold(true);
            } else {
                setIsHostHousehold(false);
            }
        }
    }, [event, groupMembers, user]);

    if (loading) return <div className="flex justify-center items-center h-screen">Loading...</div>;
    if (!event) return <div className="text-center py-12">Event not found</div>;

    const requestedDishes = dishes.filter(d => !d.bringer_id && !d.is_suggested);
    const pledgedDishes = dishes.filter(d => d.bringer_id);
    const suggestedDishes = dishes.filter(d => d.is_suggested && !d.bringer_id);

    // Filter for active (pending) swap requests relevant to the current user
    const activeSwapRequests = swapRequests.filter(req =>
        req.status === 'pending' &&
        (
            req.target_family_id === user.id ||
            req.requesting_family_id === user.id ||
            (req.target_family_id === null && req.requesting_family_id !== user.id)
        )
    );

    const shareGuestLink = () => {
        const url = `${window.location.origin}/join-event/${event.guest_join_code}`;
        navigator.clipboard.writeText(url);
        setCopiedGuestLink(true);
        setTimeout(() => setCopiedGuestLink(false), 2000);
        showToast("Guest invite link copied!");
    };

    return (
        <><div className="min-h-screen bg-gray-50 pb-20">
            {/* Header and Event Info ... */}
            <header className="bg-white shadow-sm sticky top-0 z-10">
                <div className="max-w-3xl mx-auto px-4 py-4 flex items-center gap-4">
                    <button
                        onClick={() => navigate('/dashboard', { state: { groupId: event.group_id } })}
                        className="text-gray-500 hover:text-gray-800"
                    >
                        <ArrowLeft className="w-6 h-6" />
                    </button>
                    <h1 className="text-xl font-bold text-gray-800">Event Details</h1>
                </div>
            </header>

            <main className="max-w-3xl mx-auto px-4 py-8">
                {/* Event Details Card ... */}
                <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden mb-8">
                    <div className="h-4 bg-orange-500"></div>
                    <div className="p-6">
                        <div className="flex justify-between items-start mb-6">
                            <div>
                                <h2 className="text-2xl font-bold text-gray-800 mb-2">{event.name || event.type}</h2>
                                <div className="flex items-center gap-2 text-gray-600 mb-1">
                                    <Calendar className="w-5 h-5 text-orange-500" />
                                    <span>{new Date(event.date).toLocaleDateString()} at {new Date(event.date).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
                                </div>
                                <div className="flex items-center gap-2 text-gray-600">
                                    <span>{event.location}</span>
                                </div>
                                <div className="flex items-center gap-2 text-gray-600 mt-1">
                                    <User className="w-5 h-5 text-orange-500" />
                                    <span>Hosted by <span className="font-semibold">{event.host_name || 'Unknown'}</span></span>
                                </div>
                                {event.recurrence && (
                                    <button
                                        onClick={() => setShowStatsModal(true)}
                                        className="mt-3 flex items-center gap-1.5 text-sm font-medium text-orange-600 hover:text-orange-700 bg-orange-50 hover:bg-orange-100 px-3 py-1.5 rounded-lg transition w-fit"
                                    >
                                        <BarChart2 className="w-4 h-4" />
                                        View History
                                    </button>
                                )}
                            </div>
                            <div className="flex flex-col items-end gap-2">
                                {event.recurrence && (
                                    <span className="bg-blue-50 text-blue-600 text-xs px-3 py-1 rounded-full font-medium h-fit">
                                        {event.recurrence}
                                    </span>
                                )}
                                {(isAdmin || event.host_id === user.id || isHostHousehold) && (
                                    <button
                                        onClick={handleEditEventClick}
                                        className="text-gray-400 hover:text-orange-600 p-1 rounded-full hover:bg-orange-50 transition"
                                        title="Edit Event Details"
                                    >
                                        <Edit className="w-5 h-5" />
                                    </button>
                                )}
                            </div>
                        </div>

                        {(isAdmin || event.host_id === user.id || isHostHousehold) && (
                            <div className="mb-6">
                                <button
                                    onClick={shareGuestLink}
                                    className="flex items-center gap-2 text-sm text-blue-600 hover:text-blue-800 bg-blue-50 px-3 py-2 rounded-lg transition"
                                >
                                    {copiedGuestLink ? <CheckCircle className="w-4 h-4" /> : <Share2 className="w-4 h-4" />}
                                    {copiedGuestLink ? "Copied!" : "Share Guest Invite Link"}
                                </button>
                                <p className="text-xs text-gray-500 mt-1 ml-1">
                                    Share this link with non-family members (guests). They will only see this event.
                                </p>
                            </div>
                        )}

                        {event.description && (
                            <div className="bg-gray-50 p-4 rounded-lg mb-8">
                                <p className="text-gray-700 italic">"{event.description}"</p>
                            </div>
                        )}

                        <div className="border-t border-gray-100 pt-6">
                            {/* RSVP Section - Hide for Host */}
                            {event.host_id !== user.id && (
                                <>
                                    <h3 className="font-semibold text-gray-800 mb-4">Your RSVP</h3>
                                    <div className="flex gap-4 mb-6">
                                        <button
                                            onClick={() => handleRSVP('Yes')}
                                            className={`flex-1 py-3 rounded-lg flex items-center justify-center gap-2 font-medium transition ${rsvpStatus === 'Yes' ? 'bg-green-100 text-green-700 ring-2 ring-green-500' : 'bg-gray-50 text-gray-600 hover:bg-green-50 hover:text-green-700'}`}
                                        >
                                            <CheckCircle className="w-5 h-5" />
                                            Going
                                        </button>
                                        <button
                                            onClick={() => handleRSVP('Maybe')}
                                            className={`flex-1 py-3 rounded-lg flex items-center justify-center gap-2 font-medium transition ${rsvpStatus === 'Maybe' ? 'bg-yellow-100 text-yellow-700 ring-2 ring-yellow-500' : 'bg-gray-50 text-gray-600 hover:bg-yellow-50 hover:text-yellow-700'}`}
                                        >
                                            <HelpCircle className="w-5 h-5" />
                                            Maybe
                                        </button>
                                        <button
                                            onClick={() => handleRSVP('No')}
                                            className={`flex-1 py-3 rounded-lg flex items-center justify-center gap-2 font-medium transition ${rsvpStatus === 'No' ? 'bg-red-100 text-red-700 ring-2 ring-red-500' : 'bg-gray-50 text-gray-600 hover:bg-red-50 hover:text-red-700'}`}
                                        >
                                            <XCircle className="w-5 h-5" />
                                            Can't Go
                                        </button>
                                    </div>
                                    <div className="mb-6 text-center">
                                        <button
                                            onClick={() => setShowDietaryModal(true)}
                                            className="text-sm text-blue-600 hover:text-blue-800 hover:underline"
                                        >
                                            Edit My Dietary Preferences
                                        </button>
                                    </div>
                                </>
                            )}

                            <div className="mb-6">
                                <button
                                    onClick={() => setShowRSVPListModal(true)}
                                    className="w-full py-3 bg-gray-100 text-gray-700 rounded-lg font-medium hover:bg-gray-200 transition flex items-center justify-center gap-2"
                                >
                                    <User className="w-5 h-5" />
                                    View Who's Coming ({rsvps.filter(r => r.status === 'Yes').reduce((acc, curr) => acc + (curr.count || 0), 0)} Going)
                                </button>
                            </div>

                            {/* Host Swap Actions */}
                            {(event.host_id === user.id || isHostHousehold) && (
                                <div className="bg-orange-50 p-4 rounded-lg border border-orange-100">
                                    <h4 className="font-semibold text-orange-800 mb-2">Host Options</h4>
                                    <button
                                        onClick={handleRequestHostSwap}
                                        className="w-full bg-white border border-orange-200 text-orange-700 py-2 rounded-lg hover:bg-orange-100 transition flex items-center justify-center gap-2"
                                    >
                                        <RefreshCw className="w-4 h-4" />
                                        Request Host Swap (Ask for a volunteer)
                                    </button>
                                </div>
                            )}

                            {/* Show "Accept Host Duty" if there is an open host swap request */}
                            {swapRequests.some(req => req.type === 'host' && req.status === 'pending' && req.requesting_family_id !== user.id) && (
                                <div className="bg-blue-50 p-4 rounded-lg border border-blue-100 mt-4">
                                    <h4 className="font-semibold text-blue-800 mb-2">Host Needed!</h4>
                                    <p className="text-sm text-blue-600 mb-3">The current host is asking for a volunteer to take over.</p>
                                    <button
                                        onClick={() => handleAcceptHostSwapClick(swapRequests.find(req => req.type === 'host' && req.status === 'pending').id)}
                                        className="w-full bg-blue-600 text-white py-2 rounded-lg hover:bg-blue-700 transition"
                                    >
                                        Volunteer to Host
                                    </button>
                                </div>
                            )}
                        </div>
                    </div>
                </div>

                {/* Host Summary */}
                {(event.host_id === user.id || isHostHousehold) && (
                    <HostSummary rsvps={rsvps} dishes={dishes} />
                )}

                {/* Swap Requests Section */}
                {activeSwapRequests.length > 0 && (
                    <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 mb-8">
                        <div className="flex items-center gap-3 mb-6">
                            <div className="bg-blue-100 p-2 rounded-lg">
                                <RefreshCw className="w-6 h-6 text-blue-600" />
                            </div>
                            <h3 className="text-lg font-bold text-gray-800">Swap Requests</h3>
                        </div>
                        <div className="space-y-3">
                            {activeSwapRequests.map(req => {
                                const isIncoming = req.target_family_id === user.id;
                                const isOutgoing = req.requesting_family_id === user.id;
                                const isOpenOffer = req.target_family_id === null && !isOutgoing;

                                if (!isIncoming && !isOutgoing && !isOpenOffer) return null;

                                return (
                                    <div key={req.id} className="flex justify-between items-center p-3 border border-blue-100 bg-blue-50/50 rounded-lg">
                                        <div>
                                            <p className="text-sm text-gray-800">
                                                {isIncoming ? (
                                                    <span><strong>{req.requesting_family_name || 'Someone'}</strong> wants to swap with you.</span>
                                                ) : isOpenOffer ? (
                                                    <span><strong>{req.requesting_family_name || 'Someone'}</strong> is offering a dish for swap.</span>
                                                ) : (
                                                    <span>You requested to swap {req.target_family_id ? 'with' : 'dish'} <strong>{req.target_family_name || (req.target_family_id ? 'Someone' : '(Open Offer)')}</strong>.</span>
                                                )}
                                            </p>
                                            <span className={`text-xs font-medium px-2 py-0.5 rounded-full ${req.status === 'pending' ? 'bg-yellow-100 text-yellow-700' :
                                                req.status === 'approved' ? 'bg-green-100 text-green-700' :
                                                    'bg-red-100 text-red-700'}`}>
                                                {req.status.charAt(0).toUpperCase() + req.status.slice(1)}
                                            </span>
                                        </div>
                                        {(isIncoming || isOpenOffer) && req.status === 'pending' && (
                                            <div className="flex gap-2">
                                                <button
                                                    onClick={() => handleUpdateSwap(req.id, 'approved')}
                                                    className="text-xs bg-green-600 text-white px-3 py-1.5 rounded-lg hover:bg-green-700 transition"
                                                >
                                                    Accept
                                                </button>
                                            </div>
                                        )}
                                        {isOutgoing && req.status === 'pending' && (
                                            <div className="flex gap-2">
                                                <button
                                                    onClick={() => handleUpdateSwap(req.id, 'cancelled')}
                                                    className="text-xs bg-gray-200 text-gray-700 px-3 py-1.5 rounded-lg hover:bg-gray-300 transition"
                                                >
                                                    Cancel
                                                </button>
                                            </div>
                                        )}
                                    </div>
                                );
                            })}
                        </div>
                    </div>
                )}


                {/* Dishes Section */}
                <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6">
                    <div className="flex items-center gap-3 mb-6">
                        <div className="bg-orange-100 p-2 rounded-lg">
                            <ChefHat className="w-6 h-6 text-orange-600" />
                        </div>
                        <h3 className="text-lg font-bold text-gray-800">Potluck Dishes</h3>
                    </div>

                    {/* AI Suggested Dishes */}
                    {(suggestedDishes.length > 0 || isAiThinking) && (
                        <div className="mb-8">
                            <div className="flex items-center justify-between mb-3 border-b border-gray-100 pb-2">
                                <h4 className="font-semibold text-gray-700 flex items-center gap-2">
                                    <Sparkles className="w-4 h-4 text-purple-500" />
                                    AI Suggested Dishes
                                </h4>
                                {isAiThinking ? (
                                    <div className="flex items-center gap-2 text-[10px] text-purple-600 font-medium animate-pulse">
                                        <RefreshCw className="w-3 h-3 animate-spin" />
                                        AI is thinking...
                                    </div>
                                ) : (
                                    <span className="text-[10px] text-gray-400 italic">AI generated suggestions</span>
                                )}
                            </div>

                            {suggestedDishes.length > 0 ? (
                                <div className="space-y-3">
                                    {suggestedDishes.map(dish => (
                                        <div key={dish.id} className="flex justify-between items-center p-3 border border-purple-100 bg-purple-50/30 rounded-lg animate-in fade-in slide-in-from-left-2">
                                            <div>
                                                <h4 className="font-medium text-gray-800">{dish.name}</h4>
                                                {dish.description && <p className="text-sm text-gray-500">{dish.description}</p>}
                                                {dish.dietary_tags && dish.dietary_tags.length > 0 && (
                                                    <div className="flex flex-wrap gap-1 mt-1">
                                                        {dish.dietary_tags.map(tag => (
                                                            <span key={tag} className="px-2 py-0.5 bg-green-50 text-green-700 text-[10px] rounded-full border border-green-100">
                                                                {tag}
                                                            </span>
                                                        ))}
                                                    </div>
                                                )}
                                            </div>
                                            <div className="flex items-center gap-2">
                                                <button
                                                    onClick={() => handlePledgeDish(dish.id)}
                                                    className="text-sm bg-purple-600 text-white px-3 py-1.5 rounded-lg hover:bg-purple-700 transition"
                                                >
                                                    I'll Bring This
                                                </button>
                                                {(isAdmin || event.host_id === user.id || isHostHousehold) && (
                                                    <button
                                                        onClick={() => handleDeleteDish(dish.id)}
                                                        className="text-gray-400 hover:text-red-500 p-1"
                                                        title="Delete suggestion"
                                                    >
                                                        <Trash2 className="w-4 h-4" />
                                                    </button>
                                                )}
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            ) : isAiThinking && (
                                <div className="p-8 text-center bg-purple-50/20 rounded-lg border border-dashed border-purple-100 animate-pulse">
                                    <Sparkles className="w-8 h-8 text-purple-300 mx-auto mb-2" />
                                    <p className="text-sm text-purple-400">Generating delicious suggestions for your event...</p>
                                </div>
                            )}

                            {!isAiThinking && suggestedDishes.length > 0 && (
                                <p className="text-[10px] text-gray-400 mt-2 italic">
                                    * These dishes were suggested by AI based on the event details. Please verify ingredients and dietary tags.
                                </p>
                            )}
                        </div>
                    )}

                    {/* Requested Dishes */}
                    {requestedDishes.length > 0 && (
                        <div className="mb-8">
                            <h4 className="font-semibold text-gray-700 mb-3 border-b border-gray-100 pb-2">Requested Items</h4>
                            <div className="space-y-3">
                                {requestedDishes.map(dish => (
                                    <div key={dish.id} className="flex justify-between items-center p-3 border border-orange-100 bg-orange-50/50 rounded-lg">
                                        <div>
                                            <h4 className="font-medium text-gray-800">{dish.name}</h4>
                                            {dish.description && <p className="text-sm text-gray-500">{dish.description}</p>}
                                            {dish.dietary_tags && dish.dietary_tags.length > 0 && (
                                                <div className="flex flex-wrap gap-1 mt-1">
                                                    {dish.dietary_tags.map(tag => (
                                                        <span key={tag} className="px-2 py-0.5 bg-green-50 text-green-700 text-[10px] rounded-full border border-green-100">
                                                            {tag}
                                                        </span>
                                                    ))}
                                                </div>
                                            )}
                                        </div>
                                        <div className="flex items-center gap-2">
                                            <button
                                                onClick={() => handlePledgeDish(dish.id)}
                                                className="text-sm bg-orange-600 text-white px-3 py-1.5 rounded-lg hover:bg-orange-700 transition"
                                            >
                                                Pledge
                                            </button>
                                            {(isAdmin || event.host_id === user.id || isHostHousehold) && (
                                                <button
                                                    onClick={() => handleDeleteDish(dish.id)}
                                                    className="text-gray-400 hover:text-red-500 p-1"
                                                >
                                                    <XCircle className="w-5 h-5" />
                                                </button>
                                            )}
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}

                    {/* Pledged Dishes */}
                    <div className="mb-8">
                        <h4 className="font-semibold text-gray-700 mb-3 border-b border-gray-100 pb-2">Who's Bringing What</h4>
                        <div className="space-y-3">
                            {pledgedDishes.length === 0 ? (
                                <p className="text-gray-500 text-sm italic">No dishes pledged yet.</p>
                            ) : (
                                pledgedDishes.map(dish => (
                                    <div key={dish.id} className="flex justify-between items-center p-3 border border-gray-100 rounded-lg bg-gray-50">
                                        <div>
                                            <h4 className="font-medium text-gray-800">{dish.name}</h4>
                                            {dish.description && <p className="text-sm text-gray-500">{dish.description}</p>}
                                            {dish.dietary_tags && dish.dietary_tags.length > 0 && (
                                                <div className="flex flex-wrap gap-1 mt-1">
                                                    {dish.dietary_tags.map(tag => (
                                                        <span key={tag} className="px-2 py-0.5 bg-green-50 text-green-700 text-[10px] rounded-full border border-green-100">
                                                            {tag}
                                                        </span>
                                                    ))}
                                                </div>
                                            )}
                                        </div>
                                        <div className="text-right flex flex-col items-end">
                                            <span className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Brought by</span>
                                            <div className="flex items-center gap-2">
                                                <span className="text-sm font-medium text-orange-600 bg-orange-50 px-2 py-1 rounded-full flex items-center gap-1">
                                                    {dish.bringer_id === user.id ? 'You' : (dish.bringer_name || 'Someone')}
                                                    {dish.is_suggested && <Sparkles className="w-3 h-3 text-purple-400" title="AI Suggested" />}
                                                </span>
                                                {dish.bringer_id !== user.id && (
                                                    <button
                                                        onClick={() => handleRequestSwap(dish)}
                                                        className="ml-2 text-gray-400 hover:text-blue-500"
                                                        title="Request Swap"
                                                    >
                                                        <RefreshCw className="w-4 h-4" />
                                                    </button>
                                                )}
                                                {dish.bringer_id === user.id && (
                                                    <button
                                                        onClick={() => handleRequestSwap(dish)}
                                                        className="ml-2 text-gray-400 hover:text-blue-500"
                                                        title="Offer for Swap"
                                                    >
                                                        <RefreshCw className="w-4 h-4" />
                                                    </button>
                                                )}
                                                {(dish.bringer_id === user.id || isAdmin || event.host_id === user.id || isHostHousehold) && (
                                                    <div className="flex gap-1">
                                                        {dish.is_requested && dish.bringer_id === user.id && (
                                                            <button
                                                                onClick={() => handleUnpledgeDish(dish.id)}
                                                                className="text-xs text-blue-600 hover:underline"
                                                            >
                                                                Unpledge
                                                            </button>
                                                        )}
                                                        <button
                                                            onClick={() => handleDeleteDish(dish.id)}
                                                            className="text-gray-400 hover:text-red-500"
                                                            title="Delete"
                                                        >
                                                            <XCircle className="w-4 h-4" />
                                                        </button>
                                                    </div>
                                                )}
                                            </div>
                                        </div>
                                    </div>
                                ))
                            )}
                        </div>
                    </div>

                    <div className="border-t border-gray-100 pt-6">
                        <h4 className="font-semibold text-gray-800 mb-4">Add a Dish</h4>
                        <form onSubmit={handleAddDish} className="space-y-4">
                            <div>
                                <input
                                    type="text"
                                    placeholder="Dish Name (e.g. Lasagna)"
                                    className="w-full p-3 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 outline-none"
                                    value={newDish.name}
                                    onChange={e => setNewDish({ ...newDish, name: e.target.value })}
                                    required />
                            </div>
                            <div>
                                <textarea
                                    placeholder="Description (optional, e.g. Vegetarian, Contains Nuts)"
                                    className="w-full p-3 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 outline-none"
                                    rows="2"
                                    value={newDish.description}
                                    onChange={e => setNewDish({ ...newDish, description: e.target.value })} />
                            </div>

                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-2">Dietary Tags</label>
                                <div className="flex flex-wrap gap-2">
                                    {/* Predefined Tags */}
                                    {DIETARY_TAGS.map(tag => (
                                        <button
                                            key={tag}
                                            type="button"
                                            onClick={() => {
                                                const currentTags = newDish.dietary_tags || [];
                                                const newTags = currentTags.includes(tag)
                                                    ? currentTags.filter(t => t !== tag)
                                                    : [...currentTags, tag];
                                                setNewDish({ ...newDish, dietary_tags: newTags });
                                            }}
                                            className={`px-3 py-1 rounded-full text-xs font-medium border transition ${(newDish.dietary_tags || []).includes(tag)
                                                ? 'bg-green-100 text-green-700 border-green-200'
                                                : 'bg-white text-gray-600 border-gray-200 hover:border-gray-300'
                                                }`}
                                        >
                                            {tag}
                                        </button>
                                    ))}

                                    {/* Custom Tags Display */}
                                    {(newDish.dietary_tags || []).filter(tag => !DIETARY_TAGS.includes(tag)).map(tag => (
                                        <button
                                            key={tag}
                                            type="button"
                                            onClick={() => {
                                                const newTags = newDish.dietary_tags.filter(t => t !== tag);
                                                setNewDish({ ...newDish, dietary_tags: newTags });
                                            }}
                                            className="px-3 py-1 rounded-full text-xs font-medium border bg-green-100 text-green-700 border-green-200 flex items-center gap-1"
                                        >
                                            {tag}
                                            <XCircle className="w-3 h-3" />
                                        </button>
                                    ))}

                                    {/* Add Custom Tag Input */}
                                    <div className="flex items-center gap-2">
                                        <input
                                            type="text"
                                            placeholder="Add custom tag..."
                                            className="px-3 py-1 text-xs border border-gray-200 rounded-full focus:ring-2 focus:ring-orange-500 outline-none w-32"
                                            value={customTag}
                                            onChange={e => setCustomTag(e.target.value)}
                                            onKeyDown={e => {
                                                if (e.key === 'Enter') {
                                                    e.preventDefault();
                                                    if (customTag.trim()) {
                                                        const tag = customTag.trim();
                                                        const currentTags = newDish.dietary_tags || [];
                                                        if (!currentTags.includes(tag)) {
                                                            setNewDish({ ...newDish, dietary_tags: [...currentTags, tag] });
                                                        }
                                                        setCustomTag('');
                                                    }
                                                }
                                            }}
                                        />
                                        <button
                                            type="button"
                                            onClick={() => {
                                                if (customTag.trim()) {
                                                    const tag = customTag.trim();
                                                    const currentTags = newDish.dietary_tags || [];
                                                    if (!currentTags.includes(tag)) {
                                                        setNewDish({ ...newDish, dietary_tags: [...currentTags, tag] });
                                                    }
                                                    setCustomTag('');
                                                }
                                            }}
                                            className="p-1 text-orange-600 hover:bg-orange-50 rounded-full"
                                        >
                                            <Plus className="w-4 h-4" />
                                        </button>
                                    </div>
                                </div>
                            </div>

                            {(isAdmin || event.host_id === user.id || isHostHousehold) && (
                                <div className="flex items-center gap-2">
                                    <input
                                        type="checkbox"
                                        id="isRequest"
                                        checked={newDish.isRequest || false}
                                        onChange={e => setNewDish({ ...newDish, isRequest: e.target.checked })}
                                        className="w-4 h-4 text-orange-600 rounded focus:ring-orange-500" />
                                    <label htmlFor="isRequest" className="text-sm text-gray-700">Request this dish (ask others to bring it)</label>
                                </div>
                            )}

                            <button
                                type="submit"
                                disabled={addingDish}
                                className="w-full bg-orange-600 text-white py-3 rounded-lg font-medium hover:bg-orange-700 transition disabled:opacity-70"
                            >
                                {addingDish ? 'Adding...' : (newDish.isRequest ? 'Add Request' : 'Pledge Dish')}
                            </button>
                        </form>
                    </div>
                </div>
            </main>
        </div>

            {/* RSVP Modal */}
            {
                showRSVPModal && (
                    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
                        <div className="bg-white rounded-xl shadow-xl max-w-sm w-full p-6">
                            <h3 className="text-xl font-bold text-gray-800 mb-4">How many are coming?</h3>
                            <div className="space-y-4">
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">Total Adults (including you)</label>
                                    <input
                                        type="number"
                                        min="1"
                                        className="w-full p-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 outline-none"
                                        value={rsvpData.count}
                                        onChange={e => setRsvpData({ ...rsvpData, count: e.target.value })} />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">Kids (under 12)</label>
                                    <input
                                        type="number"
                                        min="0"
                                        className="w-full p-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 outline-none"
                                        value={rsvpData.kidsCount}
                                        onChange={e => setRsvpData({ ...rsvpData, kidsCount: e.target.value })} />
                                </div>
                                <div className="flex gap-3 mt-6">
                                    <button
                                        onClick={() => setShowRSVPModal(false)}
                                        className="flex-1 py-2 text-gray-600 hover:bg-gray-100 rounded-lg transition"
                                    >
                                        Cancel
                                    </button>
                                    <button
                                        onClick={() => submitRSVP('Yes', rsvpData.count, rsvpData.kidsCount)}
                                        className="flex-1 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition font-medium"
                                    >
                                        Confirm
                                    </button>
                                </div>
                            </div>
                        </div>
                    </div>
                )
            }

            {/* Host Accept Modal */}
            {
                showHostAcceptModal && (
                    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
                        <div className="bg-white rounded-xl shadow-xl max-w-md w-full p-6">
                            <h3 className="text-xl font-bold text-gray-800 mb-4">Confirm Host Swap</h3>
                            <p className="text-gray-600 mb-4">
                                You are volunteering to host this event. You can update the event details if needed.
                            </p>

                            <div className="space-y-4 mb-6">
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">Date</label>
                                    <input
                                        type="date"
                                        value={hostUpdateData.date}
                                        onChange={(e) => setHostUpdateData({ ...hostUpdateData, date: e.target.value })}
                                        className="w-full px-4 py-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none" />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">Time</label>
                                    <input
                                        type="time"
                                        value={hostUpdateData.time}
                                        onChange={(e) => setHostUpdateData({ ...hostUpdateData, time: e.target.value })}
                                        className="w-full px-4 py-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none" />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">Location</label>
                                    <input
                                        type="text"
                                        value={hostUpdateData.location}
                                        onChange={(e) => setHostUpdateData({ ...hostUpdateData, location: e.target.value })}
                                        className="w-full px-4 py-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none" />
                                </div>
                            </div>

                            <div className="flex gap-3">
                                <button
                                    onClick={() => setShowHostAcceptModal(false)}
                                    className="flex-1 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50 transition"
                                >
                                    Cancel
                                </button>
                                <button
                                    onClick={confirmAcceptHostSwap}
                                    className="flex-1 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
                                >
                                    Confirm & Host
                                </button>
                            </div>
                        </div>
                    </div>
                )
            }

            {/* Edit Event Modal */}
            {showEditEventModal && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
                    <div className="bg-white rounded-xl shadow-xl max-w-md w-full p-6">
                        <h3 className="text-xl font-bold text-gray-800 mb-4">Edit Event Details</h3>
                        <form onSubmit={handleUpdateEvent} className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">Event Name</label>
                                <input
                                    type="text"
                                    value={editEventData.name}
                                    onChange={(e) => setEditEventData({ ...editEventData, name: e.target.value })}
                                    className="w-full px-4 py-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 outline-none"
                                    required
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">Date</label>
                                <input
                                    type="date"
                                    value={editEventData.date}
                                    onChange={(e) => setEditEventData({ ...editEventData, date: e.target.value })}
                                    className="w-full px-4 py-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 outline-none"
                                    required
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">Type</label>
                                <select
                                    className="w-full px-4 py-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 outline-none"
                                    value={isEditCustomType ? 'Other' : editEventData.type}
                                    onChange={e => {
                                        if (e.target.value === 'Other') {
                                            setIsEditCustomType(true);
                                            setEditEventData({ ...editEventData, type: '' });
                                        } else {
                                            setIsEditCustomType(false);
                                            setEditEventData({ ...editEventData, type: e.target.value });
                                        }
                                    }}
                                >
                                    <option>Dinner</option>
                                    <option>Lunch</option>
                                    <option>Coffee Meet</option>
                                    <option>Picnic</option>
                                    <option>Other</option>
                                </select>
                                {isEditCustomType && (
                                    <input
                                        type="text"
                                        placeholder="Enter custom type"
                                        className="w-full mt-2 px-4 py-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 outline-none"
                                        value={editEventData.type}
                                        onChange={e => setEditEventData({ ...editEventData, type: e.target.value })}
                                        required
                                    />
                                )}
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">Time</label>
                                <input
                                    type="time"
                                    value={editEventData.time}
                                    onChange={(e) => setEditEventData({ ...editEventData, time: e.target.value })}
                                    className="w-full px-4 py-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 outline-none"
                                    required
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">Location</label>
                                <input
                                    type="text"
                                    value={editEventData.location}
                                    onChange={(e) => setEditEventData({ ...editEventData, location: e.target.value })}
                                    className="w-full px-4 py-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 outline-none"
                                    required
                                />
                            </div>
                            <div className="flex items-center gap-2">
                                <input
                                    type="checkbox"
                                    id="editIsRecurring"
                                    className="w-4 h-4 text-orange-600 border-gray-300 rounded focus:ring-orange-500"
                                    checked={!!editEventData.recurrence}
                                    onChange={e => setEditEventData({ ...editEventData, recurrence: e.target.checked ? 'Weekly' : '' })}
                                />
                                <label htmlFor="editIsRecurring" className="text-sm font-medium text-gray-700">Recurring Event</label>
                            </div>
                            {editEventData.recurrence && (
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-1">Frequency</label>
                                    <select
                                        className="w-full px-4 py-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 outline-none"
                                        value={editEventData.recurrence}
                                        onChange={e => setEditEventData({ ...editEventData, recurrence: e.target.value })}
                                    >
                                        <option value="Daily">Daily</option>
                                        <option value="Weekly">Weekly</option>
                                        <option value="Bi-Weekly">Bi-Weekly</option>
                                        <option value="Monthly">Monthly</option>
                                    </select>
                                </div>
                            )}
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
                                <textarea
                                    value={editEventData.description}
                                    onChange={(e) => setEditEventData({ ...editEventData, description: e.target.value })}
                                    className="w-full px-4 py-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 outline-none"
                                    rows="3"
                                />
                            </div>
                            <div className="flex gap-3 mt-6">
                                <button
                                    type="button"
                                    onClick={() => setShowEditEventModal(false)}
                                    className="flex-1 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50 transition"
                                >
                                    Cancel
                                </button>
                                <button
                                    type="submit"
                                    className="flex-1 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700 transition"
                                >
                                    Save Changes
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            )}

            <EventStatsModal
                isOpen={showStatsModal}
                onClose={() => setShowStatsModal(false)}
                stats={eventStats}
                eventType={event.type}
            />

            <EventRSVPModal
                isOpen={showRSVPListModal}
                onClose={() => setShowRSVPListModal(false)}
                rsvps={rsvps}
                groupMembers={groupMembers}
                hostId={event.host_id}
            />

            <DietaryPreferencesModal
                isOpen={showDietaryModal}
                onClose={() => setShowDietaryModal(false)}
                user={user}
                onUpdate={() => {
                    fetchRSVPs(); // Refresh to update summary
                    showToast("Preferences updated!", "success");
                }}
            />
        </>
    );
};

export default EventDetails;
