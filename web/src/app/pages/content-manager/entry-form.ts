import { AbstractControl, FormArray, FormControl, FormGroup, ValidatorFn, Validators } from '@angular/forms';

import { ComponentDefinition, getComponentUID } from '../../core/content/components.service';
import { Attribute, ContentType } from '../../core/content/content-types.service';
import { Entry, FieldValue, WriteValue } from '../../core/content/content.service';
import { MediaRef } from '../../core/media/media.service';

/** A form control's value: text, number, switch, media id, or component data. */
export type FormValue = string | number | boolean | Record<string, any> | Array<Record<string, any>> | null;
export type EntryForm = FormGroup<Record<string, AbstractControl>>;

export function isMediaRef(value: FieldValue | undefined): value is MediaRef {
  return typeof value === 'object' && value !== null && 'url' in value;
}

/** One control per attribute, prefilled from `entry` when editing. */
export function buildEntryForm(
  ct: ContentType,
  componentsOrEntry?: ComponentDefinition[] | Entry | null,
  entry?: Entry | null,
): EntryForm {
  let components: ComponentDefinition[] = [];
  let existingEntry: Entry | null | undefined = entry;

  if (Array.isArray(componentsOrEntry)) {
    components = componentsOrEntry;
  } else if (componentsOrEntry !== undefined) {
    existingEntry = componentsOrEntry;
  }

  const controls: Record<string, AbstractControl> = {};

  for (const attr of ct.attributes) {
    if (attr.type === 'component') {
      const comp = components.find((c) => getComponentUID(c) === attr.component);
      const val = existingEntry?.[attr.name];

      if (attr.repeatable) {
        const arr = Array.isArray(val) ? val : [];
        const groups = comp ? arr.map((item) => buildComponentGroup(comp, item)) : [];
        controls[attr.name] = new FormArray(groups);
      } else {
        const initVal = typeof val === 'object' && val !== null ? (val as Record<string, any>) : undefined;
        controls[attr.name] = comp ? buildComponentGroup(comp, initVal) : new FormControl(null);
      }
    } else {
      controls[attr.name] = new FormControl<FormValue>(initialValue(attr, existingEntry?.[attr.name]), validatorsFor(attr));
    }
  }

  return new FormGroup(controls);
}

export function buildComponentGroup(comp: ComponentDefinition, initial?: Record<string, any>): FormGroup {
  const controls: Record<string, AbstractControl> = {};

  for (const attr of comp.attributes) {
    controls[attr.name] = new FormControl<FormValue>(initialValue(attr, initial?.[attr.name]), validatorsFor(attr));
  }

  return new FormGroup(controls);
}

function initialValue(attr: Attribute, value: FieldValue | undefined): FormValue {
  switch (attr.type) {
    case 'string':
      return typeof value === 'string' ? value : '';
    case 'integer':
      return typeof value === 'number' ? value : null;
    case 'boolean':
      return typeof value === 'boolean' ? value : false;
    case 'media':
      return isMediaRef(value) ? value.id : null;
    case 'component':
      return null;
  }
}

function validatorsFor(attr: Attribute): ValidatorFn[] {
  // A switch always has a value, so "required" never blocks it.
  return attr.required && attr.type !== 'boolean' ? [Validators.required] : [];
}

/**
 * The `data` object for POST/PUT. An empty optional text field is sent as null rather than "", so
 * "not filled in" is stored as no value.
 */
export function toPayload(ct: ContentType, value: Record<string, any>): Record<string, WriteValue> {
  const data: Record<string, WriteValue> = {};

  for (const attr of ct.attributes) {
    const v = value[attr.name];

    if (attr.type === 'component') {
      if (attr.repeatable) {
        data[attr.name] = Array.isArray(v) ? v : [];
      } else {
        data[attr.name] = v ?? null;
      }
    } else if (attr.type === 'string' && v === '' && !attr.required) {
      data[attr.name] = null;
    } else {
      data[attr.name] = v ?? null;
    }
  }

  return data;
}
