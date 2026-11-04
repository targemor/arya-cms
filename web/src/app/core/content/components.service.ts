import { HttpClient } from '@angular/common/http';
import { Injectable, inject, signal } from '@angular/core';
import { Observable, tap } from 'rxjs';

import { API_BASE } from '../api';
import { Attribute } from './content-types.service';

export interface ComponentDefinition {
  name: string;
  category: string;
  displayName?: string;
  icon?: string;
  attributes: Attribute[];
}

const COMPONENTS_URL = `${API_BASE}/internal/components`;

@Injectable({ providedIn: 'root' })
export class ComponentsService {
  private readonly http = inject(HttpClient);

  private readonly state = signal<ComponentDefinition[]>([]);
  private readonly loadedState = signal(false);

  readonly components = this.state.asReadonly();
  readonly loaded = this.loadedState.asReadonly();

  refresh(): Observable<ComponentDefinition[]> {
    return this.http.get<ComponentDefinition[]>(COMPONENTS_URL).pipe(
      tap((comps) => {
        this.state.set(sortComponents(comps));
        this.loadedState.set(true);
      }),
    );
  }

  load(): void {
    this.refresh().subscribe({ error: () => undefined });
  }

  create(comp: ComponentDefinition): Observable<ComponentDefinition> {
    return this.http
      .post<ComponentDefinition>(COMPONENTS_URL, comp)
      .pipe(tap((created) => this.state.update((comps) => sortComponents([...comps, created]))));
  }

  update(uid: string, comp: Partial<ComponentDefinition>): Observable<ComponentDefinition> {
    return this.http
      .put<ComponentDefinition>(`${COMPONENTS_URL}/${encodeURIComponent(uid)}`, comp)
      .pipe(
        tap((updated) =>
          this.state.update((comps) =>
            sortComponents(comps.map((c) => (getComponentUID(c) === uid ? updated : c))),
          ),
        ),
      );
  }

  delete(uid: string): Observable<void> {
    return this.http
      .delete<void>(`${COMPONENTS_URL}/${encodeURIComponent(uid)}`)
      .pipe(tap(() => this.state.update((comps) => comps.filter((c) => getComponentUID(c) !== uid))));
  }
}

export function getComponentUID(c: ComponentDefinition): string {
  return c.category ? `${c.category}.${c.name}` : c.name;
}

function sortComponents(comps: ComponentDefinition[]): ComponentDefinition[] {
  return [...comps].sort((a, b) => (a.displayName || a.name).localeCompare(b.displayName || b.name));
}

