import React from 'react';
import { render, screen } from '@testing-library/react';
import EventStatsModal from './EventStatsModal';

describe('EventStatsModal', () => {
    const mockStats = {
        total_occurrences: 5,
        host_counts: [
            { family_id: '1', family_name: 'Smith', count: 3 },
            { family_id: '2', family_name: 'Jones', count: 2 },
        ]
    };

    it('renders stats correctly', () => {
        render(
            <EventStatsModal
                isOpen={true}
                onClose={() => { }}
                stats={mockStats}
                eventType="Potluck"
            />
        );

        expect(screen.getByText('Event History')).toBeInTheDocument();
        expect(screen.getByText('5')).toBeInTheDocument(); // Total occurrences
        expect(screen.getByText('Smith')).toBeInTheDocument();
        expect(screen.getByText('3')).toBeInTheDocument(); // Smith count
    });

    it('renders empty state when no stats', () => {
        render(
            <EventStatsModal
                isOpen={true}
                onClose={() => { }}
                stats={{ total_occurrences: 0, host_counts: [] }}
                eventType="Potluck"
            />
        );

        expect(screen.getByText('No history available for this event yet.')).toBeInTheDocument();
    });
});
