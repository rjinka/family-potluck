import React from 'react';
import { X, CheckCircle, HelpCircle, XCircle, User } from 'lucide-react';

const EventRSVPModal = ({ isOpen, onClose, rsvps, groupMembers, hostId }) => {
    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4 animate-in fade-in duration-200">
            <div className="bg-white rounded-2xl shadow-xl w-full max-w-2xl overflow-hidden animate-in zoom-in-95 duration-200 flex flex-col max-h-[85vh]">
                <div className="bg-blue-600 p-6 text-white flex justify-between items-start shrink-0">
                    <div>
                        <h2 className="text-xl font-bold flex items-center gap-2">
                            <User className="w-6 h-6" />
                            Who's Coming
                        </h2>
                        <p className="text-blue-100 text-sm mt-1">
                            Guest list status
                        </p>
                    </div>
                    <button
                        onClick={onClose}
                        className="text-blue-200 hover:text-white transition p-1 rounded-full hover:bg-blue-500/20"
                    >
                        <X className="w-6 h-6" />
                    </button>
                </div>

                <div className="p-6 overflow-y-auto">
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                        {/* Going */}
                        <div className="bg-green-50 p-4 rounded-xl border border-green-100">
                            <div className="flex items-center gap-2 mb-3">
                                <CheckCircle className="w-5 h-5 text-green-600" />
                                <span className="font-bold text-green-800">
                                    Going ({rsvps.filter(r => r.status === 'Yes').length})
                                </span>
                            </div>
                            <div className="space-y-2">
                                {rsvps.filter(r => r.status === 'Yes').map(r => (
                                    <div key={r.id} className="text-sm text-green-700 flex justify-between items-center">
                                        <span className="font-medium">{r.family_name}</span>
                                        <span className="text-xs bg-green-100 px-2 py-0.5 rounded-full text-green-800">
                                            {r.count || 1} adults{r.kids_count > 0 ? `, ${r.kids_count} kids` : ''}
                                        </span>
                                    </div>
                                ))}
                                {rsvps.filter(r => r.status === 'Yes').length === 0 && (
                                    <p className="text-xs text-green-600 italic">No one yet</p>
                                )}
                            </div>
                        </div>

                        {/* Maybe */}
                        <div className="bg-yellow-50 p-4 rounded-xl border border-yellow-100">
                            <div className="flex items-center gap-2 mb-3">
                                <HelpCircle className="w-5 h-5 text-yellow-600" />
                                <span className="font-bold text-yellow-800">
                                    Maybe ({rsvps.filter(r => r.status === 'Maybe').length})
                                </span>
                            </div>
                            <div className="space-y-2">
                                {rsvps.filter(r => r.status === 'Maybe').map(r => (
                                    <div key={r.id} className="text-sm text-yellow-700 font-medium">
                                        {r.family_name}
                                    </div>
                                ))}
                                {rsvps.filter(r => r.status === 'Maybe').length === 0 && (
                                    <p className="text-xs text-yellow-600 italic">No one yet</p>
                                )}
                            </div>
                        </div>

                        {/* Can't Go */}
                        <div className="bg-red-50 p-4 rounded-xl border border-red-100">
                            <div className="flex items-center gap-2 mb-3">
                                <XCircle className="w-5 h-5 text-red-600" />
                                <span className="font-bold text-red-800">
                                    Can't Go ({rsvps.filter(r => r.status === 'No').length})
                                </span>
                            </div>
                            <div className="space-y-2">
                                {rsvps.filter(r => r.status === 'No').map(r => (
                                    <div key={r.id} className="text-sm text-red-700 font-medium">
                                        {r.family_name}
                                    </div>
                                ))}
                                {rsvps.filter(r => r.status === 'No').length === 0 && (
                                    <p className="text-xs text-red-600 italic">No one yet</p>
                                )}
                            </div>
                        </div>

                        {/* No Response */}
                        <div className="bg-gray-50 p-4 rounded-xl border border-gray-100">
                            <div className="flex items-center gap-2 mb-3">
                                <div className="w-5 h-5 rounded-full border-2 border-gray-400 flex items-center justify-center">
                                    <span className="text-[10px] font-bold text-gray-600">?</span>
                                </div>
                                <span className="font-bold text-gray-600">No Response</span>
                            </div>
                            <div className="space-y-2">
                                {Array.isArray(groupMembers.families) && groupMembers.families
                                    .filter(m => !rsvps.some(r => r.family_id === m.id) && m.id !== hostId)
                                    .map(m => (
                                        <div key={m.id} className="text-sm text-gray-500 font-medium">
                                            {m.name}
                                        </div>
                                    ))
                                }
                                {Array.isArray(groupMembers.families) && groupMembers.families.filter(m => !rsvps.some(r => r.family_id === m.id) && m.id !== hostId).length === 0 && (
                                    <p className="text-xs text-gray-400 italic">Everyone has responded!</p>
                                )}
                            </div>
                        </div>
                    </div>
                </div>

                <div className="p-4 border-t border-gray-100 bg-gray-50 text-center shrink-0">
                    <button
                        onClick={onClose}
                        className="text-gray-500 hover:text-gray-800 text-sm font-medium"
                    >
                        Close
                    </button>
                </div>
            </div>
        </div>
    );
};

export default EventRSVPModal;
