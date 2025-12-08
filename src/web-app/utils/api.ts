const apiRequest = async (url: string, options: RequestInit = {}, csrfToken?: string) => {
  const headers = new Headers(options.headers);
  if (csrfToken) {
    headers.set('X-CSRF-Token', csrfToken);
  }
  return fetch(`/api${url}`, {
    ...options,
    headers,
    credentials: 'include',
  });
};

export { apiRequest };