import { FormArray, FormGroup } from '@angular/forms';

import { ComponentDefinition } from '../../core/content/components.service';
import { ContentType } from '../../core/content/content-types.service';
import { Entry, entryQueryParams } from '../../core/content/content.service';
import { buildEntryForm, toPayload } from './entry-form';

const posts: ContentType = {
  tableName: 'posts',
  attributes: [
    { name: 'title', type: 'string', required: true },
    { name: 'summary', type: 'string', required: false },
    { name: 'views', type: 'integer', required: false },
    { name: 'published', type: 'boolean', required: true },
    { name: 'cover', type: 'media', required: false },
  ],
};

const authorComp: ComponentDefinition = {
  name: 'author',
  category: 'blog',
  displayName: 'Author',
  attributes: [
    { name: 'name', type: 'string', required: true },
    { name: 'bio', type: 'string', required: false },
  ],
};

const tagComp: ComponentDefinition = {
  name: 'tag',
  category: 'blog',
  displayName: 'Tag',
  attributes: [{ name: 'label', type: 'string', required: true }],
};

const articleWithComponents: ContentType = {
  tableName: 'articles',
  attributes: [
    { name: 'title', type: 'string', required: true },
    { name: 'author', type: 'component', component: 'blog.author', repeatable: false, required: false },
    { name: 'tags', type: 'component', component: 'blog.tag', repeatable: true, required: false },
  ],
};

const fileId = '550e8400-e29b-41d4-a716-446655440000';

describe('entry form', () => {
  it('starts empty for a new entry, with required validators except on switches', () => {
    const form = buildEntryForm(posts);

    expect(form.getRawValue()).toEqual({ title: '', summary: '', views: null, published: false, cover: null });
    expect(form.controls['title'].hasError('required')).toBe(true);
    expect(form.controls['published'].valid).toBe(true);
    expect(form.valid).toBe(false);
  });

  it('prefills from an entry, taking the nested media file as its id', () => {
    const entry: Entry = {
      id: 'e1',
      createdAt: '',
      updatedAt: '',
      title: 'Hola',
      summary: null,
      views: 7,
      published: true,
      cover: { id: fileId, name: 'c.png', url: '/uploads/c.png', mime: 'image/png', ext: '.png' },
    };

    expect(buildEntryForm(posts, entry).getRawValue()).toEqual({
      title: 'Hola',
      summary: '',
      views: 7,
      published: true,
      cover: fileId,
    });
  });

  it('sends empty optional text as null and media as the file id', () => {
    expect(
      toPayload(posts, { title: 'Hola', summary: '', views: null, published: false, cover: fileId }),
    ).toEqual({ title: 'Hola', summary: null, views: null, published: false, cover: fileId });
  });

  it('builds nested FormGroup for single component and FormArray for repeatable component', () => {
    const components = [authorComp, tagComp];
    const form = buildEntryForm(articleWithComponents, components);

    expect(form.controls['author'] instanceof FormGroup).toBe(true);
    expect(form.controls['tags'] instanceof FormArray).toBe(true);

    const authorGroup = form.controls['author'] as FormGroup;
    expect(authorGroup.getRawValue()).toEqual({ name: '', bio: '' });
    expect(authorGroup.controls['name'].hasError('required')).toBe(true);

    const tagsArray = form.controls['tags'] as FormArray;
    expect(tagsArray.length).toBe(0);
  });

  it('prefills component values from an existing entry and handles toPayload', () => {
    const components = [authorComp, tagComp];
    const entry: Entry = {
      id: 'a1',
      createdAt: '',
      updatedAt: '',
      title: 'Clean Architecture',
      author: { name: 'Uncle Bob', bio: 'Author of Clean Code' },
      tags: [{ label: 'Architecture' }, { label: 'Go' }],
    };

    const form = buildEntryForm(articleWithComponents, components, entry);
    expect(form.getRawValue()).toEqual({
      title: 'Clean Architecture',
      author: { name: 'Uncle Bob', bio: 'Author of Clean Code' },
      tags: [{ label: 'Architecture' }, { label: 'Go' }],
    });

    const payload = toPayload(articleWithComponents, form.getRawValue());
    expect(payload).toEqual({
      title: 'Clean Architecture',
      author: { name: 'Uncle Bob', bio: 'Author of Clean Code' },
      tags: [{ label: 'Architecture' }, { label: 'Go' }],
    });
  });
});

describe('entryQueryParams', () => {
  it('builds Strapi-style pagination and sort', () => {
    const params = entryQueryParams({ page: 3, pageSize: 20, sort: [{ field: 'createdAt', desc: true }] });

    expect(params.get('pagination[page]')).toBe('3');
    expect(params.get('pagination[pageSize]')).toBe('20');
    expect(params.get('sort')).toBe('createdAt:desc');
  });

  it('omits sort when there is none', () => {
    expect(entryQueryParams({ page: 1, pageSize: 10, sort: [] }).has('sort')).toBe(false);
  });
});
