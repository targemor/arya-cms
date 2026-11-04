import { fakeJwt } from '../../../testing/fake-jwt';
import { isTokenUsable } from './token';

describe('isTokenUsable', () => {
  const now = Date.UTC(2026, 0, 1);
  const inSeconds = (s: number) => Math.floor(now / 1000) + s;

  it('accepts a token that has not expired', () => {
    expect(isTokenUsable(fakeJwt({ sub: 'u1', exp: inSeconds(60) }), now)).toBe(true);
  });

  it('rejects an expired token', () => {
    expect(isTokenUsable(fakeJwt({ sub: 'u1', exp: inSeconds(-1) }), now)).toBe(false);
  });

  it('accepts a token without exp', () => {
    expect(isTokenUsable(fakeJwt({ sub: 'u1' }), now)).toBe(true);
  });

  it.each([null, '', 'not-a-jwt', 'a.!!!.c'])('rejects %s', (token) => {
    expect(isTokenUsable(token, now)).toBe(false);
  });
});
