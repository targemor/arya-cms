import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';
import { EyeInvisibleOutline, EyeOutline, LockOutline, MailOutline } from '@ant-design/icons-angular/icons';
import { provideNzIcons } from 'ng-zorro-antd/icon';

import { fakeJwt } from '../../../testing/fake-jwt';
import { Login } from './login';

async function render(returnUrl?: string) {
  localStorage.clear();

  await TestBed.configureTestingModule({
    imports: [Login],
    providers: [
      provideRouter([]),
      provideHttpClient(),
      provideHttpClientTesting(),
      provideNzIcons([EyeInvisibleOutline, EyeOutline, LockOutline, MailOutline]),
    ],
  }).compileComponents();

  const router = TestBed.inject(Router);
  if (returnUrl) {
    await router.navigateByUrl(`/?returnUrl=${encodeURIComponent(returnUrl)}`);
  }

  const fixture = TestBed.createComponent(Login);
  await fixture.whenStable();

  const el = fixture.nativeElement as HTMLElement;
  const type = (selector: string, value: string) => {
    const input = el.querySelector<HTMLInputElement>(selector)!;
    input.value = value;
    input.dispatchEvent(new Event('input'));
  };
  const submit = async () => {
    el.querySelector<HTMLButtonElement>('button[type=submit]')!.click();
    await fixture.whenStable();
  };

  return { fixture, el, router, type, submit, httpMock: TestBed.inject(HttpTestingController) };
}

afterEach(() => localStorage.clear());

describe('Login', () => {
  it('does not call the API while the form is invalid', async () => {
    const { el, type, submit, httpMock } = await render();

    type('#email', 'not-an-email');
    await submit();

    httpMock.expectNone('/v1/auth/login');
    expect(el.textContent).toContain('Introduce un email válido.');
    expect(el.textContent).toContain('Introduce tu contraseña.');
  });

  it('signs in and goes to the return url', async () => {
    const { fixture, router, type, submit, httpMock } = await render('/content-manager');
    const navigate = vi.spyOn(router, 'navigateByUrl').mockResolvedValue(true);

    type('#email', 'admin@example.com');
    type('#password', 'secret123');
    await submit();

    httpMock.expectOne('/v1/auth/login').flush({ token: fakeJwt({ sub: 'u1' }) });
    await fixture.whenStable();

    expect(navigate).toHaveBeenCalledWith('/content-manager');
  });

  it('shows an error for wrong credentials', async () => {
    const { fixture, el, type, submit, httpMock } = await render();

    type('#email', 'admin@example.com');
    type('#password', 'wrong');
    await submit();

    httpMock
      .expectOne('/v1/auth/login')
      .flush({ error: 'invalid credentials' }, { status: 401, statusText: 'Unauthorized' });
    await fixture.whenStable();

    expect(el.querySelector('nz-alert')?.textContent).toContain('Email o contraseña incorrectos.');
  });
});
