import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, TestRequest, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { DeleteOutline, FilePdfOutline } from '@ant-design/icons-angular/icons';
import { provideNzIcons } from 'ng-zorro-antd/icon';

import { UploadFile } from '../../core/media/media.service';
import { MediaGallery } from './media-gallery';

function file(id: string, overrides: Partial<UploadFile> = {}): UploadFile {
  return {
    id,
    name: `${id}.png`,
    alternativeText: null,
    hash: id,
    ext: '.png',
    mime: 'image/png',
    size: 2048,
    url: `/uploads/${id}.png`,
    provider: 'local',
    folderId: null,
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

function flushPage(req: TestRequest, files: UploadFile[], total = files.length): void {
  req.flush({ data: files, meta: { pagination: { page: 1, pageSize: 24, pageCount: 1, total } } });
}

const isList = (page: number) => (r: { url: string; params: { get(k: string): string | null } }) =>
  r.url === '/v1/upload/files' && r.params.get('pagination[page]') === String(page);

async function render() {
  await TestBed.configureTestingModule({
    imports: [MediaGallery],
    providers: [provideHttpClient(), provideHttpClientTesting(), provideNzIcons([DeleteOutline, FilePdfOutline])],
  }).compileComponents();

  const fixture = TestBed.createComponent(MediaGallery);
  fixture.detectChanges();

  return { fixture, el: fixture.nativeElement as HTMLElement, httpMock: TestBed.inject(HttpTestingController) };
}

describe('MediaGallery', () => {
  it('shows images as thumbnails and documents as icons, with name and size', async () => {
    const { fixture, el, httpMock } = await render();

    flushPage(httpMock.expectOne(isList(1)), [
      file('a', { alternativeText: 'A cover' }),
      file('b', { name: 'report.pdf', ext: '.pdf', mime: 'application/pdf', url: '/uploads/b.pdf' }),
    ]);
    await fixture.whenStable();

    const img = el.querySelector<HTMLImageElement>('img');
    expect(img?.getAttribute('src')).toBe('/uploads/a.png');
    expect(img?.alt).toBe('A cover');
    expect(el.querySelector('.anticon-file-pdf')).not.toBeNull();
    expect(el.textContent).toContain('report.pdf');
    expect(el.textContent).toContain('2 KB');
    expect(el.querySelectorAll('.media-delete').length).toBe(2);
  });

  it('shows an empty state', async () => {
    const { fixture, el, httpMock } = await render();

    flushPage(httpMock.expectOne(isList(1)), []);
    await fixture.whenStable();

    expect(el.textContent).toContain('No hay archivos aquí.');
  });

  it('hides delete buttons when not deletable', async () => {
    const { fixture, el, httpMock } = await render();
    fixture.componentRef.setInput('deletable', false);

    flushPage(httpMock.expectOne(isList(1)), [file('a')]);
    await fixture.whenStable();

    expect(el.querySelector('.media-delete')).toBeNull();
  });

  it('in select mode, emits the clicked file and highlights the selected one', async () => {
    const { fixture, el, httpMock } = await render();
    fixture.componentRef.setInput('selectable', true);
    fixture.componentRef.setInput('deletable', false);
    fixture.componentRef.setInput('selectedId', 'b');

    const picked: UploadFile[] = [];
    fixture.componentInstance.select.subscribe((f) => picked.push(f));

    flushPage(httpMock.expectOne(isList(1)), [file('a'), file('b')]);
    await fixture.whenStable();

    const cards = el.querySelectorAll<HTMLElement>('nz-card');
    expect(cards[1].classList).toContain('selected');
    expect(cards[0].getAttribute('role')).toBe('button');

    cards[0].click();
    cards[0].dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }));

    expect(picked.map((f) => f.id)).toEqual(['a', 'a']);
  });

  it('deleting the last file of a page reloads the previous page', async () => {
    const { fixture, httpMock } = await render();
    const gallery = fixture.componentInstance as unknown as {
      load(page: number): void;
      remove(f: UploadFile): void;
    };

    flushPage(httpMock.expectOne(isList(1)), [file('a')], 25);
    gallery.load(2);
    flushPage(httpMock.expectOne(isList(2)), [file('z')], 25);

    gallery.remove(file('z'));
    httpMock.expectOne('/v1/upload/files/z').flush(null);

    // Page 2 is now empty, so the gallery steps back to page 1.
    flushPage(httpMock.expectOne(isList(2)), [], 24);
    flushPage(httpMock.expectOne(isList(1)), [file('a')], 24);
    httpMock.verify();
  });
});
