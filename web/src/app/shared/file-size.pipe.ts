import { Pipe, PipeTransform } from '@angular/core';

const UNITS = ['B', 'KB', 'MB', 'GB'];
const STEP = 1024;

/** Formats a byte count for people: 52341 → "51,1 KB" (in the app's locale). */
@Pipe({ name: 'fileSize' })
export class FileSizePipe implements PipeTransform {
  transform(bytes: number | null | undefined): string {
    if (bytes == null || bytes < 0) {
      return '';
    }

    let value = bytes;
    let unit = 0;

    while (value >= STEP && unit < UNITS.length - 1) {
      value /= STEP;
      unit++;
    }

    const digits = unit === 0 || value >= 100 ? 0 : 1;

    return `${value.toLocaleString('es', { maximumFractionDigits: digits })} ${UNITS[unit]}`;
  }
}
