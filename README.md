# Zelko — Self-Hosted LaTeX Editor & PDF Compiler

**Zelko** es un backend API en Go que permite crear, editar y compilar documentos LaTeX a PDF desde un navegador. Es una alternativa gratuita, ilimitada y de software libre a Overleaf, diseñada para uso personal y para compartir con compañeros o la comunidad.

No hay límites de documentos, no hay planes de pago, no hay restricciones — solo tu servidor, tu LaTeX, tus PDFs.

---

## Motivación

LaTeX instalado en el equipo, surgió la idea: ¿por qué no tener mi propio Overleaf? Gratis, sin límites, sin depender de servicios externos ni pagar suscripciones. Zelko nace de esa necesidad — y del deseo de compartirla con quien quiera usarla.

---

## Stack Tecnológico

| Componente | Tecnología |
|---|---|
| **Lenguaje** | Go 1.26.4 |
| **Framework HTTP** | Gin v1.12.0 |
| **Base de datos** | PostgreSQL (GORM + pgx) |
| **Cache / Sesiones** | Redis (go-redis/v9) |
| **Autenticación** | PASETO v2 + bcrypt |
| **2FA** | TOTP (xlzd/gotp) |
| **Compilación LaTeX** | latexmk |
| **Email** | SMTP + Resend (fallback) |
| **Seguridad** | gin-helmet, CORS, rate limiting, claves de idempotencia |

---

## Arquitectura

El proyecto sigue **Clean Architecture por feature**, con 4 capas por feature:

```
internal/features/<feature>/
├── domain.go      # Interfaces Repository/Service + constructores de dominio
├── handler.go     # Handlers HTTP (Gin)
├── repository.go  # Implementación PostgreSQL (GORM)
└── usecase.go     # Lógica de negocio
```

### Features principales

- **`auth`** — Registro, login, 2FA, verificación de email, recuperación de contraseña, sesiones
- **`user`** — Perfil de usuario, avatar, datos personales
- **`document`** — CRUD de documentos LaTeX, compilación a PDF, previsualización

### Flujo de compilación LaTeX

1. El usuario crea un documento con contenido LaTeX
2. Solicita compilación vía `POST /documents/:slug/compile`
3. El servidor escribe el `.tex` en `tmp/<slug>/`
4. Ejecuta `latexmk -pdf` en el directorio
5. Devuelve una URL para descargar/previsualizar el PDF generado
6. Los archivos temporales se limpian automáticamente

---

## Endpoints

### Públicos

| Método | Ruta | Descripción |
|---|---|---|
| GET | `/` | Mensaje de bienvenida |
| GET | `/health` | Health check |
| POST | `/api/v1/auth/login` | Inicio de sesión |
| POST | `/api/v1/auth/register` | Registro |
| POST | `/api/v1/auth/forward-email-verification` | Reenviar verificación |
| POST | `/api/v1/auth/forgot-password` | Recuperar contraseña |
| POST | `/api/v1/auth/logout` | Cerrar sesión |
| POST | `/api/v1/auth/refresh-token` | Refrescar token |

### Autenticados (requieren Bearer PASETO)

| Método | Ruta | Descripción |
|---|---|---|
| GET | `/api/v1/auth/confirm-account` | Confirmar cuenta |
| GET | `/api/v1/auth/session` | Sesión actual |
| POST | `/api/v1/auth/verify-email` | Verificar email |
| PATCH | `/api/v1/auth/reset-password` | Restablecer contraseña |
| PATCH | `/api/v1/auth/change-password` | Cambiar contraseña |
| GET | `/api/v1/auth/generate-two-factor` | Generar 2FA |
| GET | `/api/v1/user/me` | Obtener perfil |
| PUT | `/api/v1/user/me` | Actualizar perfil |
| GET | `/api/v1/documents` | Listar documentos |
| GET | `/api/v1/documents/visibilities` | Listar opciones de visibilidad |
| GET | `/api/v1/documents/:slug` | Obtener documento |
| POST | `/api/v1/documents` | Crear documento |
| PUT | `/api/v1/documents/:slug` | Actualizar documento |
| PATCH | `/api/v1/documents/:slug/visibility` | Cambiar visibilidad del documento |
| POST | `/api/v1/documents/:slug/compile` | Compilar LaTeX a PDF |
| GET | `/api/v1/documents/:slug/preview` | Previsualizar PDF |

### Públicos (sin autenticación)

| Método | Ruta | Descripción |
|---|---|---|
| GET | `/api/v1/documents/:slug/public` | Ver documento público compartido |

---

## Requisitos

- Go 1.26+
- PostgreSQL
- Redis
- `latexmk` (parte de TeX Live)
- Una instalación completa de LaTeX (TeX Live recomendado)

---

## Configuración

1. Clona el repositorio:

```bash
git clone <repo-url>
cd zelko
```

2. Crea un archivo `.env` basado en las variables de entorno que necesites:

```
GO_ENV=dev
HOST=localhost
PORT=5432
DBUSER=postgres
PASSWORD=postgres
DBNAME=zelko
REDIS_HOST=localhost
REDIS_PORT=6379
SECRET_KEY_JWT=<32-bytes-base64>
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=...
SMTP_PASSWORD=...
HOST_URL_DEV=http://localhost:5173
HOST_URL_PROD=http://localhost:5173
```

3. Inicia el servidor:

```bash
go run main.go
```

El servidor se inicia en el puerto `4106` y automáticamente ejecuta migraciones y seeding en modo `dev`.

---

## Licencia

Este proyecto está bajo una **Licencia No Comercial** — uso personal, educativo e investigación permitidos. No está permitido el uso comercial sin autorización expresa.

Para consultas de licencias comerciales, contacta a: santosxphdz34@gmail.com

---

## Notas

- Los archivos temporales de compilación se almacenan en `tmp/` y se limpian automáticamente
- Este proyecto es puramente backend; necesitas un frontend para interactuar con la API
