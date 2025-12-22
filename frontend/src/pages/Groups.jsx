import React, { useState, useEffect, useCallback } from 'react';
import { useAuth } from '../context/AuthContext';
import { useUI } from '../context/UIContext';
import api from '../api/axios';
import { useNavigate } from 'react-router-dom';
import { Plus, Trash2, Share2, Copy, LogOut, Users } from 'lucide-react';
import ManageHouseholdModal from '../components/ManageHouseholdModal';

const Groups = () => {
    const { user, refreshUser } = useAuth();
    const { showToast, confirm } = useUI();
    const navigate = useNavigate();
    const [groups, setGroups] = useState([]);
    const [newGroupName, setNewGroupName] = useState('');
    const [loading, setLoading] = useState(true);
    const [copiedId, setCopiedId] = useState(null);
    const [joinCodeInput, setJoinCodeInput] = useState('');
    const [joining, setJoining] = useState(false);
    const [manageHouseholdModalOpen, setManageHouseholdModalOpen] = useState(false);
    const [manageHouseholdId, setManageHouseholdId] = useState(null);

    const fetchGroups = useCallback(async () => {
        try {
            const response = await api.get('/groups');
            // Only show groups the user is a member of
            const myGroups = (response.data || []).filter(g => user?.group_ids?.includes(g.id));
            setGroups(myGroups);
        } catch (error) {
            console.error("Failed to fetch groups", error);
        } finally {
            setLoading(false);
        }
    }, [user?.group_ids]);

    useEffect(() => {
        fetchGroups();
    }, [user, navigate, fetchGroups]);

    const createGroup = async (e) => {
        e.preventDefault();
        if (!newGroupName) return;
        try {
            await api.post('/groups', {
                name: newGroupName,
                admin_id: user.id
            });
            fetchGroups();
            setNewGroupName('');
            await refreshUser();
            navigate('/dashboard');
            showToast("Group created successfully!");
        } catch (error) {
            console.error("Failed to create group", error);
            showToast("Failed to create group", "error");
        }
    };

    const deleteGroup = async (groupId) => {
        confirm({
            title: "Delete Group",
            message: "Are you sure you want to delete this group? This action cannot be undone.",
            confirmText: "Delete",
            isDestructive: true,
            onConfirm: async () => {
                try {
                    await api.delete(`/groups/${groupId}?admin_id=${user.id}`);
                    fetchGroups();
                    await refreshUser();
                    showToast("Group deleted successfully");
                } catch (error) {
                    console.error("Failed to delete group", error);
                    showToast("Failed to delete group", "error");
                }
            }
        });
    };

    const confirmLeaveGroup = (group) => {
        confirm({
            title: "Leave Group",
            message: `Are you sure you want to leave ${group?.name}? You will need a new invite code to rejoin.`,
            confirmText: "Leave Group",
            isDestructive: true,
            onConfirm: () => handleLeaveGroup(group)
        });
    };

    const handleLeaveGroup = async (group) => {
        try {
            await api.post('/groups/leave', {
                family_id: user.id,
                group_id: group.id
            });
            fetchGroups();
            await refreshUser();
            showToast(`You have left ${group.name}`);
        } catch (error) {
            console.error("Failed to leave group", error);
            showToast("Failed to leave group", "error");
        }
    };

    const shareGroup = (group) => {
        const url = `${window.location.origin}/join/${group.join_code}`;
        navigator.clipboard.writeText(url);
        setCopiedId(group.id);
        setTimeout(() => setCopiedId(null), 2000);
        showToast("Invite link copied to clipboard!");
    };

    const handleJoinByCode = async (e) => {
        e.preventDefault();
        if (!joinCodeInput) return;
        setJoining(true);
        try {
            await api.post('/groups/join-by-code', {
                family_id: user.id,
                join_code: joinCodeInput
            });
            await refreshUser();
            fetchGroups();
            setJoinCodeInput('');
            navigate('/dashboard');
            showToast("Successfully joined group!");
        } catch (error) {
            console.error("Failed to join group", error);
            showToast("Failed to join group. Please check the code.", "error");
        } finally {
            setJoining(false);
        }
    };

    const [viewMembersModalOpen, setViewMembersModalOpen] = useState(false);
    const [currentGroupMembers, setCurrentGroupMembers] = useState([]);
    const [viewingGroup, setViewingGroup] = useState(null);

    const handleViewMembers = async (group) => {
        setViewingGroup(group);
        setViewMembersModalOpen(true);
        setCurrentGroupMembers([]); // Clear previous
        try {
            const response = await api.get(`/groups/members?group_id=${group.id}`);
            const { families, households } = response.data;

            const householdMap = new Map();
            if (households) {
                households.forEach(h => householdMap.set(h.id, { ...h, members: [], isHousehold: true }));
            }

            const standaloneFamilies = [];
            if (families) {
                families.forEach(f => {
                    if (f.household_id && householdMap.has(f.household_id)) {
                        householdMap.get(f.household_id).members.push(f);
                    } else {
                        standaloneFamilies.push(f);
                    }
                });
            }

            const displayList = [...standaloneFamilies, ...householdMap.values()];
            setCurrentGroupMembers(displayList);
        } catch (error) {
            console.error("Failed to fetch group members", error);
            showToast("Failed to fetch group members", "error");
        }
    };

    if (loading) return <div className="flex justify-center items-center h-screen">Loading...</div>;

    return (
        <div className="min-h-screen bg-gray-50 flex flex-col items-center py-12 px-4">
            <div className="w-full max-w-4xl">
                <div className="text-center mb-12">
                    <h1 className="text-3xl font-bold text-gray-800 mb-2">Welcome, {user?.name}!</h1>
                    <p className="text-gray-600 mb-4">Manage your gathering circles.</p>
                    <button
                        onClick={() => setManageHouseholdModalOpen(true)}
                        className="text-sm text-purple-600 font-medium hover:text-purple-800 flex items-center justify-center gap-1 mx-auto"
                    >
                        <Users className="w-4 h-4" />
                        Manage Household
                    </button>
                </div>

                <div className="grid md:grid-cols-2 gap-8">
                    {/* Create Group */}
                    <div className="bg-white p-8 rounded-2xl shadow-sm border border-gray-100">
                        <div className="flex items-center gap-4 mb-6">
                            <div className="bg-orange-100 p-3 rounded-full">
                                <Plus className="w-6 h-6 text-orange-600" />
                            </div>
                            <h2 className="text-xl font-semibold text-gray-800">Create a New Group</h2>
                        </div>
                        <p className="text-gray-500 mb-6">Start a new gathering circle for your family.</p>
                        <form onSubmit={createGroup}>
                            <input
                                type="text"
                                placeholder="Group Name (e.g. Smith Family Dinners)"
                                className="w-full p-3 border border-gray-200 rounded-lg mb-4 focus:ring-2 focus:ring-orange-500 outline-none"
                                value={newGroupName}
                                onChange={(e) => setNewGroupName(e.target.value)}
                                required
                            />
                            <button type="submit" className="w-full bg-orange-600 text-white py-3 rounded-lg font-medium hover:bg-orange-700 transition">
                                Create Group
                            </button>
                        </form>
                    </div>

                    {/* My Groups List */}
                    <div className="bg-white p-8 rounded-2xl shadow-sm border border-gray-100">
                        <div className="flex items-center gap-4 mb-6">
                            <div className="bg-blue-100 p-3 rounded-full">
                                <Share2 className="w-6 h-6 text-blue-600" />
                            </div>
                            <h2 className="text-xl font-semibold text-gray-800">Your Groups</h2>
                        </div>
                        <p className="text-gray-500 mb-6">Manage and share your circles.</p>

                        <div className="space-y-3 max-h-60 overflow-y-auto pr-2 mb-6">
                            {groups.map(group => {
                                const isAdmin = group.admin_id === user.id;
                                return (
                                    <div key={group.id} className="flex justify-between items-center p-3 border border-gray-100 rounded-lg hover:bg-gray-50">
                                        <span className="font-medium text-gray-700">{group.name}</span>
                                        <div className="flex items-center gap-2">
                                            <button
                                                onClick={() => handleViewMembers(group)}
                                                className="p-1 text-gray-400 hover:text-blue-500 transition rounded-full hover:bg-blue-50"
                                                title="View Members"
                                            >
                                                <Users className="w-4 h-4" />
                                            </button>
                                            {isAdmin ? (
                                                <>
                                                    <button
                                                        onClick={() => shareGroup(group)}
                                                        className="p-1 text-gray-400 hover:text-blue-500 transition rounded-full hover:bg-blue-50 relative"
                                                        title="Copy Invite Link"
                                                    >
                                                        {copiedId === group.id ? (
                                                            <span className="text-xs font-bold text-green-600">Copied!</span>
                                                        ) : (
                                                            <Copy className="w-4 h-4" />
                                                        )}
                                                    </button>
                                                    <button
                                                        onClick={() => deleteGroup(group.id)}
                                                        className="p-1 text-gray-400 hover:text-red-500 transition rounded-full hover:bg-red-50"
                                                        title="Delete Group"
                                                    >
                                                        <Trash2 className="w-4 h-4" />
                                                    </button>
                                                </>
                                            ) : (
                                                <button
                                                    onClick={() => confirmLeaveGroup(group)}
                                                    className="p-1 text-gray-400 hover:text-red-500 transition rounded-full hover:bg-red-50"
                                                    title="Leave Group"
                                                >
                                                    <LogOut className="w-4 h-4" />
                                                </button>
                                            )}
                                        </div>
                                    </div>
                                );
                            })}
                            {groups.length === 0 && (
                                <p className="text-center text-gray-400 text-sm">You haven't joined any groups yet.</p>
                            )}
                        </div>

                        <div className="border-t border-gray-100 pt-6">
                            <h3 className="text-sm font-semibold text-gray-700 mb-3">Join by Code</h3>
                            <form onSubmit={handleJoinByCode} className="flex gap-2">
                                <input
                                    type="text"
                                    placeholder="Enter 6-digit code"
                                    className="flex-1 p-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-orange-500 outline-none text-sm uppercase"
                                    value={joinCodeInput}
                                    onChange={(e) => setJoinCodeInput(e.target.value.toUpperCase())}
                                    maxLength={6}
                                />
                                <button
                                    type="submit"
                                    disabled={joining || !joinCodeInput}
                                    className="bg-gray-800 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-gray-900 transition disabled:opacity-50"
                                >
                                    {joining ? 'Joining...' : 'Join'}
                                </button>
                            </form>
                        </div>
                    </div>
                </div>

                {user?.group_ids?.length > 0 && (
                    <div className="mt-12 text-center">
                        <button onClick={() => navigate('/dashboard')} className="text-gray-500 hover:text-gray-800 underline">
                            Go to Dashboard
                        </button>
                    </div>
                )}
            </div>

            {/* Members Modal */}
            {viewMembersModalOpen && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50 animate-in fade-in duration-200">
                    <div className="bg-white rounded-xl shadow-xl max-w-md w-full p-6 transform transition-all scale-100">
                        <div className="flex justify-between items-center mb-6">
                            <h3 className="text-xl font-bold text-gray-800">Members of {viewingGroup?.name}</h3>
                            <button onClick={() => setViewMembersModalOpen(false)} className="text-gray-400 hover:text-gray-600 transition">
                                <Plus className="w-6 h-6 rotate-45" />
                            </button>
                        </div>

                        <div className="space-y-3 max-h-80 overflow-y-auto pr-2">
                            {currentGroupMembers.length === 0 ? (
                                <p className="text-gray-500 text-center py-4">Loading members...</p>
                            ) : (
                                currentGroupMembers.map(member => {
                                    if (member.isHousehold) {
                                        return (
                                            <div key={member.id} className="p-3 bg-gray-50 rounded-lg border border-gray-100">
                                                <div className="flex items-center gap-3 mb-2">
                                                    <div className="w-10 h-10 rounded-full bg-purple-100 flex items-center justify-center text-purple-600 font-bold border border-purple-200">
                                                        <Users className="w-5 h-5" />
                                                    </div>
                                                    <div>
                                                        <p className="font-medium text-gray-800">{member.name}</p>
                                                        <p className="text-xs text-gray-500">Household</p>
                                                    </div>
                                                    {viewingGroup?.admin_id === user.id && (
                                                        <button
                                                            onClick={() => {
                                                                setManageHouseholdId(member.id);
                                                                setManageHouseholdModalOpen(true);
                                                            }}
                                                            className="ml-auto text-xs bg-purple-100 text-purple-700 px-2 py-1 rounded-full font-medium hover:bg-purple-200 transition"
                                                        >
                                                            Manage
                                                        </button>
                                                    )}
                                                </div>
                                                <div className="pl-14 space-y-2">
                                                    {member.members.map(fam => (
                                                        <div key={fam.id} className="flex items-center gap-2 text-sm text-gray-600">
                                                            {fam.picture ? (
                                                                <img src={fam.picture} alt={fam.name} className="w-6 h-6 rounded-full object-cover" />
                                                            ) : (
                                                                <div className="w-6 h-6 rounded-full bg-gray-200 flex items-center justify-center text-xs font-medium">
                                                                    {fam.name.charAt(0)}
                                                                </div>
                                                            )}
                                                            <span>{fam.name}</span>
                                                            {viewingGroup?.admin_id === fam.id && (
                                                                <span className="text-[10px] bg-blue-100 text-blue-700 px-1.5 py-0.5 rounded-full font-medium">Admin</span>
                                                            )}
                                                        </div>
                                                    ))}
                                                </div>
                                            </div>
                                        );
                                    }
                                    return (
                                        <div key={member.id} className="flex items-center gap-3 p-2 hover:bg-gray-50 rounded-lg">
                                            {member.picture ? (
                                                <img src={member.picture} alt={member.name} className="w-10 h-10 rounded-full object-cover border border-gray-200" />
                                            ) : (
                                                <div className="w-10 h-10 rounded-full bg-orange-100 flex items-center justify-center text-orange-600 font-bold border border-orange-200">
                                                    {member.name.charAt(0)}
                                                </div>
                                            )}
                                            <div>
                                                <p className="font-medium text-gray-800">{member.name}</p>
                                                <p className="text-xs text-gray-500">{member.email}</p>
                                            </div>
                                            {viewingGroup?.admin_id === member.id && (
                                                <span className="ml-auto text-xs bg-blue-100 text-blue-700 px-2 py-1 rounded-full font-medium">Admin</span>
                                            )}
                                        </div>
                                    );
                                })
                            )}
                        </div>
                    </div>
                </div>
            )}

            <ManageHouseholdModal
                isOpen={manageHouseholdModalOpen}
                onClose={() => {
                    setManageHouseholdModalOpen(false);
                    setManageHouseholdId(null);
                }}
                user={user}
                refreshUser={refreshUser}
                householdId={manageHouseholdId}
                isAdmin={viewingGroup?.admin_id === user.id}
                groupId={viewingGroup?.id}
            />
        </div>
    );
};

export default Groups;
