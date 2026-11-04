import { HttpErrorResponse, HttpInterceptorFn, HttpStatusCode } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';

import { API_BASE } from '../api';
import { AuthService } from './auth.service';

/**
 * Adds `Authorization: Bearer <jwt>` to API requests. When the API rejects the token (expired,
 * revoked secret) the user is signed out and sent to the login page, which brings them back here.
 */
export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const token = auth.token();

  const isApi = req.url.startsWith(`${API_BASE}/`);
  const isAuthEndpoint = req.url.startsWith(`${API_BASE}/auth/`);

  if (!token || !isApi || isAuthEndpoint) {
    return next(req);
  }

  return next(req.clone({ setHeaders: { Authorization: `Bearer ${token}` } })).pipe(
    catchError((err: unknown) => {
      if (err instanceof HttpErrorResponse && err.status === HttpStatusCode.Unauthorized) {
        auth.logout(router.url);
      }

      return throwError(() => err);
    }),
  );
};
