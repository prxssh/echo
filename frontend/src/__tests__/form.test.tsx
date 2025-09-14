import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import Form, { FormField } from '../components/patterns/Form';

describe('Form', () => {
    it('renders field with label and helper', () => {
        render(
            <Form>
                <FormField
                    name="q"
                    label="Query"
                    helperText="Type to search"
                    placeholder="Search"
                />
            </Form>
        );
        expect(screen.getByLabelText('Query')).toBeInTheDocument();
        expect(screen.getByText('Type to search')).toBeInTheDocument();
    });
});
