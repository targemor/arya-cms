import { HttpErrorResponse, HttpStatusCode } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject, viewChild } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { ActivatedRoute, Router } from '@angular/router';
import { NzIconModule } from 'ng-zorro-antd/icon';
import { NzMessageService } from 'ng-zorro-antd/message';
import { NzUploadChangeParam, NzUploadFile, NzUploadModule } from 'ng-zorro-antd/upload';
import { map } from 'rxjs';

import { ApiError } from '../../core/api';
import { MAX_UPLOAD_SIZE_MB, UPLOAD_URL } from '../../core/media/media.service';
import { MediaBrowser } from '../../shared/media-browser/media-browser';

const BYTES_PER_KB = 1024;

@Component({
  selector: 'app-media-library',
  imports: [MediaBrowser, NzIconModule, NzUploadModule],
  templateUrl: './media-library.html',
  styleUrl: './media-library.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MediaLibrary {
  private readonly message = inject(NzMessageService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  private readonly browser = viewChild.required(MediaBrowser);

  /** Open folder, kept in the URL (?folder=<id>) so it can be linked and survives a reload. */
  protected readonly folderId = toSignal(this.route.queryParamMap.pipe(map((p) => p.get('folder'))), {
    initialValue: null,
  });

  /** Sent with each upload so files land in the open folder. */
  protected readonly uploadData = (): Record<string, string> => {
    const folderId = this.folderId();

    return folderId ? { folderId } : {};
  };

  // The JWT is added by authInterceptor: nz-upload sends through HttpClient.
  protected readonly uploadUrl = UPLOAD_URL;
  protected readonly maxSizeMb = MAX_UPLOAD_SIZE_MB;

  /**
   * Refuses files over the server limit before sending them, saying why. (nz-upload's own nzSize
   * filter drops them silently.) An arrow function so `this` survives being passed as an input.
   */
  protected readonly beforeUpload = (file: NzUploadFile): boolean => {
    const sizeMb = (file.size ?? 0) / BYTES_PER_KB / BYTES_PER_KB;
    if (sizeMb > MAX_UPLOAD_SIZE_MB) {
      this.message.error(`«${file.name}» supera ${MAX_UPLOAD_SIZE_MB} MB.`);

      return false;
    }

    return true;
  };

  protected openFolder(folderId: string | null): void {
    void this.router.navigate([], {
      relativeTo: this.route,
      queryParams: { folder: folderId },
    });
  }

  protected onChange({ file, type }: NzUploadChangeParam): void {
    if (type === 'success') {
      this.message.success(`«${file.name}» subido`);
      this.browser().reload();
    } else if (type === 'error') {
      this.message.error(`No se pudo subir «${file.name}»: ${uploadErrorMessage(file)}`);
    }
  }
}

function uploadErrorMessage(file: NzUploadFile): string {
  const err: unknown = file.error;

  if (err instanceof HttpErrorResponse) {
    if (err.status === HttpStatusCode.PayloadTooLarge) {
      return `supera ${MAX_UPLOAD_SIZE_MB} MB.`;
    }

    const body = err.error as Partial<ApiError> | null;
    if (body?.error) {
      return body.error;
    }
  }

  return 'error inesperado.';
}
