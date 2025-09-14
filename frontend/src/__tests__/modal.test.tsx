import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import Modal from '../components/primitives/Modal';

describe('Modal', () => {
    it('renders when open', () => {
        render(
            <Modal open onOpenChange={() => {}} title="Test">
                Hello
            </Modal>
        );
        expect(
            screen.getByRole('dialog', { name: 'Test' })
        ).toBeInTheDocument();
    });
});
