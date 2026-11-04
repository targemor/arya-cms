import { HttpErrorResponse, HttpStatusCode } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  computed,
  effect,
  inject,
  input,
  model,
  output,
  signal,
  viewChild,
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { AbstractControl, FormControl, FormGroup, ReactiveFormsModule, ValidationErrors, Validators } from '@angular/forms';
import { NzBreadCrumbModule } from 'ng-zorro-antd/breadcrumb';
import { NzButtonModule } from 'ng-zorro-antd/button';
import { NzDropdownModule } from 'ng-zorro-antd/dropdown';
import { NzFormModule } from 'ng-zorro-antd/form';
import { NzGridModule } from 'ng-zorro-antd/grid';
import { NzIconModule } from 'ng-zorro-antd/icon';
import { NzInputModule } from 'ng-zorro-antd/input';
import { NzMenuModule } from 'ng-zorro-antd/menu';
import { NzMessageService } from 'ng-zorro-antd/message';
import { NzModalModule, NzModalService } from 'ng-zorro-antd/modal';
import { NzTreeSelectModule } from 'ng-zorro-antd/tree-select';
import { Observable, finalize } from 'rxjs';

import { ROOT_KEY, ROOT_NAME, childrenOf, destinationTree, keyToFolderId, pathTo } from '../../core/media/folders';
import { Folder, MediaService, UploadFile } from '../../core/media/media.service';
import { showFormErrors } from '../forms';
import { MediaGallery } from '../media-gallery/media-gallery';

/** Same rules the API applies to folder names (it also trims them). */
function folderName(control: AbstractControl<string>): ValidationErrors | null {
  const value = control.value.trim();

  if (value === '') {
    return { required: true };
  }

  return /[/\\]/.test(value) ? { slash: true } : null;
}

type NameDialog = { mode: 'create' } | { mode: 'rename'; folder: Folder };
type MoveDialog = { kind: 'file'; file: UploadFile } | { kind: 'folder'; folder: Folder };

/**
 * Browses the Media Library by folder: breadcrumbs, the open folder's subfolders, and its files.
 * In "manage" mode it also creates, renames, moves and deletes folders and moves and deletes files;
 * in "select" mode clicking a file emits `select` (the media picker).
 */
