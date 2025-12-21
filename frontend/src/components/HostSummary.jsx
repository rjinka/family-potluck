import React from 'react';
import { Users, AlertTriangle, CheckCircle, Utensils } from 'lucide-react';

const HostSummary = ({ rsvps, dishes }) => {
    // Filter only "Yes" RSVPs
    const confirmedRSVPs = rsvps.filter(r => r.status === 'Yes');

    // Calculate Headcount
    const totalAdults = confirmedRSVPs.reduce((acc, curr) => acc + (curr.count || 0), 0);
    const totalKids = confirmedRSVPs.reduce((acc, curr) => acc + (curr.kids_count || 0), 0);
    const totalHeadcount = totalAdults + totalKids;

    // Analyze Dietary Needs
    const dietaryNeeds = {};
    confirmedRSVPs.forEach(rsvp => {
        if (rsvp.dietary_preferences && rsvp.dietary_preferences.length > 0) {
            rsvp.dietary_preferences.forEach(pref => {
                dietaryNeeds[pref] = (dietaryNeeds[pref] || 0) + (rsvp.count + rsvp.kids_count); // Assume all family members share the preference for simplicity, or just count families? "3 people are Vegetarian" is safer.
            });
        }
    });

    // Analyze Dish Coverage
    const dishCoverage = {};
    dishes.forEach(dish => {
        if (dish.dietary_tags && dish.dietary_tags.length > 0) {
            dish.dietary_tags.forEach(tag => {
                dishCoverage[tag] = (dishCoverage[tag] || 0) + 1;
            });
        }
    });

    // Generate Alerts
    const alerts = [];
    Object.keys(dietaryNeeds).forEach(pref => {
        const needs = dietaryNeeds[pref];
        const coverage = dishCoverage[pref] || 0;

        // Simple logic: If there are people with a need, there should be at least one dish.
        // Better logic: Maybe 1 dish per 5 people? Let's stick to "at least one" for now.
        if (coverage === 0) {
            alerts.push({
                type: 'critical',
                message: `${needs} guest${needs > 1 ? 's' : ''} need${needs === 1 ? 's' : ''} ${pref} food, but no dishes are tagged ${pref}!`
            });
        } else if (coverage < Math.ceil(needs / 5)) { // Arbitrary ratio
            alerts.push({
                type: 'warning',
                message: `${needs} guest${needs > 1 ? 's' : ''} need${needs === 1 ? 's' : ''} ${pref} food, but only ${coverage} dish${coverage > 1 ? 'es' : ''} available.`
            });
        }
    });

    if (totalHeadcount === 0) return null;

    return (
        <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 mb-8">
            <div className="flex items-center gap-3 mb-6">
                <div className="bg-purple-100 p-2 rounded-lg">
                    <Utensils className="w-6 h-6 text-purple-600" />
                </div>
                <h3 className="text-lg font-bold text-gray-800">Host Summary</h3>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
                <div className="bg-gray-50 p-4 rounded-lg border border-gray-100">
                    <div className="flex items-center gap-2 mb-1">
                        <Users className="w-4 h-4 text-gray-500" />
                        <span className="text-sm font-medium text-gray-500">Total Headcount</span>
                    </div>
                    <div className="text-2xl font-bold text-gray-800">{totalHeadcount}</div>
                    <div className="text-xs text-gray-500">{totalAdults} Adults, {totalKids} Kids</div>
                </div>

                <div className="bg-gray-50 p-4 rounded-lg border border-gray-100">
                    <div className="flex items-center gap-2 mb-1">
                        <Utensils className="w-4 h-4 text-gray-500" />
                        <span className="text-sm font-medium text-gray-500">Total Dishes</span>
                    </div>
                    <div className="text-2xl font-bold text-gray-800">{dishes.length}</div>
                    <div className="text-xs text-gray-500">{dishes.length > 0 ? (totalHeadcount / dishes.length).toFixed(1) : 0} people per dish</div>
                </div>

                <div className="bg-gray-50 p-4 rounded-lg border border-gray-100">
                    <div className="flex items-center gap-2 mb-1">
                        <CheckCircle className="w-4 h-4 text-gray-500" />
                        <span className="text-sm font-medium text-gray-500">Dietary Needs</span>
                    </div>
                    <div className="text-2xl font-bold text-gray-800">{Object.keys(dietaryNeeds).length}</div>
                    <div className="text-xs text-gray-500">Types of restrictions</div>
                </div>
            </div>

            {alerts.length > 0 ? (
                <div className="space-y-3">
                    <h4 className="font-semibold text-gray-700">Dietary Alerts</h4>
                    {alerts.map((alert, index) => (
                        <div key={index} className={`flex items-start gap-3 p-3 rounded-lg border ${alert.type === 'critical' ? 'bg-red-50 border-red-100 text-red-700' : 'bg-yellow-50 border-yellow-100 text-yellow-700'}`}>
                            <AlertTriangle className="w-5 h-5 shrink-0 mt-0.5" />
                            <p className="text-sm">{alert.message}</p>
                        </div>
                    ))}
                </div>
            ) : (
                <div className="flex items-center gap-2 text-green-600 bg-green-50 p-3 rounded-lg border border-green-100">
                    <CheckCircle className="w-5 h-5" />
                    <p className="text-sm font-medium">All dietary needs seem to be covered!</p>
                </div>
            )}
        </div>
    );
};

export default HostSummary;
