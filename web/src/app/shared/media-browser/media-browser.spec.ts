import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import {
  DeleteOutline,
  EditOutline,
  FolderAddOutline,
  FolderFill,
  FolderOutline,
  HomeOutline,
  MoreOutline,
} from '@ant-design/icons-angular/icons';
import { provideNzIcons } from 'ng-zorro-antd/icon';

import { Folder } from '../../core/media/media.service';
import { MediaBrowser } from './media-browser';

function folder(id: string, name: string, parentId: string | null = null): Folder {
  return { id, name, parentId, createdAt: '', updatedAt: '' };
}

const folders = [folder('photos', 'Fotos'), folder('covers', 'Portadas', 'photos'), folder('docs', 'Documentos')];
const emptyPage = { data: [], meta: { pagination: { page: 1, pageSize: 24, pageCount: 0, total: 0 } } };

async function render(mode: 'manage' | 'select', folderId: string | null = null) {
  await TestBed.configureTestingModule({
    imports: [MediaBrowser],
    providers: [
      provideHttpClient(),
      provideHttpClientTesting(),
      provideNzIcons([DeleteOutline, EditOutline, FolderAddOutline, FolderFill, FolderOutline, HomeOutline, MoreOutline]),
    ],
  }).compileComponents();

  const fixture = TestBed.createComponent(MediaBrowser);
  fixture.componentRef.setInput('mode', mode);
  fixture.componentRef.setInput('folderId', folderId);
  fixture.detectChanges();

  const httpMock = TestBed.inject(HttpTestingController);
  httpMock.expectOne('/v1/upload/folders').flush({ data: folders });
  for (const req of httpMock.match((r) => r.url === '/v1/upload/files')) {
    req.flush(emptyPage);
  }
  await fixture.whenStable();

  return { fixture, el: fixture.nativeElement as HTMLElement, httpMock };
}

const names = (el: HTMLElement, selector: string) =>
  [...el.querySelectorAll(selector)].map((e) => e.textContent!.replace(/\s+/g, ' ').trim());

describe('MediaBrowser', () => {
  it('shows the root folders and an empty-root message', async () => {
    const { el } = await render('manage');

    expect(names(el, '.folder-name')).toEqual(['Documentos', 'Fotos']);
    expect(names(el, 'nz-breadcrumb-item a, nz-breadcrumb-item .crumb-current')).toEqual(['Biblioteca']);
    expect(el.textContent).toContain('Nueva carpeta');
  });

  it('opens a folder: breadcrumbs, subfolders and its files', async () => {
    const { fixture, el, httpMock } = await render('manage');

    el.querySelector<HTMLElement>('[aria-label="Abrir carpeta Fotos"]')!.click();
    fixture.detectChanges();

    const files = httpMock.expectOne((r) => r.url === '/v1/upload/files' && r.params.get('folder') === 'photos');
    files.flush(emptyPage);
    await fixture.whenStable();

    expect(fixture.componentInstance.folderId()).toBe('photos');
    expect(names(el, 'nz-breadcrumb-item a, nz-breadcrumb-item .crumb-current')).toEqual(['Biblioteca', 'Fotos']);
    expect(names(el, '.folder-name')).toEqual(['Portadas']);
    expect(el.textContent).toContain('Esta carpeta no tiene archivos.');
  });

  it('creates a folder inside the open one', async () => {
    const { fixture, httpMock } = await render('manage', 'photos');
    const browser = fixture.componentInstance as unknown as {
      openCreate(): void;
      nameControl: { setValue(v: string): void };
      saveName(): void;
    };

    browser.openCreate();
    browser.nameControl.setValue('  Nuevas  ');
    browser.saveName();

    const req = httpMock.expectOne({ method: 'POST', url: '/v1/upload/folders' });
    expect(req.request.body).toEqual({ name: 'Nuevas', parentId: 'photos' });
  });

  it('rejects a name with a slash before calling the API', async () => {
    const { fixture, httpMock } = await render('manage');
    const browser = fixture.componentInstance as unknown as {
      openCreate(): void;
      nameControl: { setValue(v: string): void; hasError(e: string): boolean };
      saveName(): void;
    };

    browser.openCreate();
    browser.nameControl.setValue('a/b');
    browser.saveName();

    httpMock.expectNone({ method: 'POST', url: '/v1/upload/folders' });
    expect(browser.nameControl.hasError('slash')).toBe(true);
  });

  it('falls back to the root when the open folder no longer exists', async () => {
    const { fixture } = await render('select', 'deleted-folder');

    expect(fixture.componentInstance.folderId()).toBeNull();
  });

  it('has no management actions in select mode', async () => {
    const { el } = await render('select');

    expect(el.textContent).not.toContain('Nueva carpeta');
    expect(el.querySelector('.folder-menu')).toBeNull();
  });
});
