import { AbstractControl, FormArray, FormGroup } from '@angular/forms';

/**
 * Shows the errors of every field, including ones the user never touched. nz-form-control repaints
 * on statusChanges, which markAllAsDirty() alone does not emit, so each control is also revalidated.
 */
export function showFormErrors(control: AbstractControl): void {
  if (control instanceof FormGroup || control instanceof FormArray) {
    Object.values(control.controls).forEach((child: AbstractControl) => showFormErrors(child));
  }

  control.markAsDirty();
  control.updateValueAndValidity({ onlySelf: true });
}