@Component({
  selector: 'app-media-browser',
  imports: [
    MediaGallery,
    ReactiveFormsModule,
    NzBreadCrumbModule,
    NzButtonModule,
    NzDropdownModule,
    NzFormModule,
    NzGridModule,
    NzIconModule,
    NzInputModule,
    NzMenuModule,
    NzModalModule,
    NzTreeSelectModule,
  ],
  templateUrl: './media-browser.html',
  styleUrl: './media-browser.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MediaBrowser {
  private readonly media = inject(MediaService);
  private readonly message = inject(NzMessageService);
  private readonly modal = inject(NzModalService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly gallery = viewChild(MediaGallery);

  readonly mode = input<'manage' | 'select'>('manage');
  /** Open folder; null is the library root. Two-way bindable. */
  readonly folderId = model<string | null>(null);
  /** Highlighted file in select mode. */
  readonly selectedId = input<string | null>(null);
  readonly select = output<UploadFile>();

  protected readonly rootName = ROOT_NAME;
  protected readonly folders = signal<Folder[]>([]);
  protected readonly foldersLoaded = signal(false);
  protected readonly manage = computed(() => this.mode() === 'manage');
  protected readonly path = computed(() => pathTo(this.folders(), this.folderId()));
  protected readonly subfolders = computed(() => childrenOf(this.folders(), this.folderId()));
  protected readonly emptyText = computed(() =>
    this.folderId() ? 'Esta carpeta no tiene archivos.' : 'No hay archivos en la raíz de la biblioteca.',
  );

  // Create / rename dialog
  protected readonly nameDialog = signal<NameDialog | null>(null);
  protected readonly nameControl = new FormControl('', {
    nonNullable: true,
    validators: [folderName, Validators.maxLength(100)],
  });
  /** Wraps the control so Enter submits through Angular instead of a native form post. */
  protected readonly nameForm = new FormGroup({ name: this.nameControl });

  // Move dialog
  protected readonly moveDialog = signal<MoveDialog | null>(null);
  protected readonly moveControl = new FormControl<string>(ROOT_KEY, { nonNullable: true });
  protected readonly moveNodes = computed(() => {
    const dialog = this.moveDialog();

    return destinationTree(this.folders(), dialog?.kind === 'folder' ? dialog.folder.id : undefined);
  });

  protected readonly saving = signal(false);

  constructor() {
    this.loadFolders();

    // A folder id that no longer exists (deleted elsewhere, stale link) falls back to the root.
    effect(() => {
      const id = this.folderId();
      if (id && this.foldersLoaded() && !this.folders().some((f) => f.id === id)) {
        this.folderId.set(null);
      }
    });
  }

  /** Refreshes folders and files, e.g. after an upload. */
  reload(): void {
    this.loadFolders();
    this.gallery()?.reload();
  }

  protected open(folder: Folder | null): void {
    this.folderId.set(folder?.id ?? null);
  }

  protected openCreate(): void {
    this.nameControl.reset('');
    this.nameDialog.set({ mode: 'create' });
  }

  protected openRename(folder: Folder): void {
    this.nameControl.reset(folder.name);
    this.nameDialog.set({ mode: 'rename', folder });
  }

  protected saveName(): void {
    const dialog = this.nameDialog();
    if (!dialog) {
      return;
    }

    if (this.nameControl.invalid) {
      showFormErrors(this.nameControl);

      return;
    }

    const name = this.nameControl.value.trim();
    const request =
      dialog.mode === 'create'
        ? this.media.createFolder(name, this.folderId())
        : this.media.updateFolder(dialog.folder.id, name, dialog.folder.parentId);

    this.run(request, dialog.mode === 'create' ? `Carpeta «${name}» creada` : 'Carpeta renombrada', () => {
      this.nameDialog.set(null);
      this.loadFolders();
    });
  }

  protected openMoveFile(file: UploadFile): void {
    this.moveControl.reset(file.folderId ?? ROOT_KEY);
    this.moveDialog.set({ kind: 'file', file });
  }

  protected openMoveFolder(folder: Folder): void {
    this.moveControl.reset(folder.parentId ?? ROOT_KEY);
    this.moveDialog.set({ kind: 'folder', folder });
  }

  protected confirmMove(): void {
    const dialog = this.moveDialog();
    if (!dialog) {
      return;
    }

    const destination = keyToFolderId(this.moveControl.value);

    if (dialog.kind === 'file') {
      this.run(this.media.moveFile(dialog.file.id, destination), `«${dialog.file.name}» movido`, () => {
        this.moveDialog.set(null);
        this.gallery()?.reload();
      });

      return;
    }

    this.run(
      this.media.updateFolder(dialog.folder.id, dialog.folder.name, destination),
      `Carpeta «${dialog.folder.name}» movida`,
      () => {
        this.moveDialog.set(null);
        this.loadFolders();
      },
    );
  }

  protected confirmDelete(folder: Folder): void {
    this.modal.confirm({
      nzTitle: `¿Eliminar la carpeta «${folder.name}»?`,
      nzContent: 'Solo se pueden eliminar carpetas vacías.',
      nzOkText: 'Eliminar',
      nzOkDanger: true,
      nzCancelText: 'Cancelar',
      nzOnOk: () => this.run(this.media.deleteFolder(folder.id), 'Carpeta eliminada', () => this.loadFolders()),
    });
  }

  private loadFolders(): void {
    this.media
      .folders()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (folders) => {
          this.folders.set(folders);
          this.foldersLoaded.set(true);
        },
        error: () => this.message.error('No se pudieron cargar las carpetas'),
      });
  }

  /** Runs a write, shows the outcome, and calls done on success. */
  private run(request: Observable<unknown>, success: string, done: () => void): void {
    this.saving.set(true);

    request
      .pipe(
        finalize(() => this.saving.set(false)),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe({
        next: () => {
          this.message.success(success);
          done();
        },
        error: (err: unknown) => this.message.error(folderErrorMessage(err)),
      });
  }
}

function folderErrorMessage(err: unknown): string {
  if (err instanceof HttpErrorResponse) {
    const message = (err.error as { error?: string } | null)?.error ?? '';

    if (err.status === HttpStatusCode.Conflict && message.includes('not empty')) {
      return 'La carpeta no está vacía: mueve o elimina antes su contenido.';
    }

    if (err.status === HttpStatusCode.Conflict) {
      return 'Ya hay una carpeta con ese nombre aquí.';
    }

    if (err.status === HttpStatusCode.BadRequest) {
      return 'No se puede: revisa el nombre, o el destino (una carpeta no puede ir dentro de sí misma).';
    }

    if (err.status === HttpStatusCode.NotFound) {
      return 'La carpeta ya no existe.';
    }
  }

  return 'No se pudo completar la operación.';
}
