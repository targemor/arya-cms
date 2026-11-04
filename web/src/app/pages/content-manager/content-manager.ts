import { DatePipe, NgTemplateOutlet } from '@angular/common';
import { HttpErrorResponse, HttpStatusCode } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  computed,
  effect,
  inject,
  input,
  signal,
  untracked,
} from '@angular/core';
import { takeUntilDestroyed, toObservable } from '@angular/core/rxjs-interop';
import { AbstractControl, FormArray, FormGroup, ReactiveFormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { NzButtonModule } from 'ng-zorro-antd/button';
import { NzCardModule } from 'ng-zorro-antd/card';
import { NzEmptyModule } from 'ng-zorro-antd/empty';
import { NzFormModule } from 'ng-zorro-antd/form';
import { NzIconModule } from 'ng-zorro-antd/icon';
import { NzInputModule } from 'ng-zorro-antd/input';
import { NzInputNumberModule } from 'ng-zorro-antd/input-number';
import { NzMessageService } from 'ng-zorro-antd/message';
import { NzModalModule } from 'ng-zorro-antd/modal';
import { NzPopconfirmModule } from 'ng-zorro-antd/popconfirm';
import { NzResultModule } from 'ng-zorro-antd/result';
import { NzSpinModule } from 'ng-zorro-antd/spin';
import { NzSwitchModule } from 'ng-zorro-antd/switch';
import { NzTableModule, NzTableQueryParams } from 'ng-zorro-antd/table';
import { NzTagModule } from 'ng-zorro-antd/tag';
import { catchError, filter, finalize, of, switchMap } from 'rxjs';

import { ComponentDefinition, ComponentsService, getComponentUID } from '../../core/content/components.service';
import { ContentTypesService } from '../../core/content/content-types.service';
import { ContentService, Entry, EntryQuery, FieldValue } from '../../core/content/content.service';
import { MediaRef, fileIcon } from '../../core/media/media.service';
import { showFormErrors } from '../../shared/forms';
import { MediaPicker } from '../../shared/media-picker/media-picker';
import { getFieldDisplayName } from '../content-type-builder/schema';
import { EntryForm, buildComponentGroup, buildEntryForm, isMediaRef, toPayload } from './entry-form';

const DEFAULT_PAGE_SIZE = 10;

function sameQuery(a: EntryQuery, b: EntryQuery): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}

