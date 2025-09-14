import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import Button from '../components/primitives/Button';

describe('Button', () => {
    it('renders label and handles loading', () => {
        const { rerender } = render(<Button>Click</Button>);
        expect(
            screen.getByRole('button', { name: 'Click' })
        ).toBeInTheDocument();
        rerender(<Button loading>Click</Button>);
        expect(
            screen.getByRole('button', { name: 'Loading…' })
        ).toBeInTheDocument();
    });
});
