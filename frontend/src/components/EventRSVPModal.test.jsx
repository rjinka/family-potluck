import React from 'react';
import { render, screen } from '@testing-library/react';
import EventRSVPModal from './EventRSVPModal';

describe('EventRSVPModal', () => {
    const mockRsvps = [
        { id: '1', status: 'Yes', family_name: 'Smith', count: 2, kids_count: 1 },
        { id: '2', status: 'Maybe', family_name: 'Jones' },
        { id: '3', status: 'No', family_name: 'Doe' },
    ];

    it('renders correctly when open', () => {
        render(
            <EventRSVPModal
                isOpen={true}
                onClose={() => { }}
                rsvps={mockRsvps}
                groupMembers={[]}
                hostId="host123"
            />
        );

        expect(screen.getByText("Who's Coming")).toBeInTheDocument();
        expect(screen.getByText('Smith')).toBeInTheDocument();
        expect(screen.getByText('Jones')).toBeInTheDocument();
        expect(screen.getByText('Doe')).toBeInTheDocument();
    });

    it('displays correct counts', () => {
        render(
            <EventRSVPModal
                isOpen={true}
                onClose={() => { }}
                rsvps={mockRsvps}
                groupMembers={[]}
                hostId="host123"
            />
        );

        expect(screen.getByText('Going (1)')).toBeInTheDocument();
        expect(screen.getByText('Maybe (1)')).toBeInTheDocument();
        expect(screen.getByText("Can't Go (1)")).toBeInTheDocument();
    });

    it('displays "No one yet" for empty categories', () => {
        const emptyRsvps = [];
        render(
            <EventRSVPModal
                isOpen={true}
                onClose={() => { }}
                rsvps={emptyRsvps}
                groupMembers={[]}
                hostId="host123"
            />
        );

        expect(screen.getAllByText('No one yet')).toHaveLength(3); // Yes, Maybe, No sections
    });
});
