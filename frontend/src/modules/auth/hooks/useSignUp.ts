import { useMutation } from '@tanstack/react-query';

import { signUp } from '../services';

export function useSignUp() {
  return useMutation({ mutationFn: signUp });
}
