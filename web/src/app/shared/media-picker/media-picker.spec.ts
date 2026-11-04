import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { Component, signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { FormControl, ReactiveFormsModule } from '@angular/forms';
import { FileOutline, PictureOutline } from '@ant-design/icons-angular/icons';
import { provideNzIcons } from 'ng-zorro-antd/icon';

import { MediaRef, UploadFile } from '../../core/media/media.service';
import { MediaPicker } from './media-picker';

const cover: MediaRef = { id: 'f1', name: 'cover.png', url: '/uploads/f1.png', mime: 'image/png', ext: '.png' };

@Component({
  imports: [MediaPicker, ReactiveFormsModule],
  template: `<app-media-picker [formControl]="control" [file]="file()" />`,
})
class Host {
  readonly control = new FormControl<string | null>(null);
  readonly file = signal<MediaRef | null>(null);
}

async function render(initial: string | null, file: MediaRef | null) {
  await TestBed.configureTestingModule({
    imports: [Host],
    providers: [provideHttpClient(), provideHttpClientTesting(), provideNzIcons([FileOutline, PictureOutline])],
  }).compileComponents();

  const fixture = TestBed.createComponent(Host);
  fixture.componentInstance.control.setValue(initial);
  fixture.componentInstance.file.set(file);
  await fixture.whenStable();

  const picker = fixture.debugElement.children[0].componentInstance as unknown as {
    choose(f: UploadFile): void;
    clear(): void;
  };

  return { fixture, host: fixture.componentInstance, picker, el: fixture.nativeElement as HTMLElement };
}

describe('MediaPicker', () => {
  it('shows the entry file for the current value', async () => {
    const { el } = await render('f1', cover);

    expect(el.querySelector('img')?.getAttribute('src')).toBe('/uploads/f1.png');
    expect(el.textContent).toContain('cover.png');
    expect(el.textContent).toContain('Cambiar');
  });

  it('starts empty without a value', async () => {
    const { el } = await render(null, null);

    expect(el.textContent).toContain('Ningún archivo');
    expect(el.textContent).toContain('Seleccionar archivo');
  });

  it('stores the id of the picked file in the form control', async () => {
    const { fixture, host, picker, el } = await render(null, null);

    picker.choose({ ...cover, id: 'f2', name: 'other.png', url: '/uploads/f2.png' } as UploadFile);
    await fixture.whenStable();

    expect(host.control.value).toBe('f2');
    expect(host.control.dirty || host.control.touched).toBe(true);
    expect(el.textContent).toContain('other.png');
  });

  it('clears to null', async () => {
    const { fixture, host, picker, el } = await render('f1', cover);

    picker.clear();
    await fixture.whenStable();

    expect(host.control.value).toBeNull();
    expect(el.textContent).toContain('Ningún archivo');
  });
});
