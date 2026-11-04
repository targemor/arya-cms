import { HttpErrorResponse, HttpStatusCode } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { NonNullableFormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { NzAlertModule } from 'ng-zorro-antd/alert';
import { NzButtonModule } from 'ng-zorro-antd/button';
import { NzCardModule } from 'ng-zorro-antd/card';
import { NzFormModule } from 'ng-zorro-antd/form';
import { NzIconModule } from 'ng-zorro-antd/icon';
import { NzInputModule } from 'ng-zorro-antd/input';
import { finalize } from 'rxjs';

import { safeReturnUrl } from '../../core/auth/auth.guard';
import { AuthService } from '../../core/auth/auth.service';
import { showFormErrors } from '../../shared/forms';

@Component({
  selector: 'app-login',
  imports: [
    ReactiveFormsModule,
    NzAlertModule,
    NzButtonModule,
    NzCardModule,
    NzFormModule,
    NzIconModule,
    NzInputModule,
  ],
  templateUrl: './login.html',
  styleUrl: './login.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Login {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  protected readonly form = inject(NonNullableFormBuilder).group({
    email: ['', [Validators.required, Validators.email]],
    password: ['', [Validators.required]],
  });

  protected readonly submitting = signal(false);
  protected readonly error = signal<string | null>(null);

  protected submit(): void {
    if (this.form.invalid) {
      showFormErrors(this.form);

      return;
    }

    const { email, password } = this.form.getRawValue();

    this.submitting.set(true);
    this.error.set(null);

    this.auth
      .login(email, password)
      .pipe(finalize(() => this.submitting.set(false)))
      .subscribe({
        next: () => {
          const returnUrl = this.route.snapshot.queryParamMap.get('returnUrl');
          void this.router.navigateByUrl(safeReturnUrl(returnUrl));
        },
        error: (err: unknown) => this.error.set(loginErrorMessage(err)),
      });
  }
}

function loginErrorMessage(err: unknown): string {
  if (err instanceof HttpErrorResponse) {
    if (err.status === HttpStatusCode.Unauthorized) {
      return 'Email o contraseña incorrectos.';
    }

    if (err.status === 0) {
      return 'No se pudo conectar con el servidor.';
    }
  }

  return 'No se pudo iniciar sesión. Inténtalo de nuevo.';
}
