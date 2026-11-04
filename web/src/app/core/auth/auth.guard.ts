import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { catchError, map, of } from 'rxjs';

import { AuthService } from './auth.service';

/** Lets signed-in users through; sends everyone else to /login, remembering where they were going. */
export const authGuard: CanActivateFn = (_route, state) => {
  if (inject(AuthService).isAuthenticated()) {
    return true;
  }

  return inject(Router).createUrlTree(['/login'], { queryParams: { returnUrl: state.url } });
};

/** Restricts routes to users with the 'superadmin' role. */
export const superAdminGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);

  if (!auth.isAuthenticated()) {
    return router.createUrlTree(['/login']);
  }

  if (auth.isSuperAdmin()) {
    return true;
  }

  return auth.profile().pipe(
    map((p) => {
      if (p.role === 'superadmin') {
        return true;
      }

      return router.createUrlTree(['/']);
    }),
    catchError(() => of(router.createUrlTree(['/']))),
  );
};

/** Keeps signed-in users off the login page. */
export const guestGuard: CanActivateFn = () => {
  if (!inject(AuthService).isAuthenticated()) {
    return true;
  }

  return inject(Router).createUrlTree(['/']);
};

/**
 * Only same-app paths are accepted as a post-login destination, so a crafted
 * `?returnUrl=https://evil.example` link cannot redirect off-site.
 */
export function safeReturnUrl(url: string | null | undefined): string {
  return url && url.startsWith('/') && !url.startsWith('//') ? url : '/';
}
