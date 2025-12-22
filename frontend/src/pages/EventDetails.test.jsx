import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, test, beforeEach, expect, vi } from 'vitest';
import EventDetails from './EventDetails';
import { AuthProvider } from '../context/AuthContext';
import { UIProvider } from '../context/UIContext';
import { WebSocketProvider } from '../context/WebSocketContext';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import api from '../api/axios';

// Mock dependencies
vi.mock('../api/axios');
vi.mock('../context/AuthContext', async () => {
    const actual = await vi.importActual('../context/AuthContext');
    return {
        ...actual,
        useAuth: () => ({
            user: { id: 'user1', household_id: 'household1' },
            login: vi.fn(),
            logout: vi.fn(),
        }),
    };
});

vi.mock('../context/WebSocketContext', () => ({
    WebSocketProvider: ({ children }) => <div>{children}</div>,
    useWebSocket: () => ({ lastMessage: null }),
}));

vi.mock('../context/UIContext', () => ({
    UIProvider: ({ children }) => <div>{children}</div>,
    useUI: () => ({
        showToast: vi.fn(),
        confirm: vi.fn(),
    }),
}));

const mockEvent = {
    id: 'event1',
    name: 'Test Event',
    date: '2025-12-25T18:00:00.000Z',
    location: 'Old Location',
    host_id: 'host1',
    group_id: 'group1',
};

const mockGroupMembers = {
    families: [
        { id: 'user1', name: 'User 1', household_id: 'household1' },
        { id: 'host1', name: 'Host 1', household_id: 'household2' },
    ],
    households: [
        { id: 'household1', name: 'User Household', address: 'User Address' },
        { id: 'household2', name: 'Host Household', address: 'Host Address' },
    ],
};

const mockSwapRequests = [
    {
        id: 'swap1',
        type: 'host',
        status: 'pending',
        requesting_family_id: 'host1',
        event_id: 'event1',
    },
];

describe('EventDetails Component', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        api.get.mockImplementation((url) => {
            if (url.includes('/events/event1')) return Promise.resolve({ data: mockEvent });
            if (url.includes('/dishes')) return Promise.resolve({ data: [] });
            if (url.includes('/rsvps')) return Promise.resolve({ data: [] });
            if (url.includes('/swaps')) return Promise.resolve({ data: mockSwapRequests });
            if (url.includes('/groups/members')) return Promise.resolve({ data: mockGroupMembers });
            if (url.includes('/events/stats')) return Promise.resolve({ data: null });
            return Promise.resolve({ data: {} });
        });
    });

    test('pre-fills location with user household address when accepting host swap', async () => {
        render(
            <MemoryRouter initialEntries={['/events/event1']}>
                <AuthProvider>
                    <UIProvider>
                        <WebSocketProvider>
                            <Routes>
                                <Route path="/events/:eventId" element={<EventDetails />} />
                            </Routes>
                        </WebSocketProvider>
                    </UIProvider>
                </AuthProvider>
            </MemoryRouter>
        );

        // Wait for data to load
        await waitFor(() => expect(screen.getByText('Test Event')).toBeInTheDocument());

        // Find the "Accept" button for the swap request
        // Since the swap request is from 'host1' and we are 'user1', it should be an incoming request (open offer)
        // The text might be "Accept"
        const acceptButtons = await screen.findAllByText('Volunteer to Host');
        fireEvent.click(acceptButtons[0]);

        // Check if the modal opened and location is pre-filled with "User Address"
        await waitFor(() => {
            const locationInput = screen.getByDisplayValue('User Address');
            expect(locationInput).toBeInTheDocument();
        });
    });
});
