import { HttpClient } from '@angular/common/http';
import { Injectable, inject, signal } from '@angular/core';
import { Observable, tap } from 'rxjs';

import { API_BASE } from '../api';

export type AttributeType = 'string' | 'integer' | 'boolean' | 'media' | 'component';

export interface Attribute {
  name: string;
  displayName?: string;
  type: AttributeType;
  required: boolean;
  component?: string;
  repeatable?: boolean;
}

export type ContentTypeKind = 'collectionType' | 'singleType';

/** A content type definition, as the API's /internal/schemas endpoints take and return it. */
export interface ContentType {
  tableName: string;
  displayName?: string;
  kind?: ContentTypeKind;
  attributes: Attribute[];
}

const SCHEMAS_URL = `${API_BASE}/internal/schemas`;

/**
 * The defined content types, shared by the builder, the sider menu and the content manager. Creating
 * a type through here updates every view at once.
 */
@Injectable({ providedIn: 'root' })
export class ContentTypesService {
  private readonly http = inject(HttpClient);

  private readonly state = signal<ContentType[]>([]);
  private readonly loadedState = signal(false);

  /** Sorted by table name. */
  readonly types = this.state.asReadonly();
  /** False until the first successful load, so views can tell "loading" from "none defined". */
  readonly loaded = this.loadedState.asReadonly();

  refresh(): Observable<ContentType[]> {
    return this.http.get<ContentType[]>(SCHEMAS_URL).pipe(
      tap((types) => {
        this.state.set(sortByName(types));
        this.loadedState.set(true);
      }),
    );
  }

  /** Fire-and-forget refresh; a failure leaves the current list in place. */
  load(): void {
    this.refresh().subscribe({ error: () => undefined });
  }

  create(contentType: ContentType): Observable<ContentType> {
    return this.http
      .post<ContentType>(SCHEMAS_URL, contentType)
      .pipe(tap((created) => this.state.update((types) => sortByName([...types, created]))));
  }

  update(tableName: string, contentType: ContentType): Observable<ContentType> {
    return this.http
      .put<ContentType>(`${SCHEMAS_URL}/${encodeURIComponent(tableName)}`, contentType)
      .pipe(
        tap((updated) =>
          this.state.update((types) => sortByName(types.map((t) => (t.tableName === tableName ? updated : t)))),
        ),
      );
  }
}

function sortByName(types: ContentType[]): ContentType[] {
  return [...types].sort((a, b) => (a.displayName || a.tableName).localeCompare(b.displayName || b.tableName));
}
