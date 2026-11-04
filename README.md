# Arya CMS

**Arya CMS** es un CMS Headless moderno, multi-tenant y de alto rendimiento construido con **Go** ([Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) + [Fiber](https://github.com/gofiber/fiber)) y un panel de administración en **Angular** ([NG-ZORRO](https://ng.ant.design/)).

[![Licencia](https://img.shields.io/badge/Licencia-MIT-success)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/lucimux/arya)](https://goreportcard.com/report/github.com/lucimux/arya)
[![Web Framework](https://img.shields.io/badge/Fiber-Web%20Framework-blue)](https://github.com/gofiber/fiber)
[![Documentación API](https://img.shields.io/badge/Swagger-Documentación%20API-blue)](https://github.com/swaggo/swag)
[![Validación](https://img.shields.io/badge/Validator-Integridad%20de%20Datos-blue)](https://github.com/go-playground/validator)
[![Base de Datos](https://img.shields.io/badge/PostgreSQL-Multi--Tenant-blue)](https://www.postgresql.org/)
[![Migraciones](https://img.shields.io/badge/Migraciones-golang--migrate-blue)](https://github.com/golang-migrate/migrate)
[![Trazabilidad](https://img.shields.io/badge/OpenTelemetry-Trazabilidad%20Distribuida-blue)](https://opentelemetry.io/)
[![Frontend](https://img.shields.io/badge/Angular%2019-NG--ZORRO-red)](https://angular.dev)

---

## Descripción General

Arya CMS provee un motor completo de gestión de contenidos headless inspirado en Strapi, diseñado bajo los principios de **Clean Architecture** (Arquitectura Limpia) de Robert C. Martin ("Uncle Bob"). Cuenta con aislamiento multi-proyecto (multi-tenant), modelado dinámico de tipos de contenido en tiempo de ejecución, llaves de API con permisos granulares para consumo REST y biblioteca multimedia.

### Características Principales

- **Proyectos Multi-Tenant**: Esquemas y tablas físicas completamente aisladas por cada proyecto (`content_<slugProyecto>_<nombreTabla>`).
- **Super Administrador y Roles (RBAC)**: Niveles de acceso globales (`superadmin`, `admin`, `editor`, `viewer`) y asignación de roles por miembro en cada proyecto (`admin`, `editor`, `viewer`).
- **Gestión de API Keys con Permisos Granulares**: Creación de llaves por proyecto con matrices detalladas de acciones permitidas (`find`, `findOne`, `create`, `update`, `delete`) por cada tipo de contenido.
- **Content-Type Builder en Tiempo Real**: Creación de tipos de contenido (colecciones o tipos únicos) que generan automáticamente esquemas JSON Schema y tablas en PostgreSQL sin reiniciar el servidor.
- **Content API con Consultas Avanzadas**: CRUD completo vía REST con filtros estilo Strapi (`$eq`, `$ne`, `$contains`, `$gte`, etc.), ordenamiento múltiple y paginación.
- **Componentes y Campos Repetibles**: Sub-estructuras reutilizables incrustables dentro de los tipos de contenido.
- **Media Library (Biblioteca Multimedia)**: Gestión jerárquica de carpetas y archivos con vista previa de imágenes, validación de tipos MIME y entrega segura de archivos.
- **Observabilidad con OpenTelemetry**: Trazabilidad distribuida de extremo a extremo que abarca controladores HTTP, casos de uso y repositorios.

---

## Dominios y Módulos del Sistema

### 1. Proyectos Multi-Tenant y Panel de Super Admin
Cada proyecto funciona como un tenant independiente con sus propios esquemas y tablas físicas en PostgreSQL.

| Operación | Método / Ruta | Nivel de Acceso Requerido |
|---|---|---|
| Listar proyectos | `GET /v1/projects` | Autenticado |
| Obtener proyecto por slug | `GET /v1/projects/:slug` | Autenticado |
| Crear nuevo proyecto | `POST /v1/projects` | Super Administrador |
| Actualizar proyecto | `PUT /v1/projects/:slug` | Super Administrador |
| Eliminar proyecto | `DELETE /v1/projects/:slug` | Super Administrador |
| Listar miembros del proyecto | `GET /v1/projects/:slug/members` | Super Admin / Admin |
| Asignar o actualizar rol de miembro | `POST /v1/projects/:slug/members` | Super Admin / Admin |
| Desvincular miembro del proyecto | `DELETE /v1/projects/:slug/members/:userId` | Super Admin / Admin |

### 2. API Keys y Permisos Granulares REST
Permite generar tokens seguros con prefijo (`cms_<hex>`) vinculados a un proyecto específico para su consumo vía aplicaciones externas o frontends públicos/privados.

| Operación | Método / Ruta | Nivel de Acceso Requerido |
|---|---|---|
| Listar API Keys del proyecto | `GET /v1/projects/:slug/api-keys` | Super Admin / Admin |
| Generar nueva API Key | `POST /v1/projects/:slug/api-keys` | Super Admin / Admin |
| Actualizar permisos de API Key | `PUT /v1/projects/:slug/api-keys/:id` | Super Admin / Admin |
| Revocar permanentemente API Key | `DELETE /v1/projects/:slug/api-keys/:id` | Super Admin / Admin |

#### Consumo de la API con API Key
Puedes enviar el token mediante la cabecera `X-API-Key` o la cabecera estándar `Authorization: Bearer`:
```bash
curl -H "X-API-Key: cms_xxxxxxxxxxxxxxxx" http://localhost:8080/v1/content/articulos
```
El sistema detecta automáticamente el proyecto asociado a la clave y verifica que tenga permiso específico para la acción solicitada (`find`, `findOne`, `create`, `update`, `delete`).

### 3. Autenticación y Control de Acceso Global
Autenticación mediante JWT, contraseñas hasheadas con bcrypt y validación de roles en middleware.

| Operación | Método / Ruta | Nivel de Acceso Requerido |
|---|---|---|
| Registrar usuario | `POST /v1/auth/register` | Público |
| Iniciar sesión | `POST /v1/auth/login` | Público |
| Perfil del usuario activo | `GET /v1/user/profile` | Autenticado |
| Listar todos los usuarios | `GET /v1/users` | Super Administrador |
| Cambiar rol global de un usuario | `PUT /v1/users/:id/role` | Super Administrador |

### 4. Content-Type Builder y Esquemas
Define modelos de datos dinámicamente. Cada petición se contextualiza al proyecto activo mediante la cabecera `X-Project: <slug>` (por defecto `default`).

| Operación | Método / Ruta |
|---|---|
| Listar esquemas del proyecto | `GET /v1/internal/schemas` |
| Crear tipo de contenido | `POST /v1/internal/schemas` |
| Modificar tipo de contenido | `PUT /v1/internal/schemas/:tableName` |

- Los esquemas se guardan como archivos JSON Schema en `schemas/<slugProyecto>/<nombreTabla>.json`.
- Ejecuta automáticamente `CREATE TABLE IF NOT EXISTS content_<slugProyecto>_<nombreTabla>` y agrega columnas con `ADD COLUMN IF NOT EXISTS`.
- Atributos admitidos: `string`, `integer`, `boolean`, `media` (referencia foránea a `upload_files`) y `component`.

### 5. Content API (Consumo y Gestión de Contenido)
Rutas REST para interactuar con las entradas de cualquier tipo de contenido creado.

| Operación | Método / Ruta | Acción Requerida |
|---|---|---|
| Listar entradas (Colecciones) | `GET /v1/content/:tableName` | `find` |
| Obtener entrada por ID | `GET /v1/content/:tableName/:id` | `findOne` |
| Obtener entrada de Tipo Único (Single) | `GET /v1/content/:tableName/single` | `findOne` |
| Crear entrada | `POST /v1/content/:tableName` | `create` |
| Actualizar entrada por ID | `PUT /v1/content/:tableName/:id` | `update` |
| Actualizar Tipo Único | `PUT /v1/content/:tableName/single` | `update` |
| Eliminar entrada | `DELETE /v1/content/:tableName/:id` | `delete` |

#### Filtros, Ordenación y Paginación
```bash
GET /v1/content/articulos?pagination[page]=1&pagination[pageSize]=10&sort=createdAt:desc&filters[titulo][$containsi]=tecnologia&filters[visitas][$gte]=50
```
Operadores compatibles: `$eq`, `$ne`, `$lt`, `$lte`, `$gt`, `$gte`, `$in`, `$notIn`, `$contains`, `$containsi`, `$startsWith`, `$endsWith`, `$null`, `$notNull`.

### 6. Media Library (Biblioteca de Medios)
Carga y organiza archivos y carpetas con generación de enlaces públicos seguros.

| Operación | Método / Ruta |
|---|---|
| Subir archivo | `POST /v1/upload` |
| Listar archivos | `GET /v1/upload/files?folder=<folderId>` |
| Mover archivo de carpeta | `PUT /v1/upload/files/:id` |
| Eliminar archivo | `DELETE /v1/upload/files/:id` |
| Listar carpetas | `GET /v1/upload/folders` |
| Crear carpeta | `POST /v1/upload/folders` |
| Renombrar o mover carpeta | `PUT /v1/upload/folders/:id` |
| Eliminar carpeta | `DELETE /v1/upload/folders/:id` |
| Visualizar archivo público | `GET /uploads/:hash:ext` |

---

## Guía de Inicio Rápido

### Requisitos Previos
- [Go](https://go.dev/) 1.24 o superior
- [Node.js](https://nodejs.org/) 20 o superior y npm
- [Docker](https://www.docker.com/) y Docker Compose

### 1. Iniciar Base de Datos PostgreSQL
Levanta el contenedor de PostgreSQL con Docker Compose:
```sh
make compose-up
```

### 2. Ejecutar Backend y Frontend en Desarrollo
Puedes ejecutar ambos entornos simultáneamente:

```sh
# Terminal 1: API Backend en Go (puerto :8080)
make run

# Terminal 2: Panel de Administración en Angular (puerto :4200)
cd web && npm install && npm start
```

O si prefieres levantar todo el entorno de desarrollo local con un solo comando:
```sh
make dev
```

Abre tu navegador en [http://localhost:4200](http://localhost:4200).

---

## Observabilidad y Trazabilidad

El sistema incluye instrumentación distribuida con **OpenTelemetry**:
- **Propagación de Contexto**: Cabeceras W3C `traceparent` y `baggage` gestionadas mediante middleware de Fiber.
- **Capas Instrumentadas**: Controladores, casos de uso, repositorios y proveedores de almacenamiento cuentan con decoradores de trazas (`*/tracing.go`).
- **Exportador**: Las trazas se exportan vía OTLP/gRPC a Jaeger o a cualquier colector compatible con OpenTelemetry.

Variables de configuración en el archivo `.env`:
```env
TRACING_ENABLED=true
TRACING_OTLP_ENDPOINT=localhost:4317
TRACING_SAMPLE_RATE=1.0
```

---

## Estructura de Clean Architecture

```
├── cmd/app/main.go          # Punto de entrada y carga de configuración
├── config/                  # Definición de variables de entorno y configuración
├── docs/                    # Documentación OpenAPI / Swagger autogenerada
├── internal/
│   ├── app/                 # Inicialización de la aplicación e inyección de dependencias
│   ├── controller/
│   │   └── restapi/         # Controladores HTTP de Fiber y middlewares (Auth, Roles, Tenant)
│   ├── entity/              # Modelos puros del dominio (Project, APIKey, Content, User, Media)
│   ├── repo/                # Contratos abstractos de repositorios e implementaciones
│   │   ├── filestore/       # Almacenamiento en archivos de esquemas JSON Schema
│   │   ├── persistent/      # Repositorios PostgreSQL (usuarios, proyectos, api keys, etc.)
│   │   └── storage/         # Proveedores de almacenamiento de archivos en disco
│   └── usecase/             # Lógica de negocio (casos de uso) y decoradores de trazabilidad
├── migrations/              # Archivos de migración de base de datos SQL (golang-migrate)
├── pkg/                     # Paquetes y utilidades reutilizables (jwt, logger, postgres, etc.)
└── web/                     # Panel de administración en Angular 19 + NG-ZORRO
```

---

