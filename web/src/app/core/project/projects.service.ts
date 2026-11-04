import { HttpClient } from '@angular/common/http';
import { Injectable, computed, inject, signal } from '@angular/core';
import { Observable, tap } from 'rxjs';

import { API_BASE } from '../api';

export interface Project {
  id: string;
  name: string;
  slug: string;
  description?: string;
  imageUrl?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface ProjectMember {
  projectId: string;
  userId: string;
  username: string;
  email: string;
  role: 'admin' | 'editor' | 'viewer';
  createdAt: string;
  updatedAt: string;
}

export type APIKeyAction = 'find' | 'findOne' | 'create' | 'update' | 'delete';
export type APIKeyAccessType = 'custom' | 'read-only' | 'full-access';

export interface APIKey {
  id: string;
  projectId: string;
  projectSlug?: string;
  name: string;
  description?: string;
  keyPrefix: string;
  token?: string;
  accessType: APIKeyAccessType;
  permissions: Record<string, APIKeyAction[]>;
  lastUsedAt?: string;
  createdAt: string;
  updatedAt: string;
}

const PROJECTS_URL = `${API_BASE}/projects`;
const STORAGE_KEY = 'cms_active_project_slug';

@Injectable({ providedIn: 'root' })
export class ProjectsService {
  private readonly http = inject(HttpClient);

  private readonly state = signal<Project[]>([]);
  private readonly loadedState = signal(false);
  private readonly activeProjectState = signal<Project | null>(null);

  readonly projects = this.state.asReadonly();
  readonly loaded = this.loadedState.asReadonly();
  readonly activeProject = this.activeProjectState.asReadonly();
  readonly activeSlug = computed(() => this.activeProjectState()?.slug || 'default');

  load(): Observable<Project[]> {
    return this.http.get<Project[]>(PROJECTS_URL).pipe(
      tap((projs) => {
        this.state.set(projs);
        this.loadedState.set(true);

        const savedSlug = localStorage.getItem(STORAGE_KEY);
        const match = projs.find((p) => p.slug === savedSlug);
        if (match) {
          this.activeProjectState.set(match);
        } else if (projs.length > 0) {
          const defaultProj = projs.find((p) => p.slug === 'default') || projs[0];
          this.activeProjectState.set(defaultProj);
          localStorage.setItem(STORAGE_KEY, defaultProj.slug);
        }
      }),
    );
  }

  setActive(project: Project): void {
    this.activeProjectState.set(project);
    localStorage.setItem(STORAGE_KEY, project.slug);
  }

  create(data: { name: string; slug: string; description?: string; imageUrl?: string }): Observable<Project> {
    return this.http.post<Project>(PROJECTS_URL, data).pipe(
      tap((created) => {
        this.state.update((list) => [...list, created]);
        this.setActive(created);
      }),
    );
  }

  update(slug: string, data: { name: string; description?: string; imageUrl?: string }): Observable<Project> {
    return this.http.put<Project>(`${PROJECTS_URL}/${encodeURIComponent(slug)}`, data).pipe(
      tap((updated) => {
        this.state.update((list) => list.map((p) => (p.slug === slug ? updated : p)));
        if (this.activeProjectState()?.slug === slug) {
          this.activeProjectState.set(updated);
        }
      }),
    );
  }

  delete(slug: string): Observable<void> {
    return this.http.delete<void>(`${PROJECTS_URL}/${encodeURIComponent(slug)}`).pipe(
      tap(() => {
        this.state.update((list) => list.filter((p) => p.slug !== slug));
        if (this.activeProjectState()?.slug === slug) {
          const remaining = this.state();
          const next = remaining.find((p) => p.slug === 'default') || remaining[0] || null;
          if (next) {
            this.setActive(next);
          } else {
            this.activeProjectState.set(null);
            localStorage.removeItem(STORAGE_KEY);
          }
        }
      }),
    );
  }

  getMembers(slug: string): Observable<ProjectMember[]> {
    return this.http.get<ProjectMember[]>(`${PROJECTS_URL}/${encodeURIComponent(slug)}/members`);
  }

  addOrUpdateMember(slug: string, data: { userId: string; role: string }): Observable<void> {
    return this.http.post<void>(`${PROJECTS_URL}/${encodeURIComponent(slug)}/members`, data);
  }

  removeMember(slug: string, userId: string): Observable<void> {
    return this.http.delete<void>(
      `${PROJECTS_URL}/${encodeURIComponent(slug)}/members/${encodeURIComponent(userId)}`,
    );
  }

  getAPIKeys(slug: string): Observable<APIKey[]> {
    return this.http.get<APIKey[]>(`${PROJECTS_URL}/${encodeURIComponent(slug)}/api-keys`);
  }

  createAPIKey(
    slug: string,
    data: {
      name: string;
      description?: string;
      accessType: APIKeyAccessType;
      permissions: Record<string, APIKeyAction[]>;
    },
  ): Observable<APIKey> {
    return this.http.post<APIKey>(`${PROJECTS_URL}/${encodeURIComponent(slug)}/api-keys`, data);
  }

  updateAPIKey(
    slug: string,
    id: string,
    data: {
      name: string;
      description?: string;
      accessType: APIKeyAccessType;
      permissions: Record<string, APIKeyAction[]>;
    },
  ): Observable<APIKey> {
    return this.http.put<APIKey>(
      `${PROJECTS_URL}/${encodeURIComponent(slug)}/api-keys/${encodeURIComponent(id)}`,
      data,
    );
  }

  deleteAPIKey(slug: string, id: string): Observable<void> {
    return this.http.delete<void>(
      `${PROJECTS_URL}/${encodeURIComponent(slug)}/api-keys/${encodeURIComponent(id)}`,
    );
  }
}

