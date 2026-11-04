import { AbstractControl, FormArray, ValidationErrors, ValidatorFn } from '@angular/forms';

import { AttributeType, ContentType } from '../../core/content/content-types.service';

/** Same rules the API applies: lowercase snake_case starting with a letter, at most 48 characters. */
export const IDENTIFIER_PATTERN = /^[a-z][a-z0-9_]*$/;
export const IDENTIFIER_MAX_LENGTH = 48;

/** Every content table already has these columns. */
const RESERVED_NAMES = new Set(['id', 'created_at', 'updated_at']);

export const ATTRIBUTE_TYPES: ReadonlyArray<{
  value: AttributeType;
  label: string;
  icon: string;
  description: string;
  color: string;
}> = [
  { value: 'string', label: 'Texto', icon: 'font-size', description: 'Títulos, nombres, párrafos...', color: '#4945ff' },
  { value: 'integer', label: 'Número', icon: 'number', description: 'Enteros, contadores, cantidades...', color: '#f59e0b' },
  { value: 'boolean', label: 'Booleano', icon: 'check-square', description: 'Verdadero o falso', color: '#10b981' },
  { value: 'media', label: 'Media', icon: 'picture', description: 'Imágenes, videos, archivos', color: '#8b5cf6' },
  { value: 'component', label: 'Componente', icon: 'appstore', description: 'Estructura reutilizable de campos', color: '#ec4899' },
];

export const notReserved: ValidatorFn = (control: AbstractControl): ValidationErrors | null =>
  RESERVED_NAMES.has(control.value as string) ? { reserved: true } : null;

/** On the attributes FormArray: marks a duplicated name on the controls that repeat it. */
export const uniqueNames: ValidatorFn = (control: AbstractControl): ValidationErrors | null => {
  const names = (control as FormArray).controls.map((group) => group.get('name'));
  const seen = new Map<string, number>();

  for (const name of names) {
    const value = (name?.value as string) ?? '';
    seen.set(value, (seen.get(value) ?? 0) + 1);
  }

  let duplicated = false;

  for (const name of names) {
    if (!name) {
      continue;
    }

    const isDuplicate = name.value !== '' && (seen.get(name.value as string) ?? 0) > 1;
    duplicated ||= isDuplicate;

    // Keep the control's other errors; add or remove only "duplicate".
    const { duplicate: _, ...others } = name.errors ?? {};
    const errors = isDuplicate ? { ...others, duplicate: true } : others;
    name.setErrors(Object.keys(errors).length > 0 ? errors : null, { emitEvent: false });
  }

  return duplicated ? { duplicateNames: true } : null;
};

/**
 * Formats a snake_case attribute name into a clean, human-readable Title Case label
 * e.g. "direccion_maps" -> "Direccion Maps", "nombre" -> "Nombre".
 */
export function formatFieldLabel(name: string): string {
  if (!name) {
    return '';
  }

  return name
    .split('_')
    .filter(Boolean)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}

/**
 * Returns the attribute's display name if provided, or converts its technical name to Title Case.
 */
export function getFieldDisplayName(attr: { name: string; displayName?: string }): string {
  if (attr.displayName && attr.displayName.trim() !== '') {
    return attr.displayName.trim();
  }

  return formatFieldLabel(attr.name);
}

/**
 * The JSON Schema the API stores for a content type (schemas/<tableName>.json), for the builder's
 * preview. It mirrors the server's format; the server is what actually writes the file.
 */
export function toJsonSchema(ct: ContentType): object {
  const properties: Record<string, object> = {};

  for (const attr of ct.attributes ?? []) {
    if (!attr.name) {
      continue;
    }

    const titleProp = attr.displayName ? { title: attr.displayName } : {};

    if (attr.type === 'media') {
      properties[attr.name] = { ...titleProp, type: 'string', format: 'uuid', 'x-cms-type': 'media' };
    } else if (attr.type === 'component') {
      properties[attr.name] = attr.repeatable
        ? {
            ...titleProp,
            type: 'array',
            'x-cms-type': 'component',
            'x-component': attr.component,
            'x-repeatable': true,
            items: { type: 'object' },
          }
        : {
            ...titleProp,
            type: 'object',
            'x-cms-type': 'component',
            'x-component': attr.component,
            'x-repeatable': false,
          };
    } else {
      properties[attr.name] = { ...titleProp, type: attr.type };
    }
  }

  const required = (ct.attributes ?? []).filter((a) => a.required && a.name).map((a) => a.name);

  return {
    $schema: 'https://json-schema.org/draft/2020-12/schema',
    title: ct.tableName,
    ...(ct.displayName ? { 'x-display-name': ct.displayName } : {}),
    ...(ct.kind ? { 'x-cms-kind': ct.kind } : {}),
    type: 'object',
    properties,
    ...(required.length > 0 ? { required } : {}),
    additionalProperties: false,
  };
}
