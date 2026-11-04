import { registerLocaleData } from '@angular/common';
import { provideHttpClient, withFetch, withInterceptors } from '@angular/common/http';
import es from '@angular/common/locales/es';
import { ApplicationConfig, LOCALE_ID, provideBrowserGlobalErrorListeners } from '@angular/core';
import { provideRouter, withComponentInputBinding } from '@angular/router';
import {
  AppstoreOutline,
  BuildOutline,
  CheckOutline,
  CheckSquareOutline,
  ControlOutline,
  CopyOutline,
  CrownOutline,
  DatabaseOutline,
  DeleteOutline,
  DownOutline,
  EditOutline,
  EyeInvisibleOutline,
  EyeOutline,
  FileExcelOutline,
  FileOutline,
  FilePdfOutline,
  FileTextOutline,
  FileWordOutline,
  FileZipOutline,
  FolderAddOutline,
  FolderFill,
  FolderOutline,
  FontSizeOutline,
  HomeOutline,
  InboxOutline,
  KeyOutline,
  LockOutline,
  LogoutOutline,
  MailOutline,
  MenuFoldOutline,
  MenuUnfoldOutline,
  MoreOutline,
  NumberOutline,
  PictureOutline,
  PlusOutline,
  SafetyCertificateOutline,
  SaveOutline,
  SettingOutline,
  SyncOutline,
  TeamOutline,
  ThunderboltOutline,
  UploadOutline,
  UserAddOutline,
  UserOutline,
} from '@ant-design/icons-angular/icons';
import { provideNzDateFnsAdapter } from 'ng-zorro-antd/core/time';
import { es_ES, provideNzI18n } from 'ng-zorro-antd/i18n';
import { provideNzIcons } from 'ng-zorro-antd/icon';

import { routes } from './app.routes';
import { authInterceptor } from './core/auth/auth.interceptor';
import { projectInterceptor } from './core/project/project.interceptor';

registerLocaleData(es);

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    // Dates and numbers in Spanish, matching the NG-ZORRO locale below.
    { provide: LOCALE_ID, useValue: 'es' },
    provideRouter(routes, withComponentInputBinding()),
    provideHttpClient(withFetch(), withInterceptors([authInterceptor, projectInterceptor])),
    provideNzI18n(es_ES),
    provideNzDateFnsAdapter(),
    // Icons are registered one by one so only the ones used end up in the bundle.
    provideNzIcons([
      AppstoreOutline,
      BuildOutline,
      CheckOutline,
      CheckSquareOutline,
      ControlOutline,
      CopyOutline,
      CrownOutline,
      DatabaseOutline,
      DeleteOutline,
      DownOutline,
      EditOutline,
      EyeInvisibleOutline,
      EyeOutline,
      FileExcelOutline,
      FileOutline,
      FilePdfOutline,
      FileTextOutline,
      FileWordOutline,
      FileZipOutline,
      FolderAddOutline,
      FolderFill,
      FolderOutline,
      FontSizeOutline,
      HomeOutline,
      InboxOutline,
      KeyOutline,
      LockOutline,
      LogoutOutline,
      MailOutline,
      MenuFoldOutline,
      MenuUnfoldOutline,
      MoreOutline,
      NumberOutline,
      PictureOutline,
      PlusOutline,
      SafetyCertificateOutline,
      SaveOutline,
      SettingOutline,
      SyncOutline,
      TeamOutline,
      ThunderboltOutline,
      UploadOutline,
      UserAddOutline,
      UserOutline,
    ]),
  ],
};
