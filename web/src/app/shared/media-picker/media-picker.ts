import { ChangeDetectionStrategy, Component, computed, forwardRef, input, signal } from '@angular/core';
import { ControlValueAccessor, NG_VALUE_ACCESSOR } from '@angular/forms';
import { NzButtonModule } from 'ng-zorro-antd/button';
import { NzIconModule } from 'ng-zorro-antd/icon';
import { NzModalModule } from 'ng-zorro-antd/modal';

import { MediaRef, UploadFile, fileIcon } from '../../core/media/media.service';
import { MediaBrowser } from '../media-browser/media-browser';

/**
 * Form control for a media field. Its value is the id of a Media Library file (or null), which is
 * what the API expects when writing an entry; "Seleccionar archivo" opens the library in a modal.
 *
 * `file` is the file the current value refers to, as nested in the entry being edited, so the
 * preview can show it before anything is picked.
 */
@Component({
  selector: 'app-media-picker',
  imports: [MediaBrowser, NzButtonModule, NzIconModule, NzModalModule],
  templateUrl: './media-picker.html',
  styleUrl: './media-picker.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [{ provide: NG_VALUE_ACCESSOR, useExisting: forwardRef(() => MediaPicker), multi: true }],
})
export class MediaPicker implements ControlValueAccessor {
  readonly file = input<MediaRef | null>(null);
  /** Text of the open button, e.g. the field's label. */
  readonly label = input('Seleccionar archivo');

  protected readonly value = signal<string | null>(null);
  protected readonly picked = signal<MediaRef | null>(null);
  protected readonly open = signal(false);
  /** Folder the picker's library is showing; it remembers where the user was between openings. */
  protected readonly folderId = signal<string | null>(null);
  protected readonly disabled = signal(false);

  /** What to preview: the file picked here, else the entry's file if it is still the value. */
  protected readonly selected = computed<MediaRef | null>(() => {
    const id = this.value();
    if (!id) {
      return null;
    }

    const picked = this.picked();
    if (picked?.id === id) {
      return picked;
    }

    const initial = this.file();

    return initial?.id === id ? initial : null;
  });

  protected readonly fileIcon = fileIcon;

  private onChange: (value: string | null) => void = () => undefined;
  private onTouched: () => void = () => undefined;

  writeValue(value: string | null): void {
    this.value.set(value ?? null);
  }

  registerOnChange(fn: (value: string | null) => void): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  setDisabledState(disabled: boolean): void {
    this.disabled.set(disabled);
  }

  protected choose(file: UploadFile): void {
    this.picked.set(file);
    this.update(file.id);
    this.open.set(false);
  }

  protected clear(): void {
    this.picked.set(null);
    this.update(null);
  }

  protected close(): void {
    this.open.set(false);
    this.onTouched();
  }

  private update(id: string | null): void {
    this.value.set(id);
    this.onChange(id);
    this.onTouched();
  }
}
