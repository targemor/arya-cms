import { JsonPipe } from '@angular/common';
import { HttpErrorResponse, HttpStatusCode } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, OnInit, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormControl, FormGroup, NonNullableFormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { NzBadgeModule } from 'ng-zorro-antd/badge';
import { NzButtonModule } from 'ng-zorro-antd/button';
import { NzCardModule } from 'ng-zorro-antd/card';
import { NzCheckboxModule } from 'ng-zorro-antd/checkbox';
import { NzCollapseModule } from 'ng-zorro-antd/collapse';
import { NzDropdownModule } from 'ng-zorro-antd/dropdown';
import { NzEmptyModule } from 'ng-zorro-antd/empty';
import { NzFormModule } from 'ng-zorro-antd/form';
import { NzGridModule } from 'ng-zorro-antd/grid';
import { NzIconModule } from 'ng-zorro-antd/icon';
import { NzInputModule } from 'ng-zorro-antd/input';
import { NzMessageService } from 'ng-zorro-antd/message';
import { NzModalModule } from 'ng-zorro-antd/modal';
import { NzPopconfirmModule } from 'ng-zorro-antd/popconfirm';
import { NzRadioModule } from 'ng-zorro-antd/radio';
import { NzSelectModule } from 'ng-zorro-antd/select';
import { NzTagModule } from 'ng-zorro-antd/tag';
import { finalize } from 'rxjs';

import { ComponentDefinition, ComponentsService, getComponentUID } from '../../core/content/components.service';
import { Attribute, AttributeType, ContentType, ContentTypeKind, ContentTypesService } from '../../core/content/content-types.service';
import { showFormErrors } from '../../shared/forms';
import { ATTRIBUTE_TYPES, IDENTIFIER_MAX_LENGTH, IDENTIFIER_PATTERN, getFieldDisplayName, notReserved, toJsonSchema } from './schema';

function toSnakeCase(val: string): string {
  return val
    .toLowerCase()
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/[^a-z0-9_]+/g, '_')
    .replace(/^_+|_+$/g, '')
    .slice(0, IDENTIFIER_MAX_LENGTH);
}

type SelectedTarget =
  | { kind: 'contentType'; data: ContentType }
  | { kind: 'component'; data: ComponentDefinition }
  | null;

