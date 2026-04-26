import { useMutation } from '@tanstack/react-query';

import { signIn } from '../services';

export function useSignIn() {
  return useMutation({ mutationFn: signIn });
}
