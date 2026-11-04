import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  effect,
  inject,
  input,
  output,
  signal,
  untracked,
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { NzButtonModule } from 'ng-zorro-antd/button';
import { NzCardModule } from 'ng-zorro-antd/card';
import { NzEmptyModule } from 'ng-zorro-antd/empty';
import { NzGridModule } from 'ng-zorro-antd/grid';
import { NzIconModule } from 'ng-zorro-antd/icon';
import { NzMessageService } from 'ng-zorro-antd/message';
import { NzPaginationModule } from 'ng-zorro-antd/pagination';
import { NzPopconfirmModule } from 'ng-zorro-antd/popconfirm';
import { NzSpinModule } from 'ng-zorro-antd/spin';
import { finalize } from 'rxjs';

import { MediaService, UploadFile, fileIcon } from '../../core/media/media.service';
import { FileSizePipe } from '../file-size.pipe';

const DEFAULT_PAGE_SIZE = 24;

/**
 * Grid of Media Library files with pagination. Loads its own data, so it can be dropped into a page
 * or a modal; call reload() after something outside it changes the library (e.g. an upload).
 */
@Component({
  selector: 'app-media-gallery',
  imports: [
    FileSizePipe,
    NzButtonModule,
    NzCardModule,
    NzEmptyModule,
    NzGridModule,
    NzIconModule,
    NzPaginationModule,
    NzPopconfirmModule,
    NzSpinModule,
  ],
  templateUrl: './media-gallery.html',
  styleUrl: './media-gallery.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MediaGallery {
  private readonly media = inject(MediaService);
  private readonly message = inject(NzMessageService);
  private readonly destroyRef = inject(DestroyRef);

  /** Folder whose files are shown; null is the library root. Changing it reloads the gallery. */
  readonly folderId = input<string | null>(null);
  /** Show the delete button on each card. */
  readonly deletable = input(true);
  /** Show a "move" button on each card, emitting `move`. */
  readonly movable = input(false);
  /** Text when the folder has no files. */
  readonly emptyText = input('No hay archivos aquí.');
  /** Make cards clickable and emit `select`, e.g. inside a media picker. */
  readonly selectable = input(false);
  /** Highlights this file's card when selectable. */
  readonly selectedId = input<string | null>(null);
  readonly pageSize = input(DEFAULT_PAGE_SIZE);

  readonly select = output<UploadFile>();
  readonly move = output<UploadFile>();

  protected readonly files = signal<UploadFile[]>([]);
  protected readonly total = signal(0);
  protected readonly page = signal(1);
  protected readonly loading = signal(false);
  protected readonly error = signal(false);
  /** Ids being deleted, to show a spinner on their button. */
  protected readonly deleting = signal<ReadonlySet<string>>(new Set());

  protected readonly fileIcon = fileIcon;

  /** Increments per request, so a slow answer for a folder the user already left is ignored. */
  private requestSeq = 0;

  constructor() {
    // Load page 1 at start and whenever the folder changes.
    effect(() => {
      this.folderId();
      untracked(() => this.load(1));
    });
  }

  /** Reloads the current page, e.g. after an upload. */
  reload(): void {
    this.load(this.page());
  }

  protected load(page: number): void {
    const seq = ++this.requestSeq;

    this.page.set(page);
    this.loading.set(true);
    this.error.set(false);

    this.media
      .list(this.folderId(), page, this.pageSize())
      .pipe(
        finalize(() => seq === this.requestSeq && this.loading.set(false)),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe({
        next: ({ data, meta }) => {
          if (seq !== this.requestSeq) {
            return;
          }

          // Deleting the last file of the last page leaves it empty: step back a page.
          if (data.length === 0 && page > 1) {
            this.load(page - 1);

            return;
          }

          this.files.set(data);
          this.total.set(meta.pagination.total);
        },
        error: () => seq === this.requestSeq && this.error.set(true),
      });
  }

  /** Asks the parent to move a file; the card's own click must not also select it. */
  protected requestMove(file: UploadFile, event: Event): void {
    event.stopPropagation();
    this.move.emit(file);
  }

  /** Emits `select` for a card when the gallery is selectable. Space would otherwise scroll the page. */
  protected pick(file: UploadFile, event?: Event): void {
    if (!this.selectable()) {
      return;
    }

    event?.preventDefault();
    this.select.emit(file);
  }

  protected remove(file: UploadFile): void {
    this.deleting.update((ids) => new Set(ids).add(file.id));

    this.media
      .delete(file.id)
      .pipe(
        finalize(() =>
          this.deleting.update((ids) => {
            const next = new Set(ids);
            next.delete(file.id);

            return next;
          }),
        ),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe({
        next: () => {
          this.message.success(`«${file.name}» eliminado`);
          this.reload();
        },
        error: () => this.message.error(`No se pudo eliminar «${file.name}»`),
      });
  }
}
