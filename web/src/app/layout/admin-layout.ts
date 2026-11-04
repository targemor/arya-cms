import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { RouterLink, RouterOutlet } from '@angular/router';
import { NzAvatarModule } from 'ng-zorro-antd/avatar';
import { NzButtonModule } from 'ng-zorro-antd/button';
import { NzDropdownModule } from 'ng-zorro-antd/dropdown';
import { NzFormModule } from 'ng-zorro-antd/form';
import { NzIconModule } from 'ng-zorro-antd/icon';
import { NzInputModule } from 'ng-zorro-antd/input';
import { NzLayoutModule } from 'ng-zorro-antd/layout';
import { NzMenuModule } from 'ng-zorro-antd/menu';
import { NzMessageService } from 'ng-zorro-antd/message';
import { NzModalModule } from 'ng-zorro-antd/modal';
import { NzTagModule } from 'ng-zorro-antd/tag';
import { catchError, of } from 'rxjs';

import { API_BASE } from '../core/api';
import { AuthService } from '../core/auth/auth.service';
import { ComponentsService } from '../core/content/components.service';
import { ContentTypesService } from '../core/content/content-types.service';
import { Project, ProjectsService } from '../core/project/projects.service';

@Component({
  selector: 'app-admin-layout',
  imports: [
    FormsModule,
    RouterLink,
    RouterOutlet,
    NzAvatarModule,
    NzButtonModule,
    NzDropdownModule,
    NzFormModule,
    NzIconModule,
    NzInputModule,
    NzLayoutModule,
    NzMenuModule,
    NzModalModule,
    NzTagModule,
  ],
  templateUrl: './admin-layout.html',
  styleUrl: './admin-layout.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminLayout {
  protected readonly auth = inject(AuthService);
  private readonly msg = inject(NzMessageService);
  private readonly http = inject(HttpClient);
  protected readonly projectsService = inject(ProjectsService);
  protected readonly contentTypes = inject(ContentTypesService);
  protected readonly components = inject(ComponentsService);

  protected readonly collapsed = signal(false);

  protected readonly collectionTypes = computed(() =>
    this.contentTypes.types().filter((ct) => (ct.kind ?? 'collectionType') === 'collectionType'),
  );

  protected readonly singleTypes = computed(() =>
    this.contentTypes.types().filter((ct) => ct.kind === 'singleType'),
  );

  /** Null while loading or if the profile call fails; the header then just shows the avatar. */
  protected readonly profile = toSignal(this.auth.profile().pipe(catchError(() => of(null))), {
    initialValue: null,
  });

  // Create project modal state
  protected readonly createModalVisible = signal(false);
  protected readonly creatingProject = signal(false);
  protected readonly uploadingCreateImage = signal(false);
  protected projectName = '';
  protected projectSlug = '';
  protected projectDescription = '';
  protected projectImageUrl = '';
  private slugManuallyEdited = false;

  constructor() {
    this.projectsService.load().subscribe({
      next: () => {
        this.contentTypes.load();
        this.components.load();
      },
      error: () => {
        this.contentTypes.load();
        this.components.load();
      },
    });
  }

  protected selectProject(project: Project): void {
    if (this.projectsService.activeProject()?.slug === project.slug) {
      return;
    }
    this.projectsService.setActive(project);
    this.contentTypes.refresh().subscribe();
    this.components.refresh().subscribe();
    this.msg.success(`Proyecto activo cambiado a: ${project.name}`);
  }

  protected openCreateProjectModal(): void {
    this.projectName = '';
    this.projectSlug = '';
    this.projectDescription = '';
    this.projectImageUrl = '';
    this.slugManuallyEdited = false;
    this.createModalVisible.set(true);
  }

  protected uploadImage(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) {
      return;
    }
    const file = input.files[0];
    const fd = new FormData();
    fd.append('file', file);

    this.uploadingCreateImage.set(true);
    this.http.post<{ url: string }>(`${API_BASE}/upload`, fd).subscribe({
      next: (res) => {
        this.uploadingCreateImage.set(false);
        this.projectImageUrl = res.url;
        input.value = '';
        this.msg.success('Imagen subida correctamente');
      },
      error: (err) => {
        this.uploadingCreateImage.set(false);
        input.value = '';
        this.msg.error(err.error?.error || 'Error al subir la imagen');
      },
    });
  }

  protected clearImage(): void {
    this.projectImageUrl = '';
  }

  protected closeCreateProjectModal(): void {
    this.createModalVisible.set(false);
  }

  protected onNameInput(name: string): void {
    this.projectName = name;
    if (!this.slugManuallyEdited) {
      this.projectSlug = this.slugify(name);
    }
  }

  protected onSlugInput(slug: string): void {
    this.slugManuallyEdited = true;
    this.projectSlug = slug;
  }

  protected submitCreateProject(): void {
    const name = this.projectName.trim();
    const slug = this.projectSlug.trim();
    if (!name || !slug) {
      this.msg.error('El nombre y el identificador (slug) son obligatorios');
      return;
    }

    this.creatingProject.set(true);
    this.projectsService
      .create({
        name,
        slug,
        description: this.projectDescription.trim() || undefined,
        imageUrl: this.projectImageUrl.trim() || undefined,
      })
      .subscribe({
        next: (created) => {
          this.creatingProject.set(false);
          this.createModalVisible.set(false);
          this.contentTypes.refresh().subscribe();
          this.components.refresh().subscribe();
          this.msg.success(`Proyecto "${created.name}" creado correctamente`);
        },
        error: (err) => {
          this.creatingProject.set(false);
          this.msg.error(err.error?.error || 'Error al crear el proyecto');
        },
      });
  }

  private slugify(val: string): string {
    return val
      .toLowerCase()
      .normalize('NFD')
      .replace(/[\u0300-\u036f]/g, '')
      .replace(/[^a-z0-9_]+/g, '_')
      .replace(/^_+|_+$/g, '')
      .slice(0, 50);
  }

  protected logout(): void {
    this.auth.logout();
  }
}
