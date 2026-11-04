const TOKEN_KEY = 'cms.jwt';

/** Reads the stored JWT. Storage can throw (private mode, blocked site data), which counts as no token. */
export function readToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY);
  } catch {
    return null;
  }
}

export function writeToken(token: string): void {
  try {
    localStorage.setItem(TOKEN_KEY, token);
  } catch {
    // The session still works for this tab; it just won't survive a reload.
  }
}

export function clearToken(): void {
  try {
    localStorage.removeItem(TOKEN_KEY);
  } catch {
    // Nothing stored, nothing to clear.
  }
}

/**
 * Whether a JWT is present and not expired, judged by its `exp` claim. This only decides whether to
 * show the admin or the login page: the signature is checked by the API on every request.
 */
export function isTokenUsable(token: string | null, now = Date.now()): boolean {
  if (!token) {
    return false;
  }

  const payload = token.split('.')[1];
  if (!payload) {
    return false;
  }

  try {
    const json = atob(payload.replace(/-/g, '+').replace(/_/g, '/'));
    const { exp } = JSON.parse(json) as { exp?: unknown };

    return typeof exp !== 'number' || exp * 1000 > now;
  } catch {
    return false;
  }
}
