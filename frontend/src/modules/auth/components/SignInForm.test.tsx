import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRouter,
} from '@tanstack/react-router';
import { describe, expect, it, vi } from 'vitest';

import { withProviders } from '@/test/test-utils';

import { SignInForm } from './SignInForm';

// We render the form inside a tiny in-memory router so the <Link to="/signup">
// inside the component resolves correctly.
function renderForm(onSubmit: (v: { email: string; password: string }) => void) {
  const rootRoute = createRootRoute({ component: () => <SignInForm onSubmit={onSubmit} /> });
  const router = createRouter({
    routeTree: rootRoute,
    history: createMemoryHistory({ initialEntries: ['/'] }),
  });
  return render(withProviders(<RouterProvider router={router} />));
}

describe('<SignInForm />', () => {
  it('shows validation errors for empty submission', async () => {
    const onSubmit = vi.fn();
    renderForm(onSubmit);

    // Wait for the router's initial transition to mount the form.
    // findBy* polls until the element appears, unlike getBy* which
    // throws on the first synchronous miss.
    const submitButton = await screen.findByRole('button', { name: /sign in/i });
    await userEvent.click(submitButton);

    await waitFor(() => {
      expect(screen.getByText(/email is required/i)).toBeInTheDocument();
      expect(screen.getByText(/password is required/i)).toBeInTheDocument();
    });
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('rejects invalid email format', async () => {
    const onSubmit = vi.fn();
    renderForm(onSubmit);

    const emailInput = await screen.findByLabelText(/email/i);
    await userEvent.type(emailInput, 'not-an-email');
    await userEvent.type(screen.getByLabelText(/password/i), 'password123');
    await userEvent.click(screen.getByRole('button', { name: /sign in/i }));

    await waitFor(() => {
      expect(screen.getByText(/enter a valid email/i)).toBeInTheDocument();
    });
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('submits valid credentials', async () => {
    const onSubmit = vi.fn();
    renderForm(onSubmit);

    const emailInput = await screen.findByLabelText(/email/i);
    await userEvent.type(emailInput, 'user@example.com');
    await userEvent.type(screen.getByLabelText(/password/i), 'password123');
    await userEvent.click(screen.getByRole('button', { name: /sign in/i }));

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith(
        { email: 'user@example.com', password: 'password123' },
        expect.anything()
      );
    });
  });
});
