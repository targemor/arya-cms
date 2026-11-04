import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { ActivatedRouteSnapshot, Router, RouterStateSnapshot, UrlTree, provideRouter } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { authGuard, guestGuard, safeReturnUrl } from './auth.guard';
import { authInterceptor } from './auth.interceptor';
import { AuthService } from './auth.service';
import { fakeJwt } from '../../../testing/fake-jwt';

const validToken = fakeJwt({ sub: 'user-1', exp: Math.floor(Date.now() / 1000) + 3600 });
const expiredToken = fakeJwt({ sub: 'user-1', exp: Math.floor(Date.now() / 1000) - 60 });

let httpMock: HttpTestingController | undefined;

function setup(storedToken: string | null = null) {
  TestBed.resetTestingModule();
  localStorage.clear();
  if (storedToken) {
    localStorage.setItem('cms.jwt', storedToken);
  }

  TestBed.configureTestingModule({
    providers: [
      provideRouter([]),
      provideHttpClient(withInterceptors([authInterceptor])),
      provideHttpClientTesting(),
    ],
  });

  httpMock = TestBed.inject(HttpTestingController);

  return {
    auth: TestBed.inject(AuthService),
    http: TestBed.inject(HttpClient),
    httpMock,
    router: TestBed.inject(Router),
  };
}

function runGuard(guard: typeof authGuard, url = '/content-manager') {
  return TestBed.runInInjectionContext(() =>
    guard({} as ActivatedRouteSnapshot, { url } as RouterStateSnapshot),
  );
}

afterEach(() => {
  httpMock?.verify();
  httpMock = undefined;
  localStorage.clear();
});

describe('AuthService', () => {
  it('stores the token returned by POST /v1/auth/login', async () => {
    const { auth, httpMock } = setup();

    const done = firstValueFrom(auth.login('admin@example.com', 'secret123'));
    const req = httpMock.expectOne('/v1/auth/login');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ email: 'admin@example.com', password: 'secret123' });
    req.flush({ token: validToken });
    await done;

    expect(auth.token()).toBe(validToken);
    expect(localStorage.getItem('cms.jwt')).toBe(validToken);
    expect(auth.isAuthenticated()).toBe(true);
  });

  it('restores a stored token and rejects an expired one', () => {
    expect(setup(validToken).auth.isAuthenticated()).toBe(true);
    expect(setup(expiredToken).auth.isAuthenticated()).toBe(false);
  });

  it('logout clears the token and goes to /login with the return url', () => {
    const { auth, router } = setup(validToken);
    const navigate = vi.spyOn(router, 'navigate').mockResolvedValue(true);

    auth.logout('/content-manager');

    expect(auth.token()).toBeNull();
    expect(localStorage.getItem('cms.jwt')).toBeNull();
    expect(navigate).toHaveBeenCalledWith(['/login'], { queryParams: { returnUrl: '/content-manager' } });
  });
});

describe('authInterceptor', () => {
  it('adds the bearer token to API requests', () => {
    const { http, httpMock } = setup(validToken);

    http.get('/v1/user/profile').subscribe();

    expect(httpMock.expectOne('/v1/user/profile').request.headers.get('Authorization')).toBe(
      `Bearer ${validToken}`,
    );
  });

  it('does not send the token to the login endpoint or to other origins', () => {
    const { http, httpMock } = setup(validToken);

    http.post('/v1/auth/login', {}).subscribe();
    http.get('https://example.com/v1/x').subscribe();

    expect(httpMock.expectOne('/v1/auth/login').request.headers.has('Authorization')).toBe(false);
    expect(httpMock.expectOne('https://example.com/v1/x').request.headers.has('Authorization')).toBe(false);
  });

  it('signs out when the API answers 401', () => {
    const { auth, http, httpMock } = setup(validToken);
    const logout = vi.spyOn(auth, 'logout').mockImplementation(() => undefined);

    http.get('/v1/internal/schemas').subscribe({ error: () => undefined });
    httpMock.expectOne('/v1/internal/schemas').flush({ error: 'invalid or expired token' }, { status: 401, statusText: 'Unauthorized' });

    expect(logout).toHaveBeenCalledOnce();
  });

  it('leaves other errors to the caller', () => {
    const { auth, http, httpMock } = setup(validToken);
    const logout = vi.spyOn(auth, 'logout');

    http.get('/v1/internal/schemas').subscribe({ error: () => undefined });
    httpMock.expectOne('/v1/internal/schemas').flush({ error: 'boom' }, { status: 500, statusText: 'Server Error' });

    expect(logout).not.toHaveBeenCalled();
  });
});

describe('guards', () => {
  it('authGuard lets a signed-in user through', () => {
    setup(validToken);

    expect(runGuard(authGuard)).toBe(true);
  });

  it('authGuard sends a signed-out user to /login with the return url', () => {
    const { router } = setup();

    const result = runGuard(authGuard, '/content-manager') as UrlTree;

    expect(router.serializeUrl(result)).toBe('/login?returnUrl=%2Fcontent-manager');
  });

  it('guestGuard keeps a signed-in user off /login', () => {
    const { router } = setup(validToken);

    expect(router.serializeUrl(runGuard(guestGuard) as UrlTree)).toBe('/');
  });

  it.each([
    ['/content-manager', '/content-manager'],
    [null, '/'],
    ['https://evil.example', '/'],
    ['//evil.example', '/'],
  ])('safeReturnUrl(%s) is %s', (input, expected) => {
    expect(safeReturnUrl(input)).toBe(expected);
  });
});