@Component({
  selector: 'app-content-type-builder',
  imports: [
    JsonPipe,
    ReactiveFormsModule,
    RouterLink,
    NzBadgeModule,
    NzButtonModule,
    NzCardModule,
    NzCheckboxModule,
    NzCollapseModule,
    NzDropdownModule,
    NzEmptyModule,
    NzFormModule,
    NzGridModule,
    NzIconModule,
    NzInputModule,
    NzModalModule,
    NzPopconfirmModule,
    NzRadioModule,
    NzSelectModule,
    NzTagModule,
  ],
  templateUrl: './content-type-builder.html',
  styleUrl: './content-type-builder.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ContentTypeBuilder implements OnInit {
  private readonly fb = inject(NonNullableFormBuilder);
  private readonly message = inject(NzMessageService);
  protected readonly contentTypes = inject(ContentTypesService);
  protected readonly componentsService = inject(ComponentsService);

  protected readonly attributeTypes = ATTRIBUTE_TYPES;
  protected readonly maxLength = IDENTIFIER_MAX_LENGTH;

  // Search filter for left sidebar
  protected readonly searchQuery = signal('');

  // Selected item to display in the main panel
  protected readonly selectedTarget = signal<SelectedTarget>(null);

  // Dirty state tracking (for the Strapi top-left Save button)
  protected readonly isDirty = signal(false);
  protected readonly saving = signal(false);

  // Grouped lists
  protected readonly collectionTypes = computed(() => {
    const q = this.searchQuery().toLowerCase().trim();

    return this.contentTypes.types()
      .filter((ct) => (ct.kind ?? 'collectionType') === 'collectionType')
      .filter((ct) => !q || (ct.displayName || ct.tableName).toLowerCase().includes(q));
  });

  protected readonly singleTypes = computed(() => {
    const q = this.searchQuery().toLowerCase().trim();

    return this.contentTypes.types()
      .filter((ct) => ct.kind === 'singleType')
      .filter((ct) => !q || (ct.displayName || ct.tableName).toLowerCase().includes(q));
  });

  protected readonly componentCategories = computed(() => {
    const q = this.searchQuery().toLowerCase().trim();
    const comps = this.componentsService.components().filter(
      (c) => !q || (c.displayName || c.name).toLowerCase().includes(q) || c.category.toLowerCase().includes(q),
    );

    const map = new Map<string, ComponentDefinition[]>();
    for (const c of comps) {
      const cat = c.category || 'default';
      const list = map.get(cat) ?? [];
      list.push(c);
      map.set(cat, list);
    }

    return Array.from(map.entries()).map(([name, items]) => ({ name, items }));
  });

  // Modals state
  protected readonly createTypeModalOpen = signal(false);
  protected readonly createTypeKind = signal<ContentTypeKind>('collectionType');
  protected readonly createTypeForm = this.fb.group({
    displayName: ['', [Validators.required, Validators.maxLength(100)]],
    tableName: ['', [Validators.required, Validators.pattern(IDENTIFIER_PATTERN), Validators.maxLength(IDENTIFIER_MAX_LENGTH)]],
  });

  protected readonly createComponentModalOpen = signal(false);
  protected readonly createComponentForm = this.fb.group({
    displayName: ['', [Validators.required, Validators.maxLength(100)]],
    category: ['shared', [Validators.required, Validators.pattern(IDENTIFIER_PATTERN), Validators.maxLength(IDENTIFIER_MAX_LENGTH)]],
    name: ['', [Validators.required, Validators.pattern(IDENTIFIER_PATTERN), Validators.maxLength(IDENTIFIER_MAX_LENGTH)]],
    icon: ['appstore'],
  });

  // Field creation modals
  protected readonly selectFieldModalOpen = signal(false);
  protected readonly configureFieldModalOpen = signal(false);
  protected readonly selectedFieldType = signal<AttributeType>('string');
  protected readonly editingFieldIndex = signal<number | null>(null);

  protected readonly fieldForm = new FormGroup({
    name: new FormControl<string>('', {
      nonNullable: true,
      validators: [Validators.required, Validators.pattern(IDENTIFIER_PATTERN), Validators.maxLength(IDENTIFIER_MAX_LENGTH), notReserved],
    }),
    displayName: new FormControl<string>('', { nonNullable: true }),
    type: new FormControl<AttributeType>('string', { nonNullable: true }),
    required: new FormControl<boolean>(false, { nonNullable: true }),
    componentMode: new FormControl<'existing' | 'new'>('existing', { nonNullable: true }),
    component: new FormControl<string>('', { nonNullable: true }),
    repeatable: new FormControl<boolean>(false, { nonNullable: true }),
    // New component fields if created on the fly
    newCompDisplayName: new FormControl<string>(''),
    newCompCategory: new FormControl<string>('shared'),
    newCompName: new FormControl<string>(''),
  });

  protected readonly getFieldDisplayName = getFieldDisplayName;

  // Current working attributes (can be modified and then saved)
  protected readonly currentAttributes = signal<Attribute[]>([]);

  constructor() {
    this.fieldForm.controls.displayName.valueChanges
      .pipe(takeUntilDestroyed())
      .subscribe((title) => {
        if (this.editingFieldIndex() === null && title) {
          this.fieldForm.controls.name.setValue(toSnakeCase(title));
        }
      });
  }

  ngOnInit(): void {
    this.contentTypes.load();
    this.componentsService.load();

    // Auto-select first item when loaded
    this.contentTypes.refresh().subscribe((types) => {
      if (types.length > 0 && !this.selectedTarget()) {
        this.selectContentType(types[0]);
      }
    });
  }

  protected selectContentType(ct: ContentType): void {
    this.selectedTarget.set({ kind: 'contentType', data: { ...ct } });
    this.currentAttributes.set([...ct.attributes]);
    this.isDirty.set(false);
  }

  protected selectComponent(comp: ComponentDefinition): void {
    this.selectedTarget.set({ kind: 'component', data: { ...comp } });
    this.currentAttributes.set([...comp.attributes]);
    this.isDirty.set(false);
  }

  protected isSelected(target: { tableName?: string; uid?: string }): boolean {
    const cur = this.selectedTarget();
    if (!cur) return false;
    if (cur.kind === 'contentType' && target.tableName) {
      return cur.data.tableName === target.tableName;
    }
    if (cur.kind === 'component' && target.uid) {
      return getComponentUID(cur.data) === target.uid;
    }

    return false;
  }

  // Opening type creation modal
  protected openCreateTypeModal(kind: ContentTypeKind): void {
    this.createTypeKind.set(kind);
    this.createTypeForm.reset();
    this.createTypeModalOpen.set(true);
  }

  protected onDisplayNameChange(value: string): void {
    const slug = value
      .toLowerCase()
      .trim()
      .normalize('NFD')
      .replace(/[\u0300-\u036f]/g, '')
      .replace(/[^a-z0-9_]+/g, '_')
      .replace(/^_+|_+$/g, '');

    this.createTypeForm.controls.tableName.setValue(slug);
  }

  protected onCompDisplayNameChange(value: string): void {
    const slug = value
      .toLowerCase()
      .trim()
      .normalize('NFD')
      .replace(/[\u0300-\u036f]/g, '')
      .replace(/[^a-z0-9_]+/g, '_')
      .replace(/^_+|_+$/g, '');

    this.createComponentForm.controls.name.setValue(slug);
  }

  protected submitCreateType(): void {
    if (this.createTypeForm.invalid) {
      showFormErrors(this.createTypeForm);

      return;
    }

    const { displayName, tableName } = this.createTypeForm.getRawValue();
    const newType: ContentType = {
      tableName,
      displayName,
      kind: this.createTypeKind(),
      attributes: [{ name: 'nombre', type: 'string', required: true }],
    };

    this.saving.set(true);
    this.contentTypes.create(newType)
      .pipe(finalize(() => this.saving.set(false)))
      .subscribe({
        next: (created) => {
          this.message.success(`Tipo «${created.displayName || created.tableName}» creado`);
          this.createTypeModalOpen.set(false);
          this.selectContentType(created);
        },
        error: (err: unknown) => this.message.error(createErrorMessage(err)),
      });
  }

  protected openCreateComponentModal(): void {
    this.createComponentForm.reset({ category: 'shared', icon: 'appstore' });
    this.createComponentModalOpen.set(true);
  }

  protected submitCreateComponent(): void {
    if (this.createComponentForm.invalid) {
      showFormErrors(this.createComponentForm);

      return;
    }

    const { displayName, category, name, icon } = this.createComponentForm.getRawValue();
    const newComp: ComponentDefinition = {
      name,
      category,
      displayName,
      icon,
      attributes: [{ name: 'titulo', type: 'string', required: true }],
    };

    this.saving.set(true);
    this.componentsService.create(newComp)
      .pipe(finalize(() => this.saving.set(false)))
      .subscribe({
        next: (created) => {
          this.message.success(`Componente «${created.displayName || created.name}» creado`);
          this.createComponentModalOpen.set(false);
          this.selectComponent(created);
        },
        error: () => this.message.error('No se pudo crear el componente.'),
      });
  }

  // Field addition flow
  protected openAddField(): void {
    this.selectFieldModalOpen.set(true);
  }

  protected chooseFieldType(type: AttributeType): void {
    this.selectFieldModalOpen.set(false);
    this.selectedFieldType.set(type);
    this.editingFieldIndex.set(null);

    const defaultComp = this.componentsService.components()[0];
    this.fieldForm.reset({
      type,
      name: '',
      displayName: '',
      required: false,
      componentMode: 'existing',
      component: defaultComp ? getComponentUID(defaultComp) : '',
      repeatable: false,
      newCompCategory: 'shared',
    });

    this.configureFieldModalOpen.set(true);
  }

  protected openEditField(index: number): void {
    const attr = this.currentAttributes()[index];
    if (!attr) return;

    this.selectedFieldType.set(attr.type);
    this.editingFieldIndex.set(index);

    this.fieldForm.reset({
      name: attr.name,
      displayName: attr.displayName || '',
      type: attr.type,
      required: attr.required,
      componentMode: 'existing',
      component: attr.component || '',
      repeatable: attr.repeatable || false,
    });

    this.configureFieldModalOpen.set(true);
  }

  protected removeField(index: number): void {
    if (this.currentAttributes().length <= 1) {
      this.message.warning('Un tipo debe tener al menos un campo.');

      return;
    }

    const attrs = [...this.currentAttributes()];
    attrs.splice(index, 1);
    this.currentAttributes.set(attrs);
    this.isDirty.set(true);
  }

  protected submitField(): void {
    if (this.fieldForm.invalid) {
      showFormErrors(this.fieldForm);

      return;
    }

    const val = this.fieldForm.getRawValue();
    const type = this.selectedFieldType();
    const displayName = val.displayName.trim() || undefined;

    // Check for duplicate name
    const editIdx = this.editingFieldIndex();
    const exists = this.currentAttributes().some((a, i) => i !== editIdx && a.name === val.name);
    if (exists) {
      this.message.error('Ya existe un campo con ese nombre.');

      return;
    }

    if (type === 'component') {
      if (val.componentMode === 'new') {
        const cat = val.newCompCategory || 'shared';
        const name = val.newCompName;
        if (!name) {
          this.message.error('Ingresa el nombre del nuevo componente.');

          return;
        }

        const newComp: ComponentDefinition = {
          name,
          category: cat,
          displayName: val.newCompDisplayName || name,
          attributes: [{ name: 'texto', type: 'string', required: true }],
        };

        this.saving.set(true);
        this.componentsService.create(newComp)
          .pipe(finalize(() => this.saving.set(false)))
          .subscribe({
            next: (created) => {
              this.applyField({
                name: val.name,
                displayName,
                type: 'component',
                required: val.required,
                component: getComponentUID(created),
                repeatable: val.repeatable,
              });
              this.configureFieldModalOpen.set(false);
            },
            error: () => this.message.error('No se pudo crear el componente.'),
          });

        return;
      }

      if (!val.component) {
        this.message.error('Selecciona un componente.');

        return;
      }
    }

    this.applyField({
      name: val.name,
      displayName,
      type,
      required: val.required,
      component: type === 'component' ? val.component : undefined,
      repeatable: type === 'component' ? val.repeatable : undefined,
    });

    this.configureFieldModalOpen.set(false);
  }

  private applyField(attr: Attribute): void {
    const attrs = [...this.currentAttributes()];
    const idx = this.editingFieldIndex();

    if (idx !== null && idx >= 0) {
      attrs[idx] = attr;
    } else {
      attrs.push(attr);
    }

    this.currentAttributes.set(attrs);
    this.isDirty.set(true);
  }

  // Save changes to current selected target (Save button)
  protected saveCurrent(): void {
    const target = this.selectedTarget();
    if (!target) return;

    this.saving.set(true);
    if (target.kind === 'contentType') {
      const updated: ContentType = {
        ...target.data,
        attributes: this.currentAttributes(),
      };

      this.contentTypes.update(target.data.tableName, updated)
        .pipe(finalize(() => this.saving.set(false)))
        .subscribe({
          next: (saved) => {
            this.message.success(`«${saved.displayName || saved.tableName}» guardado`);
            this.selectContentType(saved);
          },
          error: (err: unknown) => this.message.error(createErrorMessage(err)),
        });
    } else if (target.kind === 'component') {
      const updated: ComponentDefinition = {
        ...target.data,
        attributes: this.currentAttributes(),
      };

      this.componentsService.update(getComponentUID(target.data), updated)
        .pipe(finalize(() => this.saving.set(false)))
        .subscribe({
          next: (saved) => {
            this.message.success(`Componente «${saved.displayName || saved.name}» guardado`);
            this.selectComponent(saved);
          },
          error: () => this.message.error('No se pudo guardar el componente.'),
        });
    }
  }

  protected previewJson(): object {
    const target = this.selectedTarget();
    if (!target) return {};

    if (target.kind === 'contentType') {
      return toJsonSchema({ ...target.data, attributes: this.currentAttributes() });
    }

    return {
      $schema: 'https://json-schema.org/draft/2020-12/schema',
      title: target.data.name,
      'x-cms-kind': 'component',
      'x-cms-category': target.data.category,
      type: 'object',
      properties: Object.fromEntries(this.currentAttributes().map((a) => [a.name, { type: a.type }])),
    };
  }

  protected getComponentUID = getComponentUID;
}

function createErrorMessage(err: unknown): string {
  if (err instanceof HttpErrorResponse) {
    if (err.status === HttpStatusCode.Conflict) {
      return 'Ya existe un tipo de contenido con ese nombre.';
    }

    if (err.status === HttpStatusCode.BadRequest) {
      return 'La definición no es válida. Revisa los nombres de la tabla y de los campos.';
    }
  }

  return 'No se pudo guardar.';
}
