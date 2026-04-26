import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';

import { withProviders } from '@/test/test-utils';

import { ShareVideoForm } from './ShareVideoForm';

describe('<ShareVideoForm />', () => {
  it('requires a URL', async () => {
    const onSubmit = vi.fn();
    render(withProviders(<ShareVideoForm onSubmit={onSubmit} />));

    await userEvent.click(screen.getByRole('button', { name: /share video/i }));
    await waitFor(() => {
      expect(screen.getByText(/youtube url is required/i)).toBeInTheDocument();
    });
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('rejects non-URL input', async () => {
    const onSubmit = vi.fn();
    render(withProviders(<ShareVideoForm onSubmit={onSubmit} />));

    await userEvent.type(screen.getByLabelText(/youtube url/i), 'not a url');
    await userEvent.click(screen.getByRole('button', { name: /share video/i }));

    await waitFor(() => {
      expect(screen.getByText(/enter a valid url/i)).toBeInTheDocument();
    });
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('submits valid URL', async () => {
    const onSubmit = vi.fn();
    render(withProviders(<ShareVideoForm onSubmit={onSubmit} />));

    const url = 'https://www.youtube.com/watch?v=dQw4w9WgXcQ';
    await userEvent.type(screen.getByLabelText(/youtube url/i), url);
    await userEvent.click(screen.getByRole('button', { name: /share video/i }));

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalled();
    });
    expect(onSubmit.mock.calls[0][0]).toMatchObject({ url });
  });
});
