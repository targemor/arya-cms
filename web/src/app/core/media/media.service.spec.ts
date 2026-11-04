import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { firstValueFrom } from 'rxjs';

import { MediaService, fileIcon } from './media.service';

describe('MediaService', () => {
  let service: MediaService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
    service = TestBed.inject(MediaService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => httpMock.verify());

  it('lists a page with Strapi-style pagination params', async () => {
    const result = firstValueFrom(service.list(null, 2, 24));

    const req = httpMock.expectOne((r) => r.url === '/v1/upload/files');
    expect(req.request.params.get('pagination[page]')).toBe('2');
    expect(req.request.params.get('pagination[pageSize]')).toBe('24');
    req.flush({ data: [], meta: { pagination: { page: 2, pageSize: 24, pageCount: 0, total: 0 } } });

    expect((await result).data).toEqual([]);
  });

  it('lists a folder with ?folder= (and the root without it)', () => {
    service.list('f1', 1, 24).subscribe();
    service.list(null, 1, 24).subscribe();

    const reqs = httpMock.match((r) => r.url === '/v1/upload/files');
    expect(reqs.map((r) => r.request.params.get('folder'))).toEqual(['f1', null]);
  });

  it('moves a file and manages folders', () => {
    service.moveFile('a', null).subscribe();
    service.createFolder('Fotos', 'p1').subscribe();
    service.updateFolder('f1', 'Fotos 2', null).subscribe();
    service.deleteFolder('f1').subscribe();

    const move = httpMock.expectOne('/v1/upload/files/a');
    expect([move.request.method, move.request.body]).toEqual(['PUT', { folderId: null }]);
    expect(httpMock.expectOne({ method: 'POST', url: '/v1/upload/folders' }).request.body).toEqual({ name: 'Fotos', parentId: 'p1' });
    expect(httpMock.expectOne({ method: 'PUT', url: '/v1/upload/folders/f1' }).request.body).toEqual({ name: 'Fotos 2', parentId: null });
    expect(httpMock.expectOne({ method: 'DELETE', url: '/v1/upload/folders/f1' })).toBeTruthy();
  });

  it('deletes by id', () => {
    service.delete('abc').subscribe();

    expect(httpMock.expectOne('/v1/upload/files/abc').request.method).toBe('DELETE');
  });
});

describe('fileIcon', () => {
  it.each([
    [{ mime: 'image/png', ext: '.png' }, null],
    [{ mime: 'image/svg+xml', ext: '.svg' }, null],
    [{ mime: 'application/pdf', ext: '.pdf' }, 'file-pdf'],
    [{ mime: 'text/plain', ext: '.txt' }, 'file-text'],
    [{ mime: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', ext: '.docx' }, 'file-word'],
    [{ mime: 'text/csv', ext: '.csv' }, 'file-text'],
    [{ mime: 'application/zip', ext: '.zip' }, 'file-zip'],
    [{ mime: 'application/octet-stream', ext: '' }, 'file'],
  ])('%o → %s', (file, icon) => {
    expect(fileIcon(file)).toBe(icon);
  });
});
