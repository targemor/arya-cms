import { DatePipe } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, OnInit, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { NzAlertModule } from 'ng-zorro-antd/alert';
import { NzAvatarModule } from 'ng-zorro-antd/avatar';
import { NzButtonModule } from 'ng-zorro-antd/button';
import { NzCardModule } from 'ng-zorro-antd/card';
import { NzCheckboxModule } from 'ng-zorro-antd/checkbox';
import { NzDividerModule } from 'ng-zorro-antd/divider';
import { NzEmptyModule } from 'ng-zorro-antd/empty';
import { NzFormModule } from 'ng-zorro-antd/form';
import { NzIconModule } from 'ng-zorro-antd/icon';
import { NzInputModule } from 'ng-zorro-antd/input';
import { NzMessageService } from 'ng-zorro-antd/message';
import { NzModalModule } from 'ng-zorro-antd/modal';
import { NzPopconfirmModule } from 'ng-zorro-antd/popconfirm';
import { NzRadioModule } from 'ng-zorro-antd/radio';
import { NzSelectModule } from 'ng-zorro-antd/select';
import { NzSpinModule } from 'ng-zorro-antd/spin';
import { NzTableModule } from 'ng-zorro-antd/table';
import { NzTabsModule } from 'ng-zorro-antd/tabs';
import { NzTagModule } from 'ng-zorro-antd/tag';
import { NzTooltipModule } from 'ng-zorro-antd/tooltip';

import { API_BASE } from '../../core/api';
import { AuthService, Profile, UserRole } from '../../core/auth/auth.service';
import { ComponentsService } from '../../core/content/components.service';
import { ContentType, ContentTypesService } from '../../core/content/content-types.service';
import {
  APIKey,
  APIKeyAccessType,
  APIKeyAction,
  Project,
  ProjectMember,
  ProjectsService,
} from '../../core/project/projects.service';

