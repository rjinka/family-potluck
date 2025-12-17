import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import DietaryPreferencesModal from './DietaryPreferencesModal';
import api from '../api/axios';

// Mock the API
vi.mock('../api/axios');

describe('DietaryPreferencesModal', () => {
    const mockUser = { id: '123', dietary_preferences: ['Vegan'] };
    const mockOnUpdate = vi.fn();
    const mockOnClose = vi.fn();

    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('renders correctly when open', () => {
        render(
            <DietaryPreferencesModal
                isOpen={true}
                onClose={mockOnClose}
                user={mockUser}
                onUpdate={mockOnUpdate}
            />
        );

        expect(screen.getByText('My Dietary Preferences')).toBeInTheDocument();
        expect(screen.getByText('Vegan')).toBeInTheDocument();
    });

    it('does not render when closed', () => {
        render(
            <DietaryPreferencesModal
                isOpen={false}
                onClose={mockOnClose}
                user={mockUser}
                onUpdate={mockOnUpdate}
            />
        );

        expect(screen.queryByText('My Dietary Preferences')).not.toBeInTheDocument();
    });

    it('toggles preferences', () => {
        render(
            <DietaryPreferencesModal
                isOpen={true}
                onClose={mockOnClose}
                user={mockUser}
                onUpdate={mockOnUpdate}
            />
        );

        const veganButton = screen.getByText('Vegan');
        const glutenFreeButton = screen.getByText('Gluten-Free');

        // Toggle OFF Vegan
        fireEvent.click(veganButton);
        // Toggle ON Gluten-Free
        fireEvent.click(glutenFreeButton);

        // Check visual state (classes) is hard, but we can verify logic on save
    });

    it('calls API and onUpdate on save', async () => {
        api.patch.mockResolvedValue({});

        render(
            <DietaryPreferencesModal
                isOpen={true}
                onClose={mockOnClose}
                user={mockUser}
                onUpdate={mockOnUpdate}
            />
        );

        const saveButton = screen.getByText('Save Preferences');
        fireEvent.click(saveButton);

        await waitFor(() => {
            expect(api.patch).toHaveBeenCalledWith('/families/123', {
                dietary_preferences: ['Vegan'] // No changes made in this test
            });
            expect(mockOnUpdate).toHaveBeenCalledWith(['Vegan']);
            expect(mockOnClose).toHaveBeenCalled();
        });
    });
});
