import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, map } from 'rxjs';

import { API_BASE, Page } from '../api';
import { MediaRef } from '../media/media.service';

/** Value of an attribute in an entry as read from the API. Media fields come back as the nested file. */
export type FieldValue = string | number | boolean | MediaRef | Record<string, any> | Array<Record<string, any>> | null;

/** One entry of a content type: its attributes plus the system fields. */
export interface Entry {
  id: string;
  createdAt: string;
  updatedAt: string;
  [field: string]: FieldValue;
}

/** Value written for an attribute. Media fields are written as the file id. */
export type WriteValue = string | number | boolean | Record<string, any> | Array<Record<string, any>> | null;

export interface EntrySort {
  field: string;
  desc: boolean;
}

export interface EntryQuery {
  page: number;
  pageSize: number;
  sort: EntrySort[];
}

/** Builds the Strapi-style query string the API understands: pagination[page], sort=field:desc,... */
export function entryQueryParams(q: EntryQuery): HttpParams {
  let params = new HttpParams().set('pagination[page]', q.page).set('pagination[pageSize]', q.pageSize);

  if (q.sort.length > 0) {
    params = params.set('sort', q.sort.map((s) => `${s.field}:${s.desc ? 'desc' : 'asc'}`).join(','));
  }

  return params;
}

@Injectable({ providedIn: 'root' })
export class ContentService {
  private readonly http = inject(HttpClient);

  list(tableName: string, q: EntryQuery): Observable<Page<Entry>> {
    return this.http.get<Page<Entry>>(this.url(tableName), { params: entryQueryParams(q) });
  }

  getSingle(tableName: string): Observable<Entry | null> {
    return this.http.get<{ data: Entry | null }>(`${this.url(tableName)}/single`).pipe(map((r) => r.data));
  }

  create(tableName: string, data: Record<string, WriteValue>): Observable<Entry> {
    return this.http.post<{ data: Entry }>(this.url(tableName), { data }).pipe(map((r) => r.data));
  }

  update(tableName: string, id: string, data: Record<string, WriteValue>): Observable<Entry> {
    return this.http.put<{ data: Entry }>(this.url(tableName, id), { data }).pipe(map((r) => r.data));
  }

  updateSingle(tableName: string, data: Record<string, WriteValue>): Observable<Entry> {
    return this.http.put<{ data: Entry }>(`${this.url(tableName)}/single`, { data }).pipe(map((r) => r.data));
  }

  delete(tableName: string, id: string): Observable<void> {
    return this.http.delete<void>(this.url(tableName, id));
  }

  private url(tableName: string, id?: string): string {
    const base = `${API_BASE}/content/${encodeURIComponent(tableName)}`;

    return id ? `${base}/${encodeURIComponent(id)}` : base;
  }
}