@Component({
  selector: 'app-super-admin',
  imports: [
    DatePipe,
    FormsModule,
    NzAlertModule,
    NzAvatarModule,
    NzButtonModule,
    NzCardModule,
    NzCheckboxModule,
    NzDividerModule,
    NzEmptyModule,
    NzFormModule,
    NzIconModule,
    NzInputModule,
    NzModalModule,
    NzPopconfirmModule,
    NzRadioModule,
    NzSelectModule,
    NzSpinModule,
    NzTableModule,
    NzTabsModule,
    NzTagModule,
    NzTooltipModule,
  ],
  templateUrl: './super-admin.html',
  styleUrl: './super-admin.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SuperAdmin implements OnInit {
  protected readonly auth = inject(AuthService);
  protected readonly projectsService = inject(ProjectsService);
  private readonly contentTypes = inject(ContentTypesService);
  private readonly components = inject(ComponentsService);
  private readonly msg = inject(NzMessageService);
  private readonly http = inject(HttpClient);

  // Users state
  protected readonly users = signal<Profile[]>([]);
  protected readonly loadingUsers = signal(false);

  // Modal Create Project state
  protected readonly createModalVisible = signal(false);
  protected readonly creatingProject = signal(false);
  protected readonly uploadingCreateImage = signal(false);
  protected projectName = '';
  protected projectSlug = '';
  protected projectDescription = '';
  protected projectImageUrl = '';
  private slugManuallyEdited = false;

  // Modal Edit Project state
  protected readonly editModalVisible = signal(false);
  protected readonly editingProject = signal<Project | null>(null);
  protected readonly updatingProject = signal(false);
  protected readonly uploadingEditImage = signal(false);
  protected editName = '';
  protected editSlug = '';
  protected editDescription = '';
  protected editImageUrl = '';

  // Modal Project Members & Roles state
  protected readonly membersModalVisible = signal(false);
  protected readonly selectedProjectForMembers = signal<Project | null>(null);
  protected readonly projectMembers = signal<ProjectMember[]>([]);
  protected readonly loadingMembers = signal(false);
  protected selectedUserIdToAdd: string | null = null;
  protected selectedMemberRole: 'admin' | 'editor' | 'viewer' = 'editor';
  protected readonly addingMember = signal(false);

  // Modal & Tab Project API Keys state
  protected readonly apiKeysModalVisible = signal(false);
  protected readonly selectedProjectForKeys = signal<Project | null>(null);
  protected readonly projectAPIKeys = signal<APIKey[]>([]);
  protected readonly projectSchemas = signal<ContentType[]>([]);
  protected readonly loadingAPIKeys = signal(false);
  protected readonly savingAPIKey = signal(false);
  protected readonly editingAPIKey = signal<APIKey | null>(null);
  protected readonly apiKeyFormVisible = signal(false);
  protected readonly newlyCreatedToken = signal<string | null>(null);
  protected apiKeyName = '';
  protected apiKeyDescription = '';
  protected apiKeyAccessType: APIKeyAccessType = 'custom';
  protected readonly apiKeyPermissions = signal<Record<string, APIKeyAction[]>>({});

  readonly allAPIKeyActions: { label: string; shortLabel: string; value: APIKeyAction; color: string }[] = [
    { label: 'Listar (find)', shortLabel: 'find', value: 'find', color: 'blue' },
    { label: 'Ver detalle (findOne)', shortLabel: 'findOne', value: 'findOne', color: 'cyan' },
    { label: 'Crear (create)', shortLabel: 'create', value: 'create', color: 'green' },
    { label: 'Actualizar (update)', shortLabel: 'update', value: 'update', color: 'orange' },
    { label: 'Eliminar (delete)', shortLabel: 'delete', value: 'delete', color: 'red' },
  ];

  readonly availableRoles: { label: string; value: UserRole; color: string }[] = [
    { label: 'Super Admin', value: 'superadmin', color: 'magenta' },
    { label: 'Administrador', value: 'admin', color: 'blue' },
    { label: 'Editor', value: 'editor', color: 'green' },
    { label: 'Lector (Viewer)', value: 'viewer', color: 'default' },
  ];

  readonly projectRoles: { label: string; value: 'admin' | 'editor' | 'viewer'; color: string }[] = [
    { label: 'Administrador', value: 'admin', color: 'blue' },
    { label: 'Editor', value: 'editor', color: 'green' },
    { label: 'Lector (Viewer)', value: 'viewer', color: 'default' },
  ];

  ngOnInit(): void {
    this.projectsService.load().subscribe();
    this.loadUsers();
  }

  loadUsers(): void {
    this.loadingUsers.set(true);
    this.auth.listUsers().subscribe({
      next: (data) => {
        this.users.set(data);
        this.loadingUsers.set(false);
      },
      error: (err) => {
        this.loadingUsers.set(false);
        this.msg.error(err.error?.error || 'Error al cargar usuarios');
      },
    });
  }

  selectProject(project: Project): void {
    this.projectsService.setActive(project);
    this.contentTypes.refresh().subscribe();
    this.components.refresh().subscribe();
    this.msg.success(`Proyecto activo: ${project.name}`);
  }

  openCreateProjectModal(): void {
    this.projectName = '';
    this.projectSlug = '';
    this.projectDescription = '';
    this.projectImageUrl = '';
    this.slugManuallyEdited = false;
    this.createModalVisible.set(true);
  }

  closeCreateProjectModal(): void {
    this.createModalVisible.set(false);
  }

  openEditProjectModal(project: Project): void {
    this.editingProject.set(project);
    this.editName = project.name;
    this.editSlug = project.slug;
    this.editDescription = project.description || '';
    this.editImageUrl = project.imageUrl || '';
    this.editModalVisible.set(true);
  }

  closeEditProjectModal(): void {
    this.editModalVisible.set(false);
    this.editingProject.set(null);
  }

  // --- GESTIÓN DE MIEMBROS POR PROYECTO ---
  openMembersModal(project: Project): void {
    this.selectedProjectForMembers.set(project);
    this.selectedUserIdToAdd = null;
    this.selectedMemberRole = 'editor';
    this.membersModalVisible.set(true);
    this.loadProjectMembers(project.slug);
  }

  closeMembersModal(): void {
    this.membersModalVisible.set(false);
    this.selectedProjectForMembers.set(null);
    this.projectMembers.set([]);
  }

  loadProjectMembers(slug: string): void {
    this.loadingMembers.set(true);
    this.projectsService.getMembers(slug).subscribe({
      next: (members) => {
        this.projectMembers.set(members);
        this.loadingMembers.set(false);
      },
      error: (err) => {
        this.loadingMembers.set(false);
        this.msg.error(err.error?.error || 'Error al cargar miembros del proyecto');
      },
    });
  }

  addMemberToProject(): void {
    const project = this.selectedProjectForMembers();
    if (!project || !this.selectedUserIdToAdd) {
      this.msg.error('Selecciona un usuario para asignar al proyecto');
      return;
    }

    this.addingMember.set(true);
    this.projectsService
      .addOrUpdateMember(project.slug, {
        userId: this.selectedUserIdToAdd,
        role: this.selectedMemberRole,
      })
      .subscribe({
        next: () => {
          this.addingMember.set(false);
          this.selectedUserIdToAdd = null;
          this.msg.success('Usuario asignado al proyecto exitosamente');
          this.loadProjectMembers(project.slug);
        },
        error: (err) => {
          this.addingMember.set(false);
          this.msg.error(err.error?.error || 'Error al asignar usuario');
        },
      });
  }

  updateMemberRole(member: ProjectMember, newRole: 'admin' | 'editor' | 'viewer'): void {
    const project = this.selectedProjectForMembers();
    if (!project || member.role === newRole) {
      return;
    }

    this.projectsService
      .addOrUpdateMember(project.slug, {
        userId: member.userId,
        role: newRole,
      })
      .subscribe({
        next: () => {
          this.projectMembers.update((list) =>
            list.map((m) => (m.userId === member.userId ? { ...m, role: newRole } : m)),
          );
          this.msg.success(`Rol de "${member.username}" actualizado a ${newRole} en este proyecto`);
        },
        error: (err) => {
          this.msg.error(err.error?.error || 'Error al actualizar rol en el proyecto');
        },
      });
  }

  removeMemberFromProject(member: ProjectMember): void {
    const project = this.selectedProjectForMembers();
    if (!project) {
      return;
    }

    this.projectsService.removeMember(project.slug, member.userId).subscribe({
      next: () => {
        this.projectMembers.update((list) => list.filter((m) => m.userId !== member.userId));
        this.msg.success(`Usuario "${member.username}" desvinculado del proyecto`);
      },
      error: (err) => {
        this.msg.error(err.error?.error || 'Error al desvincular usuario');
      },
    });
  }

  availableUsersToAdd(): Profile[] {
    const existingIds = new Set(this.projectMembers().map((m) => m.userId));
    return this.users().filter((u) => !existingIds.has(u.id));
  }

  onNameInput(name: string): void {
    this.projectName = name;
    if (!this.slugManuallyEdited) {
      this.projectSlug = this.slugify(name);
    }
  }

  onSlugInput(slug: string): void {
    this.slugManuallyEdited = true;
    this.projectSlug = slug;
  }

  uploadImage(event: Event, isEdit: boolean): void {
    const input = event.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) {
      return;
    }
    const file = input.files[0];
    const fd = new FormData();
    fd.append('file', file);

    const flag = isEdit ? this.uploadingEditImage : this.uploadingCreateImage;
    flag.set(true);

    this.http.post<{ url: string }>(`${API_BASE}/upload`, fd).subscribe({
      next: (res) => {
        flag.set(false);
        if (isEdit) {
          this.editImageUrl = res.url;
        } else {
          this.projectImageUrl = res.url;
        }
        input.value = '';
        this.msg.success('Imagen subida correctamente');
      },
      error: (err) => {
        flag.set(false);
        input.value = '';
        this.msg.error(err.error?.error || 'Error al subir la imagen');
      },
    });
  }

  clearImage(isEdit: boolean): void {
    if (isEdit) {
      this.editImageUrl = '';
    } else {
      this.projectImageUrl = '';
    }
  }

  submitCreateProject(): void {
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
          this.msg.success(`Proyecto "${created.name}" creado con éxito`);
        },
        error: (err) => {
          this.creatingProject.set(false);
          this.msg.error(err.error?.error || 'Error al crear el proyecto');
        },
      });
  }

  submitEditProject(): void {
    const project = this.editingProject();
    if (!project) {
      return;
    }
    const name = this.editName.trim();
    if (!name) {
      this.msg.error('El nombre del proyecto es obligatorio');
      return;
    }

    this.updatingProject.set(true);
    this.projectsService
      .update(project.slug, {
        name,
        description: this.editDescription.trim() || undefined,
        imageUrl: this.editImageUrl.trim() || undefined,
      })
      .subscribe({
        next: (updated) => {
          this.updatingProject.set(false);
          this.editModalVisible.set(false);
          this.editingProject.set(null);
          this.msg.success(`Proyecto "${updated.name}" actualizado`);
        },
        error: (err) => {
          this.updatingProject.set(false);
          this.msg.error(err.error?.error || 'Error al actualizar el proyecto');
        },
      });
  }

  deleteProject(project: Project): void {
    if (project.slug === 'default') {
      this.msg.warning('No es posible eliminar el proyecto predeterminado (default)');
      return;
    }

    this.projectsService.delete(project.slug).subscribe({
      next: () => {
        this.contentTypes.refresh().subscribe();
        this.components.refresh().subscribe();
        this.msg.success(`Proyecto "${project.name}" eliminado`);
      },
      error: (err) => {
        this.msg.error(err.error?.error || 'Error al eliminar el proyecto');
      },
    });
  }

  changeUserRole(user: Profile, newRole: UserRole): void {
    if (user.role === newRole) {
      return;
    }

    this.auth.updateUserRole(user.id, newRole).subscribe({
      next: () => {
        this.users.update((list) =>
          list.map((u) => (u.id === user.id ? { ...u, role: newRole } : u)),
        );
        this.msg.success(`Rol de "${user.username}" actualizado a ${newRole}`);
      },
      error: (err) => {
        this.msg.error(err.error?.error || 'Error al actualizar el rol');
      },
    });
  }

  getRoleColor(role: UserRole): string {
    const found = this.availableRoles.find((r) => r.value === role);
    return found ? found.color : 'default';
  }

  getRoleLabel(role: UserRole): string {
    const found = this.availableRoles.find((r) => r.value === role);
    return found ? found.label : role;
  }

  getProjectRoleColor(role: string): string {
    const found = this.projectRoles.find((r) => r.value === role);
    return found ? found.color : 'default';
  }

  // --- GESTIÓN DE API KEYS Y PERMISOS GRANULARES POR CONTENT TYPE ---
  openAPIKeysModal(project: Project): void {
    this.selectedProjectForKeys.set(project);
    this.newlyCreatedToken.set(null);
    this.apiKeyFormVisible.set(false);
    this.editingAPIKey.set(null);
    this.apiKeysModalVisible.set(true);
    this.loadProjectAPIKeysAndSchemas(project.slug);
  }

  closeAPIKeysModal(): void {
    this.apiKeysModalVisible.set(false);
    this.selectedProjectForKeys.set(null);
    this.projectAPIKeys.set([]);
    this.projectSchemas.set([]);
    this.newlyCreatedToken.set(null);
    this.apiKeyFormVisible.set(false);
    this.editingAPIKey.set(null);
  }

  loadProjectAPIKeysAndSchemas(slug: string): void {
    this.loadingAPIKeys.set(true);
    this.projectsService.getAPIKeys(slug).subscribe({
      next: (keys) => {
        this.projectAPIKeys.set(keys);
        this.loadingAPIKeys.set(false);
      },
      error: (err) => {
        this.loadingAPIKeys.set(false);
        this.msg.error(err.error?.error || 'Error al cargar API Keys del proyecto');
      },
    });

    this.http
      .get<ContentType[]>(`${API_BASE}/internal/schemas`, {
        headers: { 'X-Project': slug },
      })
      .subscribe({
        next: (schemas) => {
          this.projectSchemas.set(schemas ?? []);
        },
        error: () => {
          this.projectSchemas.set([]);
        },
      });
  }

  openCreateAPIKeyForm(): void {
    this.editingAPIKey.set(null);
    this.apiKeyName = '';
    this.apiKeyDescription = '';
    this.apiKeyAccessType = 'custom';
    const initialPerms: Record<string, APIKeyAction[]> = {};
    for (const ct of this.projectSchemas()) {
      initialPerms[ct.tableName] = ['find', 'findOne'];
    }
    this.apiKeyPermissions.set(initialPerms);
    this.apiKeyFormVisible.set(true);
  }

  openEditAPIKeyForm(key: APIKey): void {
    this.editingAPIKey.set(key);
    this.apiKeyName = key.name;
    this.apiKeyDescription = key.description || '';
    this.apiKeyAccessType = key.accessType;
    const clonedPerms: Record<string, APIKeyAction[]> = {};
    if (key.permissions) {
      for (const [table, actions] of Object.entries(key.permissions)) {
        clonedPerms[table] = [...actions];
      }
    }
    this.apiKeyPermissions.set(clonedPerms);
    this.apiKeyFormVisible.set(true);
  }

  cancelAPIKeyForm(): void {
    this.apiKeyFormVisible.set(false);
    this.editingAPIKey.set(null);
  }

  hasPermission(tableName: string, action: APIKeyAction): boolean {
    const perms = this.apiKeyPermissions()[tableName];
    return Array.isArray(perms) && perms.includes(action);
  }

  togglePermission(tableName: string, action: APIKeyAction, checked: boolean): void {
    this.apiKeyPermissions.update((current) => {
      const existing = current[tableName] ? [...current[tableName]] : [];
      const next = checked
        ? existing.includes(action)
          ? existing
          : [...existing, action]
        : existing.filter((a) => a !== action);
      return {
        ...current,
        [tableName]: next,
      };
    });
  }

  isAllSelectedForContentType(tableName: string): boolean {
    const perms = this.apiKeyPermissions()[tableName];
    return Array.isArray(perms) && perms.length === this.allAPIKeyActions.length;
  }

  toggleAllForContentType(tableName: string, checked: boolean): void {
    this.apiKeyPermissions.update((current) => ({
      ...current,
      [tableName]: checked ? this.allAPIKeyActions.map((a) => a.value) : [],
    }));
  }

  selectAllPermissionsMatrix(): void {
    const next: Record<string, APIKeyAction[]> = {};
    for (const ct of this.projectSchemas()) {
      next[ct.tableName] = this.allAPIKeyActions.map((a) => a.value);
    }
    this.apiKeyPermissions.set(next);
  }

  selectReadOnlyPermissionsMatrix(): void {
    const next: Record<string, APIKeyAction[]> = {};
    for (const ct of this.projectSchemas()) {
      next[ct.tableName] = ['find', 'findOne'];
    }
    this.apiKeyPermissions.set(next);
  }

  clearAllPermissionsMatrix(): void {
    const next: Record<string, APIKeyAction[]> = {};
    for (const ct of this.projectSchemas()) {
      next[ct.tableName] = [];
    }
    this.apiKeyPermissions.set(next);
  }

  submitAPIKeyForm(): void {
    const project = this.selectedProjectForKeys();
    const name = this.apiKeyName.trim();
    if (!project || !name) {
      this.msg.error('El nombre de la API Key es obligatorio');
      return;
    }

    const payload = {
      name,
      description: this.apiKeyDescription.trim() || undefined,
      accessType: this.apiKeyAccessType,
      permissions: this.apiKeyAccessType === 'custom' ? this.apiKeyPermissions() : {},
    };

    this.savingAPIKey.set(true);
    const editing = this.editingAPIKey();

    if (editing) {
      this.projectsService.updateAPIKey(project.slug, editing.id, payload).subscribe({
        next: (updated) => {
          this.savingAPIKey.set(false);
          this.apiKeyFormVisible.set(false);
          this.editingAPIKey.set(null);
          this.projectAPIKeys.update((list) =>
            list.map((k) => (k.id === updated.id ? updated : k)),
          );
          this.msg.success(`API Key "${updated.name}" actualizada`);
        },
        error: (err) => {
          this.savingAPIKey.set(false);
          this.msg.error(err.error?.error || 'Error al actualizar la API Key');
        },
      });
    } else {
      this.projectsService.createAPIKey(project.slug, payload).subscribe({
        next: (created) => {
          this.savingAPIKey.set(false);
          this.apiKeyFormVisible.set(false);
          if (created.token) {
            this.newlyCreatedToken.set(created.token);
          }
          this.projectAPIKeys.update((list) => [created, ...list]);
          this.msg.success(`API Key "${created.name}" generada exitosamente`);
        },
        error: (err) => {
          this.savingAPIKey.set(false);
          this.msg.error(err.error?.error || 'Error al generar la API Key');
        },
      });
    }
  }

  revokeAPIKey(key: APIKey): void {
    const project = this.selectedProjectForKeys();
    if (!project) {
      return;
    }

    this.projectsService.deleteAPIKey(project.slug, key.id).subscribe({
      next: () => {
        this.projectAPIKeys.update((list) => list.filter((k) => k.id !== key.id));
        this.msg.success(`API Key "${key.name}" revocada permanentemente`);
      },
      error: (err) => {
        this.msg.error(err.error?.error || 'Error al revocar la API Key');
      },
    });
  }

  copyTokenToClipboard(token: string): void {
    navigator.clipboard.writeText(token).then(
      () => this.msg.success('Token copiado al portapapeles'),
      () => this.msg.info('Copia manualmente el token mostrado en pantalla'),
    );
  }

  getAccessTypeLabel(type: APIKeyAccessType): string {
    switch (type) {
      case 'full-access':
        return 'Acceso Total (Full Access)';
      case 'read-only':
        return 'Solo Lectura (Read-Only)';
      default:
        return 'Personalizado (Granular)';
    }
  }

  getAccessTypeColor(type: APIKeyAccessType): string {
    switch (type) {
      case 'full-access':
        return 'purple';
      case 'read-only':
        return 'cyan';
      default:
        return 'geekblue';
    }
  }

  getPermissionsSummary(key: APIKey): { table: string; actions: APIKeyAction[] }[] {
    if (!key.permissions) {
      return [];
    }
    return Object.entries(key.permissions)
      .filter(([, actions]) => Array.isArray(actions) && actions.length > 0)
      .map(([table, actions]) => ({ table, actions }));
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
}
