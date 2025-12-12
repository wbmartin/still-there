let csrfToken: string | null = null;

export const getCsrfToken = (): string | null => {
  return csrfToken;
};

export const setCsrfToken = (token: string): void => {
  csrfToken = token;
};

export const clearCsrfToken = (): void => {
  csrfToken = null;
};

const apiRequest = async (
  url: string,
  options: RequestInit = {},
  csrfToken?: string,
) => {
  const headers = new Headers(options.headers);
  const token = csrfToken || getCsrfToken();
  if (token) {
    headers.set('X-CSRF-Token', token);
  }
  return fetch(`/api${url}`, {
    ...options,
    headers,
    credentials: 'include',
  });
};

export { apiRequest };