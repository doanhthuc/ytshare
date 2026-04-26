export type SessionUser = {
  id: string;
  email: string;
  name: string;
};

export type AuthSession = {
  user: SessionUser;
  accessToken: string;
  refreshToken: string;
  expiresAt: string;
};

export type SignInPayload = {
  email: string;
  password: string;
};

export type SignUpPayload = {
  email: string;
  name: string;
  password: string;
};
