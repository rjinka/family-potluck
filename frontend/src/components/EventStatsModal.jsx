import React from 'react';
import { BarChart2, X } from 'lucide-react';

const EventStatsModal = ({ isOpen, onClose, stats, eventType }) => {
    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4 animate-in fade-in duration-200">
            <div className="bg-white rounded-2xl shadow-xl w-full max-w-lg overflow-hidden animate-in zoom-in-95 duration-200">
                <div className="bg-orange-600 p-6 text-white flex justify-between items-start">
                    <div>
                        <h2 className="text-xl font-bold flex items-center gap-2">
                            <BarChart2 className="w-6 h-6" />
                            Event History
                        </h2>
                        <p className="text-orange-100 text-sm mt-1">
                            Stats for recurring {eventType}
                        </p>
                    </div>
                    <button
                        onClick={onClose}
                        className="text-orange-200 hover:text-white transition p-1 rounded-full hover:bg-orange-500/20"
                    >
                        <X className="w-6 h-6" />
                    </button>
                </div>

                <div className="p-6 max-h-[70vh] overflow-y-auto">
                    {stats && stats.total_occurrences > 0 ? (
                        <div className="space-y-8">
                            <div className="bg-orange-50 rounded-xl p-6 border border-orange-100 text-center">
                                <span className="text-sm text-orange-800 font-medium uppercase tracking-wide block mb-2">Total Occurrences</span>
                                <span className="text-4xl font-black text-orange-600 block">
                                    {stats.total_occurrences}
                                </span>
                                <p className="text-xs text-orange-600 mt-2">Times this event has happened</p>
                            </div>

                            <div>
                                <h3 className="font-bold text-gray-800 mb-4 flex items-center gap-2">
                                    <span className="bg-orange-100 text-orange-600 w-6 h-6 rounded-full flex items-center justify-center text-xs">#</span>
                                    Hosting Leaderboard
                                </h3>
                                <div className="space-y-3">
                                    {stats.host_counts.map((stat, index) => (
                                        <div key={stat.family_id} className="bg-white p-3 rounded-xl border border-gray-100 shadow-sm flex items-center justify-between hover:border-orange-200 transition">
                                            <div className="flex items-center gap-3">
                                                <div className={`w-8 h-8 rounded-full flex items-center justify-center font-bold text-sm ${index === 0 ? 'bg-yellow-100 text-yellow-700' :
                                                        index === 1 ? 'bg-gray-100 text-gray-700' :
                                                            index === 2 ? 'bg-orange-50 text-orange-700' : 'bg-gray-50 text-gray-500'
                                                    }`}>
                                                    {index + 1}
                                                </div>
                                                <div>
                                                    <p className="font-semibold text-gray-800 text-sm">{stat.family_name}</p>
                                                </div>
                                            </div>
                                            <div className="text-right">
                                                <span className="block font-bold text-orange-600">{stat.count}</span>
                                                <span className="text-[10px] text-gray-400 uppercase font-medium">Hosted</span>
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        </div>
                    ) : (
                        <div className="text-center py-12">
                            <BarChart2 className="w-12 h-12 text-gray-300 mx-auto mb-3" />
                            <p className="text-gray-500">No history available for this event yet.</p>
                            <p className="text-xs text-gray-400 mt-1">Stats will appear after the first event is completed.</p>
                        </div>
                    )}
                </div>

                <div className="p-4 border-t border-gray-100 bg-gray-50 text-center">
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

export default EventStatsModal;
