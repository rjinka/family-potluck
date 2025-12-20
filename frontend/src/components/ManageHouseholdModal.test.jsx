import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import ManageHouseholdModal from './ManageHouseholdModal';
import { UIProvider } from '../context/UIContext';
import api from '../api/axios';
import { vi } from 'vitest';

// Mock api
vi.mock('../api/axios');

// Mock UI Context
const mockShowToast = vi.fn();
const mockConfirm = vi.fn();

vi.mock('../context/UIContext', () => ({
    useUI: () => ({
        showToast: mockShowToast,
        confirm: mockConfirm
    }),
    UIProvider: ({ children }) => <div>{children}</div>
}));

describe('ManageHouseholdModal', () => {
    const mockUser = {
        id: 'user1',
        name: 'Test User',
        household_id: 'household1'
    };
    const mockRefreshUser = vi.fn();
    const mockOnClose = vi.fn();

    beforeEach(() => {
        vi.clearAllMocks();
    });

    test('renders nothing when not open', () => {
        render(
            <ManageHouseholdModal
                isOpen={false}
                onClose={mockOnClose}
                user={mockUser}
                refreshUser={mockRefreshUser}
            />
        );
        expect(screen.queryByText('Manage Household')).not.toBeInTheDocument();
    });

    test('renders create household form when user has no household', async () => {
        const userNoHousehold = { ...mockUser, household_id: null };
        render(
            <ManageHouseholdModal
                isOpen={true}
                onClose={mockOnClose}
                user={userNoHousehold}
                refreshUser={mockRefreshUser}
            />
        );

        expect(screen.getByText('Create a Household')).toBeInTheDocument();
        expect(screen.getByPlaceholderText('e.g. The Smiths')).toBeInTheDocument();
    });

    test('renders household details when user has household', async () => {
        const mockHousehold = {
            id: 'household1',
            name: 'The Smiths',
            member_ids: ['user1', 'user2']
        };
        api.get.mockResolvedValueOnce({ data: mockHousehold });

        render(
            <ManageHouseholdModal
                isOpen={true}
                onClose={mockOnClose}
                user={mockUser}
                refreshUser={mockRefreshUser}
            />
        );

        await waitFor(() => {
            expect(screen.getByText('The Smiths')).toBeInTheDocument();
            expect(screen.getByText('2 members')).toBeInTheDocument();
        });
    });

    test('handles create household submission', async () => {
        const userNoHousehold = { ...mockUser, household_id: null };
        const newHousehold = { id: 'new1', name: 'New Family', member_ids: ['user1'] };
        api.post.mockResolvedValueOnce({ data: newHousehold });

        render(
            <ManageHouseholdModal
                isOpen={true}
                onClose={mockOnClose}
                user={userNoHousehold}
                refreshUser={mockRefreshUser}
            />
        );

        fireEvent.change(screen.getByPlaceholderText('e.g. The Smiths'), { target: { value: 'New Family' } });
        fireEvent.click(screen.getByText('Create Household'));

        await waitFor(() => {
            expect(api.post).toHaveBeenCalledWith('/households', {
                name: 'New Family',
                family_id: 'user1'
            });
            expect(mockRefreshUser).toHaveBeenCalled();
            expect(mockShowToast).toHaveBeenCalledWith('Household created successfully!');
        });
    });

    test('handles add member submission', async () => {
        const mockHousehold = {
            id: 'household1',
            name: 'The Smiths',
            member_ids: ['user1']
        };
        api.get.mockResolvedValueOnce({ data: mockHousehold }); // Initial fetch
        api.post.mockResolvedValueOnce({}); // Add member response
        api.get.mockResolvedValueOnce({ data: mockHousehold }); // Re-fetch after add

        render(
            <ManageHouseholdModal
                isOpen={true}
                onClose={mockOnClose}
                user={mockUser}
                refreshUser={mockRefreshUser}
            />
        );

        await waitFor(() => expect(screen.getByText('The Smiths')).toBeInTheDocument());

        fireEvent.change(screen.getByPlaceholderText('Enter email address'), { target: { value: 'test@example.com' } });
        fireEvent.click(screen.getByText('Add'));

        await waitFor(() => {
            expect(api.post).toHaveBeenCalledWith('/households/add-member', {
                household_id: 'household1',
                email: 'test@example.com'
            });
            expect(mockShowToast).toHaveBeenCalledWith('Member added successfully!');
        });
    });
});
