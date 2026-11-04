/** Builds an unsigned JWT with the given claims, for tests: only the payload is ever read client-side. */
export function fakeJwt(claims: Record<string, unknown>): string {
  const b64url = (value: object) =>
    btoa(JSON.stringify(value)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');

  return `${b64url({ alg: 'HS256', typ: 'JWT' })}.${b64url(claims)}.signature`;
}
