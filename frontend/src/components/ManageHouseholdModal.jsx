import React, { useState, useEffect } from 'react';
import { X, Plus, Users, Save, Trash2 } from 'lucide-react';
import api from '../api/axios';
import { useUI } from '../context/UIContext';

const ManageHouseholdModal = ({ isOpen, onClose, user, refreshUser, householdId, isAdmin, groupId }) => {
    const { showToast, confirm } = useUI();
    const [household, setHousehold] = useState(null);
    const [loading, setLoading] = useState(false);
    const [createName, setCreateName] = useState('');
    const [createAddress, setCreateAddress] = useState('');
    const [addEmail, setAddEmail] = useState('');
    const [addingMember, setAddingMember] = useState(false);
    const [members, setMembers] = useState([]);
    const [updateName, setUpdateName] = useState('');
    const [updateAddress, setUpdateAddress] = useState('');
    const [updating, setUpdating] = useState(false);

    const targetHouseholdId = householdId || user?.household_id;

    useEffect(() => {
        if (isOpen && targetHouseholdId) {
            fetchHousehold();
        } else {
            setHousehold(null);
            setMembers([]);
            setCreateName('');
            setCreateAddress('');
        }
    }, [isOpen, targetHouseholdId]);

    const fetchHousehold = async () => {
        setLoading(true);
        try {
            const response = await api.get(`/households/${targetHouseholdId}`);
            setHousehold(response.data);
            setUpdateName(response.data.name);
            setUpdateAddress(response.data.address || '');

            // Use members details from response if available
            if (response.data.members) {
                setMembers(response.data.members);
            } else if (response.data.member_ids && response.data.member_ids.length > 0 && groupId) {
                // Fallback to group members if backend didn't provide them
                const membersResponse = await api.get(`/groups/members?group_id=${groupId}`);
                const allFamilies = membersResponse.data.families || [];
                const householdMembers = allFamilies.filter(f => response.data.member_ids.includes(f.id));
                setMembers(householdMembers);
            }
        } catch (error) {
            console.error("Failed to fetch household", error);
            showToast("Failed to fetch household", "error");
        } finally {
            setLoading(false);
        }
    };

    const handleCreateHousehold = async (e) => {
        e.preventDefault();
        if (!createName) return;
        try {
            const response = await api.post('/households', {
                name: createName,
                address: createAddress,
                family_id: user.id
            });
            await refreshUser();
            setHousehold(response.data);
            showToast("Household created successfully!");
        } catch (error) {
            console.error("Failed to create household", error);
            showToast("Failed to create household", "error");
        }
    };

    const handleUpdateHousehold = async (e) => {
        e.preventDefault();
        if (!updateName) return;
        setUpdating(true);
        try {
            await api.patch(`/households/${targetHouseholdId}`, {
                name: updateName,
                address: updateAddress
            });
            showToast("Household updated successfully!");
            fetchHousehold();
        } catch (error) {
            console.error("Failed to update household", error);
            showToast("Failed to update household", "error");
        } finally {
            setUpdating(false);
        }
    };

    const handleAddMember = async (e) => {
        e.preventDefault();
        if (!addEmail) return;
        setAddingMember(true);
        try {
            await api.post('/households/add-member', {
                household_id: targetHouseholdId,
                email: addEmail
            });
            setAddEmail('');
            showToast("Member added successfully!");
            fetchHousehold();
        } catch (error) {
            console.error("Failed to add member", error);
            showToast(error.response?.data || "Failed to add member", "error");
        } finally {
            setAddingMember(false);
        }
    };

    const handleRemoveMember = async (memberId) => {
        confirm({
            title: "Remove Member",
            message: "Are you sure you want to remove this member from the household?",
            confirmText: "Remove",
            isDestructive: true,
            onConfirm: async () => {
                try {
                    await api.post('/households/remove-member', {
                        household_id: targetHouseholdId,
                        family_id: memberId,
                        admin_id: isAdmin ? user.id : undefined,
                        group_id: isAdmin ? groupId : undefined
                    });
                    showToast("Member removed successfully");
                    fetchHousehold();
                    if (memberId === user.id) {
                        refreshUser();
                        onClose();
                    }
                } catch (error) {
                    console.error("Failed to remove member", error);
                    showToast("Failed to remove member", "error");
                }
            }
        });
    };

    const handleDeleteHousehold = async () => {
        confirm({
            title: "Delete Household",
            message: "Are you sure you want to delete this household? This will disband all members.",
            confirmText: "Delete Household",
            isDestructive: true,
            onConfirm: async () => {
                try {
                    let url = `/households/${targetHouseholdId}`;
                    if (isAdmin && groupId) {
                        url += `?admin_id=${user.id}&group_id=${groupId}`;
                    }
                    await api.delete(url);
                    showToast("Household deleted successfully");
                    await refreshUser();
                    onClose();
                } catch (error) {
                    console.error("Failed to delete household", error);
                    showToast("Failed to delete household", "error");
                }
            }
        });
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50 animate-in fade-in duration-200">
            <div className="bg-white rounded-xl shadow-xl max-w-md w-full p-6 transform transition-all scale-100">
                <div className="flex justify-between items-center mb-6">
                    <h3 className="text-xl font-bold text-gray-800">Manage Household</h3>
                    <button onClick={onClose} className="text-gray-400 hover:text-gray-600 transition">
                        <X className="w-6 h-6" />
                    </button>
                </div>

                {loading ? (
                    <div className="text-center py-8 text-gray-500">Loading...</div>
                ) : household ? (
                    <>
                        <div className="bg-purple-50 p-4 rounded-lg border border-purple-100">
                            <div className="flex items-start justify-between gap-3 mb-4">
                                <div className="flex items-center gap-3">
                                    <div className="bg-purple-100 p-2 rounded-full">
                                        <Users className="w-6 h-6 text-purple-600" />
                                    </div>
                                    <div>
                                        <h4 className="font-semibold text-gray-800">{household.name}</h4>
                                        <p className="text-sm text-gray-500">{household.member_ids?.length || 1} members</p>
                                        {household.address && (
                                            <p className="text-xs text-gray-500 mt-1">{household.address}</p>
                                        )}
                                    </div>
                                </div>
                                {(isAdmin || user.household_id === household.id) && (
                                    <button
                                        onClick={handleDeleteHousehold}
                                        className="text-red-500 hover:text-red-700 p-2"
                                        title="Delete Household"
                                    >
                                        <Trash2 className="w-5 h-5" />
                                    </button>
                                )}
                            </div>

                            {(isAdmin || user.household_id === household.id) && (
                                <form onSubmit={handleUpdateHousehold} className="space-y-3 border-t border-purple-200 pt-3">
                                    <div>
                                        <label className="block text-xs font-medium text-gray-700 mb-1">Update Name</label>
                                        <input
                                            type="text"
                                            value={updateName}
                                            onChange={(e) => setUpdateName(e.target.value)}
                                            className="w-full p-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-purple-500 outline-none"
                                            placeholder="Household Name"
                                        />
                                    </div>
                                    <div>
                                        <label className="block text-xs font-medium text-gray-700 mb-1">Update Address</label>
                                        <input
                                            type="text"
                                            value={updateAddress}
                                            onChange={(e) => setUpdateAddress(e.target.value)}
                                            className="w-full p-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-purple-500 outline-none"
                                            placeholder="Household Address"
                                        />
                                    </div>
                                    <div className="flex justify-end">
                                        <button
                                            type="submit"
                                            disabled={updating}
                                            className="bg-purple-600 text-white px-3 py-1.5 rounded-lg text-xs font-medium hover:bg-purple-700 transition disabled:opacity-50 flex items-center gap-1"
                                        >
                                            <Save className="w-3 h-3" />
                                            {updating ? 'Saving...' : 'Save Changes'}
                                        </button>
                                    </div>
                                </form>
                            )}
                        </div>

                        <div>
                            <h4 className="text-sm font-semibold text-gray-700 mb-3">Members</h4>
                            <div className="space-y-2 mb-4">
                                {members.length > 0 ? (
                                    members.map(member => (
                                        <div key={member.id} className="flex items-center justify-between p-2 bg-gray-50 rounded-lg">
                                            <div className="flex items-center gap-2">
                                                <div className="w-8 h-8 rounded-full bg-purple-200 flex items-center justify-center text-purple-700 font-bold text-xs">
                                                    {member.name.charAt(0)}
                                                </div>
                                                <span className="text-sm font-medium text-gray-700">{member.name}</span>
                                            </div>
                                            {(isAdmin || user.household_id === household.id) && (
                                                <button
                                                    onClick={() => handleRemoveMember(member.id)}
                                                    className="text-gray-400 hover:text-red-500 transition"
                                                    title="Remove Member"
                                                >
                                                    <X className="w-4 h-4" />
                                                </button>
                                            )}
                                        </div>
                                    ))
                                ) : (
                                    <p className="text-sm text-gray-500 italic">Loading members details...</p>
                                )}
                            </div>

                            <h4 className="text-sm font-semibold text-gray-700 mb-3">Add Family Member</h4>
                            <form onSubmit={handleAddMember} className="flex gap-2">
                                <input
                                    type="email"
                                    placeholder="Enter email address"
                                    className="flex-1 p-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-purple-500 outline-none text-sm"
                                    value={addEmail}
                                    onChange={(e) => setAddEmail(e.target.value)}
                                    required
                                />
                                <button
                                    type="submit"
                                    disabled={addingMember}
                                    className="bg-purple-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-purple-700 transition disabled:opacity-50"
                                >
                                    {addingMember ? 'Adding...' : 'Add'}
                                </button>
                            </form>
                            <p className="text-xs text-gray-500 mt-2">
                                Enter the email of the family member you want to add. They must already have an account.
                            </p>
                        </div>
                    </>
                ) : (
                    <div>
                        <div className="text-center mb-6">
                            <div className="bg-gray-100 w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-4">
                                <Users className="w-8 h-8 text-gray-400" />
                            </div>
                            <h4 className="text-lg font-semibold text-gray-800">Create a Household</h4>
                            <p className="text-gray-500 text-sm mt-1">
                                Bundle multiple family members together so you can host events as a single unit (e.g., "The Smiths").
                            </p>
                        </div>

                        <form onSubmit={handleCreateHousehold}>
                            <div className="mb-4">
                                <label className="block text-sm font-medium text-gray-700 mb-1">Household Name</label>
                                <input
                                    type="text"
                                    placeholder="e.g. The Smiths"
                                    className="w-full p-3 border border-gray-200 rounded-lg focus:ring-2 focus:ring-purple-500 outline-none"
                                    value={createName}
                                    onChange={(e) => setCreateName(e.target.value)}
                                    required
                                />
                            </div>
                            <div className="mb-4">
                                <label className="block text-sm font-medium text-gray-700 mb-1">Address (Optional)</label>
                                <input
                                    type="text"
                                    placeholder="e.g. 123 Main St"
                                    className="w-full p-3 border border-gray-200 rounded-lg focus:ring-2 focus:ring-purple-500 outline-none"
                                    value={createAddress}
                                    onChange={(e) => setCreateAddress(e.target.value)}
                                />
                                <p className="text-xs text-gray-500 mt-1">
                                    If set, this address will be automatically used for events hosted by this household.
                                </p>
                            </div>
                            <button
                                type="submit"
                                className="w-full bg-purple-600 text-white py-3 rounded-lg font-medium hover:bg-purple-700 transition"
                            >
                                Create Household
                            </button>
                        </form>
                    </div>
                )}
            </div>
        </div >
    );
};

export default ManageHouseholdModal;
