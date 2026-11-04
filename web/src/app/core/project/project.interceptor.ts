import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';

import { API_BASE } from '../api';
import { ProjectsService } from './projects.service';

/**
 * Attaches the `X-Project: <projectSlug>` header to requests targeting the CMS API,
 * ensuring each request is scoped to the tenant project currently selected in the UI.
 */
export const projectInterceptor: HttpInterceptorFn = (req, next) => {
  const projectsService = inject(ProjectsService);
  const isApi = req.url.startsWith(`${API_BASE}/`);
  const isAuthEndpoint = req.url.startsWith(`${API_BASE}/auth/`);

  if (!isApi || isAuthEndpoint || req.headers.has('X-Project')) {
    return next(req);
  }

  const slug = projectsService.activeSlug();
  const modified = req.clone({
    setHeaders: {
      'X-Project': slug,
    },
  });

  return next(modified);
};

