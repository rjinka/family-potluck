import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import Groups from './Groups';
import api from '../api/axios';
import { useAuth } from '../context/AuthContext';
import { useUI } from '../context/UIContext';
import { useNavigate } from 'react-router-dom';

// Mock dependencies
vi.mock('../api/axios');
vi.mock('../context/AuthContext');
vi.mock('../context/UIContext');
vi.mock('react-router-dom', () => ({
    useNavigate: vi.fn(),
}));

describe('Groups Page', () => {
    const mockUser = { id: 'user1', group_ids: ['g1'] };
    const mockRefreshUser = vi.fn();
    const mockShowToast = vi.fn();
    const mockConfirm = vi.fn();
    const mockNavigate = vi.fn();

    beforeEach(() => {
        vi.clearAllMocks();
        useAuth.mockReturnValue({ user: mockUser, refreshUser: mockRefreshUser });
        useUI.mockReturnValue({ showToast: mockShowToast, confirm: mockConfirm });
        useNavigate.mockReturnValue(mockNavigate);

        // Default API mocks
        api.get.mockResolvedValue({
            data: [
                { id: 'g1', name: 'Family Group', admin_id: 'user1', join_code: 'CODE123' },
                { id: 'g2', name: 'Other Group', admin_id: 'user2' } // User not in this one
            ]
        });
    });

    it('renders user groups correctly', async () => {
        render(<Groups />);

        await waitFor(() => {
            expect(screen.getByText('Family Group')).toBeInTheDocument();
        });

        // Should filter out groups user is not in (based on logic in component)
        expect(screen.queryByText('Other Group')).not.toBeInTheDocument();
    });

    it('creates a new group', async () => {
        api.post.mockResolvedValue({});

        render(<Groups />);

        const input = await screen.findByPlaceholderText(/Group Name/i);
        const createButton = screen.getByRole('button', { name: 'Create Group' });

        fireEvent.change(input, { target: { value: 'New Group' } });
        expect(input.value).toBe('New Group');

        const form = createButton.closest('form');
        fireEvent.submit(form);

        await waitFor(() => {
            expect(api.post).toHaveBeenCalledWith('/groups', {
                name: 'New Group',
                admin_id: 'user1'
            });
            expect(mockShowToast).toHaveBeenCalledWith('Group created successfully!');
            expect(mockRefreshUser).toHaveBeenCalled();
        });
    });

    it('deletes a group (admin)', async () => {
        // Mock confirm to immediately execute callback
        mockConfirm.mockImplementation(({ onConfirm }) => onConfirm());
        api.delete.mockResolvedValue({});

        render(<Groups />);

        await waitFor(() => {
            expect(screen.getByText('Family Group')).toBeInTheDocument();
        });

        const deleteButton = screen.getByTitle('Delete Group');
        fireEvent.click(deleteButton);

        await waitFor(() => {
            expect(mockConfirm).toHaveBeenCalled();
            expect(api.delete).toHaveBeenCalledWith('/groups/g1?admin_id=user1');
            expect(mockShowToast).toHaveBeenCalledWith('Group deleted successfully');
            expect(mockRefreshUser).toHaveBeenCalled();
        });
    });

    it('leaves a group (member)', async () => {
        // Setup user as non-admin for a group
        useAuth.mockReturnValue({
            user: { id: 'user2', group_ids: ['g2'] },
            refreshUser: mockRefreshUser
        });

        api.get.mockResolvedValue({
            data: [
                { id: 'g2', name: 'Other Group', admin_id: 'admin1' }
            ]
        });

        mockConfirm.mockImplementation(({ onConfirm }) => onConfirm());
        api.post.mockResolvedValue({});

        render(<Groups />);

        await waitFor(() => {
            expect(screen.getByText('Other Group')).toBeInTheDocument();
        });

        const leaveButton = screen.getByTitle('Leave Group');
        fireEvent.click(leaveButton);

        await waitFor(() => {
            expect(mockConfirm).toHaveBeenCalled();
            expect(api.post).toHaveBeenCalledWith('/groups/leave', {
                family_id: 'user2',
                group_id: 'g2'
            });
            expect(mockShowToast).toHaveBeenCalledWith('You have left Other Group');
            expect(mockRefreshUser).toHaveBeenCalled();
        });
    });
});
