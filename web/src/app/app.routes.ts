import { Routes } from '@angular/router';

import { authGuard, guestGuard, superAdminGuard } from './core/auth/auth.guard';

export const routes: Routes = [
  {
    path: 'login',
    canActivate: [guestGuard],
    title: 'Iniciar sesión · CMS',
    loadComponent: () => import('./pages/login/login').then((m) => m.Login),
  },
  {
    path: '',
    canActivate: [authGuard],
    // Re-checked on every child navigation, so an expired token is caught moving between pages.
    canActivateChild: [authGuard],
    loadComponent: () => import('./layout/admin-layout').then((m) => m.AdminLayout),
    children: [
      { path: '', pathMatch: 'full', redirectTo: 'content-type-builder' },
      {
        path: 'content-type-builder',
        title: 'Content-Type Builder · CMS',
        loadComponent: () =>
          import('./pages/content-type-builder/content-type-builder').then((m) => m.ContentTypeBuilder),
      },
      {
        path: 'media-library',
        title: 'Media Library · CMS',
        loadComponent: () => import('./pages/media-library/media-library').then((m) => m.MediaLibrary),
      },
      {
        // Without a table name the page asks which content type to open.
        path: 'content-manager',
        title: 'Content Manager · CMS',
        loadComponent: () => import('./pages/content-manager/content-manager').then((m) => m.ContentManager),
      },
      {
        path: 'content-manager/:tableName',
        title: 'Content Manager · CMS',
        loadComponent: () => import('./pages/content-manager/content-manager').then((m) => m.ContentManager),
      },
      {
        path: 'super-admin',
        canActivate: [superAdminGuard],
        title: 'Super Admin · CMS',
        loadComponent: () => import('./pages/super-admin/super-admin').then((m) => m.SuperAdmin),
      },
    ],
  },
  { path: '**', redirectTo: '' },
];
