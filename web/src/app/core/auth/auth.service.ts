import { HttpClient } from '@angular/common/http';
import { Injectable, computed, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { Observable, map, tap } from 'rxjs';

import { API_BASE } from '../api';
import { clearToken, isTokenUsable, readToken, writeToken } from './token';

interface LoginResponse {
  token: string;
}

export type UserRole = 'superadmin' | 'admin' | 'editor' | 'viewer';

/** Current user, as returned by GET /v1/user/profile. */
export interface Profile {
  id: string;
  username: string;
  email: string;
  role: UserRole;
  created_at?: string;
  updated_at?: string;
}

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly http = inject(HttpClient);
  private readonly router = inject(Router);

  private readonly tokenState = signal<string | null>(readToken());
  private readonly profileState = signal<Profile | null>(null);

  /** The JWT sent with API requests, or null when signed out. */
  readonly token = this.tokenState.asReadonly();
  readonly currentProfile = this.profileState.asReadonly();
  readonly isSuperAdmin = computed(() => this.profileState()?.role === 'superadmin');

  /** Checked on each call rather than cached, so an expired token is noticed on the next navigation. */
  isAuthenticated(): boolean {
    return isTokenUsable(this.tokenState());
  }

  login(email: string, password: string): Observable<void> {
    return this.http.post<LoginResponse>(`${API_BASE}/auth/login`, { email, password }).pipe(
      map(({ token }) => {
        writeToken(token);
        this.tokenState.set(token);
      }),
    );
  }

  profile(): Observable<Profile> {
    return this.http.get<Profile>(`${API_BASE}/user/profile`).pipe(
      tap((p) => this.profileState.set(p)),
    );
  }

  listUsers(): Observable<Profile[]> {
    return this.http.get<Profile[]>(`${API_BASE}/users`);
  }

  updateUserRole(userId: string, role: UserRole): Observable<{ status: string }> {
    return this.http.put<{ status: string }>(`${API_BASE}/users/${encodeURIComponent(userId)}/role`, { role });
  }

  /** Signs out and goes to the login page, which returns to `returnUrl` after signing back in. */
  logout(returnUrl?: string): void {
    clearToken();
    this.tokenState.set(null);
    this.profileState.set(null);

    void this.router.navigate(['/login'], {
      queryParams: returnUrl ? { returnUrl } : {},
    });
  }
}
