import React, { useState } from 'react';
import { X, Save, CheckCircle } from 'lucide-react';
import api from '../api/axios';

const DIETARY_TAGS = ["Vegan", "Vegetarian", "Gluten-Free", "Dairy-Free", "Nut-Free", "Spicy", "Halal", "Kosher"];

const DietaryPreferencesModal = ({ isOpen, onClose, user, onUpdate }) => {
    const [preferences, setPreferences] = useState(user.dietary_preferences || []);
    const [saving, setSaving] = useState(false);

    if (!isOpen) return null;

    const togglePreference = (tag) => {
        if (preferences.includes(tag)) {
            setPreferences(preferences.filter(p => p !== tag));
        } else {
            setPreferences([...preferences, tag]);
        }
    };

    const handleSave = async () => {
        setSaving(true);
        try {
            await api.patch(`/families/${user.id}`, { dietary_preferences: preferences });
            onUpdate(preferences); // Notify parent to update local user state or refresh
            onClose();
        } catch (error) {
            console.error("Failed to update preferences", error);
        } finally {
            setSaving(false);
        }
    };

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
            <div className="bg-white rounded-2xl w-full max-w-md overflow-hidden shadow-xl animate-in fade-in zoom-in duration-200">
                <div className="p-6 border-b border-gray-100 flex justify-between items-center bg-gray-50">
                    <h3 className="text-xl font-bold text-gray-800">My Dietary Preferences</h3>
                    <button onClick={onClose} className="text-gray-400 hover:text-gray-600 transition">
                        <X className="w-6 h-6" />
                    </button>
                </div>

                <div className="p-6">
                    <p className="text-gray-600 mb-4 text-sm">
                        Select any dietary restrictions or preferences for your family. This helps hosts plan the menu.
                    </p>

                    <div className="flex flex-wrap gap-2 mb-6">
                        {DIETARY_TAGS.map(tag => (
                            <button
                                key={tag}
                                onClick={() => togglePreference(tag)}
                                className={`px-4 py-2 rounded-full text-sm font-medium border transition flex items-center gap-2 ${preferences.includes(tag)
                                        ? 'bg-green-100 text-green-700 border-green-200 ring-2 ring-green-500 ring-offset-1'
                                        : 'bg-white text-gray-600 border-gray-200 hover:border-gray-300 hover:bg-gray-50'
                                    }`}
                            >
                                {preferences.includes(tag) && <CheckCircle className="w-4 h-4" />}
                                {tag}
                            </button>
                        ))}
                    </div>

                    <div className="flex gap-3">
                        <button
                            onClick={onClose}
                            className="flex-1 py-3 text-gray-600 font-medium hover:bg-gray-50 rounded-xl transition"
                        >
                            Cancel
                        </button>
                        <button
                            onClick={handleSave}
                            disabled={saving}
                            className="flex-1 py-3 bg-green-600 text-white font-bold rounded-xl hover:bg-green-700 transition shadow-lg shadow-green-200 disabled:opacity-70 flex items-center justify-center gap-2"
                        >
                            {saving ? 'Saving...' : (
                                <>
                                    <Save className="w-5 h-5" />
                                    Save Preferences
                                </>
                            )}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default DietaryPreferencesModal;
