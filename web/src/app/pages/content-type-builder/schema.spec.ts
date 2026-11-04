import { FormArray, FormControl, FormGroup } from '@angular/forms';

import {
  IDENTIFIER_PATTERN,
  formatFieldLabel,
  getFieldDisplayName,
  notReserved,
  toJsonSchema,
  uniqueNames,
} from './schema';

describe('builder schema helpers', () => {
  it.each(['articles', 'blog_posts', 'a1'])('accepts identifier %s', (name) => {
    expect(IDENTIFIER_PATTERN.test(name)).toBe(true);
  });

  it.each(['Articles', '1st', 'my field', 'a-b', ''])('rejects identifier %s', (name) => {
    expect(IDENTIFIER_PATTERN.test(name)).toBe(false);
  });

  it('rejects system column names', () => {
    expect(notReserved(new FormControl('created_at'))).toEqual({ reserved: true });
    expect(notReserved(new FormControl('title'))).toBeNull();
  });

  it('marks duplicated attribute names and clears the mark once fixed', () => {
    const row = (name: string) => new FormGroup({ name: new FormControl(name) });
    const attrs = new FormArray([row('title'), row('title'), row('body')], [uniqueNames]);

    expect(attrs.errors).toEqual({ duplicateNames: true });
    expect(attrs.at(1).get('name')?.errors).toEqual({ duplicate: true });
    expect(attrs.at(2).get('name')?.errors).toBeNull();

    attrs.at(1).get('name')?.setValue('summary');
    attrs.updateValueAndValidity();

    expect(attrs.errors).toBeNull();
    expect(attrs.at(1).get('name')?.errors).toBeNull();
  });

  it('formats field names and handles display names properly', () => {
    expect(formatFieldLabel('direccion_maps')).toBe('Direccion Maps');
    expect(formatFieldLabel('nombre')).toBe('Nombre');
    expect(formatFieldLabel('is_published')).toBe('Is Published');

    expect(getFieldDisplayName({ name: 'direccion_maps' })).toBe('Direccion Maps');
    expect(getFieldDisplayName({ name: 'direccion_maps', displayName: 'Dirección en Google Maps' })).toBe('Dirección en Google Maps');
    expect(getFieldDisplayName({ name: 'nombre', displayName: '' })).toBe('Nombre');
  });

  it('previews the JSON Schema the server stores, media included', () => {
    expect(
      toJsonSchema({
        tableName: 'posts',
        attributes: [
          { name: 'title', displayName: 'Título Principal', type: 'string', required: true },
          { name: 'views', type: 'integer', required: false },
          { name: 'cover', type: 'media', required: false },
          { name: 'contact', type: 'component', component: 'shared.contact_data', repeatable: false, required: false },
        ],
      }),
    ).toEqual({
      $schema: 'https://json-schema.org/draft/2020-12/schema',
      title: 'posts',
      type: 'object',
      properties: {
        title: { title: 'Título Principal', type: 'string' },
        views: { type: 'integer' },
        cover: { type: 'string', format: 'uuid', 'x-cms-type': 'media' },
        contact: { type: 'object', 'x-cms-type': 'component', 'x-component': 'shared.contact_data', 'x-repeatable': false },
      },
      required: ['title'],
      additionalProperties: false,
    });
  });
});
