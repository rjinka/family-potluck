import React, { useState, useEffect } from 'react';
import { X, Plus, Users, Save } from 'lucide-react';
import api from '../api/axios';
import { useUI } from '../context/UIContext';

const ManageHouseholdModal = ({ isOpen, onClose, user, refreshUser }) => {
    const { showToast } = useUI();
    const [household, setHousehold] = useState(null);
    const [loading, setLoading] = useState(false);
    const [createName, setCreateName] = useState('');
    const [addEmail, setAddEmail] = useState('');
    const [addingMember, setAddingMember] = useState(false);

    useEffect(() => {
        if (isOpen && user?.household_id) {
            fetchHousehold();
        } else {
            setHousehold(null);
        }
    }, [isOpen, user]);

    const fetchHousehold = async () => {
        setLoading(true);
        try {
            const response = await api.get(`/households/${user.household_id}`);
            // We also need members details. 
            // The backend GetHousehold returns { id, name, member_ids }.
            // We might need to fetch member details separately or update backend to return them.
            // For now, let's assume we just show the name and ID, and maybe we can't show member names yet unless we fetch them.
            // Actually, we can fetch families by IDs.
            // But let's stick to what we have.
            setHousehold(response.data);
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

    const handleAddMember = async (e) => {
        e.preventDefault();
        if (!addEmail) return;
        setAddingMember(true);
        try {
            await api.post('/households/add-member', {
                household_id: household.id || user.household_id,
                email: addEmail
            });
            setAddEmail('');
            showToast("Member added successfully!");
            // Refresh household to show new count or members if we were showing them
            fetchHousehold();
        } catch (error) {
            console.error("Failed to add member", error);
            showToast(error.response?.data || "Failed to add member", "error");
        } finally {
            setAddingMember(false);
        }
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
                    <div className="space-y-6">
                        <div className="bg-purple-50 p-4 rounded-lg border border-purple-100">
                            <div className="flex items-center gap-3">
                                <div className="bg-purple-100 p-2 rounded-full">
                                    <Users className="w-6 h-6 text-purple-600" />
                                </div>
                                <div>
                                    <h4 className="font-semibold text-gray-800">{household.name}</h4>
                                    <p className="text-sm text-gray-500">{household.member_ids?.length || 1} members</p>
                                </div>
                            </div>
                        </div>

                        <div>
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
                    </div>
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
                            <label className="block text-sm font-medium text-gray-700 mb-1">Household Name</label>
                            <input
                                type="text"
                                placeholder="e.g. The Smiths"
                                className="w-full p-3 border border-gray-200 rounded-lg mb-4 focus:ring-2 focus:ring-purple-500 outline-none"
                                value={createName}
                                onChange={(e) => setCreateName(e.target.value)}
                                required
                            />
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
        </div>
    );
};

export default ManageHouseholdModal;
