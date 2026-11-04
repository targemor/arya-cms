import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, map } from 'rxjs';

import { API_BASE, Page } from '../api';

/** A Media Library file, as returned by the API. */
export interface UploadFile {
  id: string;
  name: string;
  alternativeText: string | null;
  hash: string;
  ext: string;
  mime: string;
  /** Bytes. */
  size: number;
  /** Public URL, e.g. /uploads/<hash>.png. */
  url: string;
  provider: string;
  /** Folder holding the file; null is the library root. */
  folderId: string | null;
  createdAt: string;
  updatedAt: string;
}

/** A Media Library folder. Folders nest through parentId; null is the library root. */
export interface Folder {
  id: string;
  name: string;
  parentId: string | null;
  createdAt: string;
  updatedAt: string;
}

export type UploadFilePage = Page<UploadFile>;

/**
 * The part of a file needed to show it: what a media field of an entry carries, nested by the API.
 * A full UploadFile also fits.
 */
export type MediaRef = Pick<UploadFile, 'id' | 'name' | 'url' | 'mime' | 'ext'> &
  Partial<Pick<UploadFile, 'alternativeText' | 'size'>>;

/** Endpoint nz-upload posts to; the multipart field is "file". */
export const UPLOAD_URL = `${API_BASE}/upload`;

/** Server-side limit (UPLOADS_MAX_SIZE_MB), so oversized files are refused before uploading. */
export const MAX_UPLOAD_SIZE_MB = 10;

@Injectable({ providedIn: 'root' })
export class MediaService {
  private readonly http = inject(HttpClient);

  /** One page of the files directly in a folder; null is the library root. */
  list(folderId: string | null, page: number, pageSize: number): Observable<UploadFilePage> {
    let params = new HttpParams()
      .set('pagination[page]', page)
      .set('pagination[pageSize]', pageSize);

    if (folderId) {
      params = params.set('folder', folderId);
    }

    return this.http.get<UploadFilePage>(`${API_BASE}/upload/files`, { params });
  }

  /** Moves a file into a folder; null is the library root. */
  moveFile(id: string, folderId: string | null): Observable<UploadFile> {
    return this.http.put<UploadFile>(`${API_BASE}/upload/files/${encodeURIComponent(id)}`, { folderId });
  }

  delete(id: string): Observable<void> {
    return this.http.delete<void>(`${API_BASE}/upload/files/${encodeURIComponent(id)}`);
  }

  /** Every folder, flat; parentId links them into a tree. */
  folders(): Observable<Folder[]> {
    return this.http.get<{ data: Folder[] }>(`${API_BASE}/upload/folders`).pipe(map((r) => r.data));
  }

  createFolder(name: string, parentId: string | null): Observable<Folder> {
    return this.http.post<Folder>(`${API_BASE}/upload/folders`, { name, parentId });
  }

  /** Renames and/or moves a folder: send its full new state. */
  updateFolder(id: string, name: string, parentId: string | null): Observable<Folder> {
    return this.http.put<Folder>(`${API_BASE}/upload/folders/${encodeURIComponent(id)}`, { name, parentId });
  }

  /** Only empty folders can be deleted; the API answers 409 otherwise. */
  deleteFolder(id: string): Observable<void> {
    return this.http.delete<void>(`${API_BASE}/upload/folders/${encodeURIComponent(id)}`);
  }
}

/** How a file is shown in the gallery: an image preview, or an icon for its kind. */
export function fileIcon(file: Pick<UploadFile, 'mime' | 'ext'>): string | null {
  if (file.mime.startsWith('image/')) {
    return null;
  }

  if (file.mime === 'application/pdf') {
    return 'file-pdf';
  }

  if (file.mime.startsWith('text/')) {
    return 'file-text';
  }

  if (/^\.(docx?|odt|rtf)$/.test(file.ext)) {
    return 'file-word';
  }

  if (/^\.(xlsx?|ods|csv)$/.test(file.ext)) {
    return 'file-excel';
  }

  if (/^\.(zip|gz|tar|7z|rar)$/.test(file.ext)) {
    return 'file-zip';
  }

  return 'file';
}