@Component({
  selector: 'app-content-manager',
  imports: [
    DatePipe,
    NgTemplateOutlet,
    MediaPicker,
    ReactiveFormsModule,
    RouterLink,
    NzButtonModule,
    NzCardModule,
    NzEmptyModule,
    NzFormModule,
    NzIconModule,
    NzInputModule,
    NzInputNumberModule,
    NzModalModule,
    NzPopconfirmModule,
    NzResultModule,
    NzSpinModule,
    NzSwitchModule,
    NzTableModule,
    NzTagModule,
  ],
  templateUrl: './content-manager.html',
  styleUrl: './content-manager.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ContentManager {
  private readonly content = inject(ContentService);
  private readonly message = inject(NzMessageService);
  private readonly destroyRef = inject(DestroyRef);
  protected readonly contentTypes = inject(ContentTypesService);
  protected readonly componentsService = inject(ComponentsService);

  /** Route parameter (withComponentInputBinding); undefined on /content-manager. */
  readonly tableName = input<string>();

  protected readonly contentType = computed(
    () => this.contentTypes.types().find((ct) => ct.tableName === this.tableName()) ?? null,
  );

  protected readonly isSingleType = computed(() => this.contentType()?.kind === 'singleType');

  protected readonly collectionTypes = computed(() =>
    this.contentTypes.types().filter((ct) => ct.kind !== 'singleType'),
  );

  protected readonly singleTypes = computed(() =>
    this.contentTypes.types().filter((ct) => ct.kind === 'singleType'),
  );

  // Single Type state
  protected readonly singleEntry = signal<Entry | null>(null);
  protected readonly singleForm = signal<EntryForm | null>(null);
  protected readonly singleLoading = signal(false);
  protected readonly singleSaving = signal(false);
  protected readonly singleLoadFailed = signal(false);
  protected readonly singleReloadTick = signal(0);

  /** Page, size and sort, as set by the table; equal values do not refetch. */
  protected readonly query = signal<EntryQuery>(
    { page: 1, pageSize: DEFAULT_PAGE_SIZE, sort: [] },
    { equal: sameQuery },
  );
  private readonly reloadTick = signal(0);

  protected readonly entries = signal<Entry[]>([]);
  protected readonly total = signal(0);
  protected readonly loading = signal(false);
  protected readonly loadFailed = signal(false);

  // Create / edit modal
  protected readonly modalOpen = signal(false);
  protected readonly editing = signal<Entry | null>(null);
  protected readonly form = signal<EntryForm | null>(null);
  protected readonly saving = signal(false);

  protected readonly fileIcon = fileIcon;
  protected readonly isMediaRef = isMediaRef;
  protected readonly getFieldDisplayName = getFieldDisplayName;

  constructor() {
    this.componentsService.refresh();

    // A different content type starts again from page 1, unsorted.
    effect(() => {
      this.tableName();
      untracked(() => this.query.update((q) => ({ page: 1, pageSize: q.pageSize, sort: [] })));
    });

    // Collection type entries fetch
    toObservable(
      computed(() => ({
        ct: this.contentType(),
        isSingle: this.isSingleType(),
        q: this.query(),
        tick: this.reloadTick(),
      })),
    )
      .pipe(
        filter(({ ct, isSingle }) => ct !== null && !isSingle),
        switchMap(({ ct, q }) => {
          this.loading.set(true);
          this.loadFailed.set(false);

          return this.content.list(ct!.tableName, q).pipe(
            catchError(() => {
              this.loadFailed.set(true);

              return of(null);
            }),
            finalize(() => this.loading.set(false)),
          );
        }),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe((page) => {
        if (!page) {
          return;
        }

        // Deleting the last entry of the last page leaves it empty: step back a page.
        if (page.data.length === 0 && this.query().page > 1) {
          this.query.update((q) => ({ ...q, page: q.page - 1 }));

          return;
        }

        this.entries.set(page.data);
        this.total.set(page.meta.pagination.total);
      });

    // Single type entry fetch
    toObservable(
      computed(() => ({
        ct: this.contentType(),
        isSingle: this.isSingleType(),
        tick: this.singleReloadTick(),
      })),
    )
      .pipe(
        filter(({ ct, isSingle }) => ct !== null && isSingle),
        switchMap(({ ct }) => {
          this.singleLoading.set(true);
          this.singleLoadFailed.set(false);

          return this.content.getSingle(ct!.tableName).pipe(
            catchError(() => {
              this.singleLoadFailed.set(true);

              return of(null);
            }),
            finalize(() => this.singleLoading.set(false)),
          );
        }),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe((entry) => {
        const ct = this.contentType();
        if (!ct) {
          return;
        }
        this.singleEntry.set(entry);
        this.singleForm.set(buildEntryForm(ct, this.componentsService.components(), entry));
      });
  }

  /** nz-table's page, page size and sort changes, turned into the API's Strapi-style query. */
  protected onQueryParams({ pageIndex, pageSize, sort }: NzTableQueryParams): void {
    const active = sort.find((s) => s.value !== null);

    this.query.set({
      page: pageIndex,
      pageSize,
      sort: active ? [{ field: active.key, desc: active.value === 'descend' }] : [],
    });
  }

  /** Column sort indicator, driven by the query so it resets when the content type changes. */
  protected sortOrder(field: string): 'ascend' | 'descend' | null {
    const s = this.query().sort.find((x) => x.field === field);

    return s ? (s.desc ? 'descend' : 'ascend') : null;
  }

  protected reload(): void {
    this.reloadTick.update((n) => n + 1);
  }

  protected reloadSingle(): void {
    this.singleReloadTick.update((n) => n + 1);
  }

  protected openCreate(): void {
    const ct = this.contentType();
    if (!ct) {
      return;
    }

    this.editing.set(null);
    this.form.set(buildEntryForm(ct, this.componentsService.components()));
    this.modalOpen.set(true);
  }

  protected openEdit(entry: Entry): void {
    const ct = this.contentType();
    if (!ct) {
      return;
    }

    this.editing.set(entry);
    this.form.set(buildEntryForm(ct, this.componentsService.components(), entry));
    this.modalOpen.set(true);
  }

  protected closeModal(): void {
    this.modalOpen.set(false);
  }

  /** The file nested in the entry or component, for the media picker's preview. */
  protected mediaOf(field: string, parentData?: any): MediaRef | null {
    const source = parentData !== undefined ? parentData : (this.isSingleType() ? this.singleEntry() : this.editing());
    const value = source?.[field];

    return isMediaRef(value) ? value : null;
  }

  protected save(): void {
    const ct = this.contentType();
    const form = this.form();
    if (!ct || !form) {
      return;
    }

    if (form.invalid) {
      showFormErrors(form);

      return;
    }

    const data = toPayload(ct, form.getRawValue());
    const editing = this.editing();
    const request = editing
      ? this.content.update(ct.tableName, editing.id, data)
      : this.content.create(ct.tableName, data);

    this.saving.set(true);
    request.pipe(finalize(() => this.saving.set(false))).subscribe({
      next: () => {
        this.message.success(editing ? 'Entrada actualizada' : 'Entrada creada');
        this.modalOpen.set(false);
        this.reload();
      },
      error: (err: unknown) => this.message.error(saveErrorMessage(err)),
    });
  }

  protected saveSingle(): void {
    const ct = this.contentType();
    const form = this.singleForm();
    if (!ct || !form) {
      return;
    }

    if (form.invalid) {
      showFormErrors(form);

      return;
    }

    const data = toPayload(ct, form.getRawValue());
    this.singleSaving.set(true);
    this.content
      .updateSingle(ct.tableName, data)
      .pipe(finalize(() => this.singleSaving.set(false)))
      .subscribe({
        next: (entry) => {
          this.message.success('Guardado correctamente');
          this.singleEntry.set(entry);
          this.singleForm.set(buildEntryForm(ct, this.componentsService.components(), entry));
        },
        error: (err: unknown) => this.message.error(saveErrorMessage(err)),
      });
  }

  protected remove(entry: Entry): void {
    const ct = this.contentType();
    if (!ct) {
      return;
    }

    this.content.delete(ct.tableName, entry.id).subscribe({
      next: () => {
        this.message.success('Entrada eliminada');
        this.reload();
      },
      error: () => this.message.error('No se pudo eliminar la entrada'),
    });
  }

  protected getComponentDef(uid?: string): ComponentDefinition | null {
    if (!uid) {
      return null;
    }

    return this.componentsService.components().find((c) => getComponentUID(c) === uid) ?? null;
  }

  protected asFormGroup(ctrl: AbstractControl | null | undefined): FormGroup {
    return ctrl as FormGroup;
  }

  protected asFormArray(ctrl: AbstractControl | null | undefined): FormArray {
    return ctrl as FormArray;
  }

  protected asArray(val: unknown): unknown[] {
    return Array.isArray(val) ? val : [];
  }

  protected addComponentItem(arr: FormArray, comp: ComponentDefinition): void {
    arr.push(buildComponentGroup(comp));
  }

  protected removeComponentItem(arr: FormArray, index: number): void {
    arr.removeAt(index);
  }

  protected isEmpty(value: FieldValue | undefined): boolean {
    return value === null || value === undefined || value === '';
  }
}

function saveErrorMessage(err: unknown): string {
  if (err instanceof HttpErrorResponse && err.status === HttpStatusCode.BadRequest) {
    return 'Los datos no son válidos para este tipo de contenido.';
  }

  if (err instanceof HttpErrorResponse && err.status === HttpStatusCode.NotFound) {
    return 'La entrada ya no existe.';
  }

  return 'No se pudo guardar la entrada.';
}
