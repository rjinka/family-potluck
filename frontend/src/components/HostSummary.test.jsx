import React from 'react';
import { render, screen } from '@testing-library/react';
import HostSummary from './HostSummary';

describe('HostSummary', () => {
    const mockRsvps = [
        { status: 'Yes', count: 2, kids_count: 1, dietary_preferences: ['Vegan'] },
        { status: 'Yes', count: 1, kids_count: 0, dietary_preferences: [] },
        { status: 'No', count: 1, kids_count: 0, dietary_preferences: [] }, // Should be ignored
    ];

    const mockDishes = [
        { name: 'Salad', dietary_tags: ['Vegan'] },
        { name: 'Chicken', dietary_tags: [] },
    ];

    it('renders headcount correctly', () => {
        render(<HostSummary rsvps={mockRsvps} dishes={mockDishes} />);

        // Total adults: 2 + 1 = 3
        // Total kids: 1 + 0 = 1
        // Total headcount: 4
        expect(screen.getByText('4')).toBeInTheDocument();
        expect(screen.getByText('3 Adults, 1 Kids')).toBeInTheDocument();
    });

    it('renders dish count correctly', () => {
        render(<HostSummary rsvps={mockRsvps} dishes={mockDishes} />);
        expect(screen.getByText('2')).toBeInTheDocument(); // 2 dishes
    });

    it('shows success message when dietary needs are covered', () => {
        render(<HostSummary rsvps={mockRsvps} dishes={mockDishes} />);
        expect(screen.getByText('All dietary needs seem to be covered!')).toBeInTheDocument();
    });

    it('shows alert when dietary needs are NOT covered', () => {
        const rsvpsWithGlutenFree = [
            ...mockRsvps,
            { status: 'Yes', count: 1, kids_count: 0, dietary_preferences: ['Gluten-Free'] }
        ];

        render(<HostSummary rsvps={rsvpsWithGlutenFree} dishes={mockDishes} />);

        // Should show alert for Gluten-Free
        expect(screen.getByText(/1 guest needs Gluten-Free food/i)).toBeInTheDocument();
    });

    it('does not render if headcount is 0', () => {
        render(<HostSummary rsvps={[]} dishes={[]} />);
        expect(screen.queryByText('Host Summary')).not.toBeInTheDocument();
    });
});
