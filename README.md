# Tarea 1 — UZM (API + CLI)

Sistema mínimo de librería universitaria con **API REST en Go + SQLite** y **cliente de consola (CLI)**.
Flujos cubiertos: registro/login, usuarios (saldo), libros (venta/arriendo), ventas, **préstamos** con devolución y **multa** por atraso, ranking de populares y transacciones.

---

## Requisitos

* Go **1.21+**
* Git
* (Windows) PowerShell 5+

## Estructura del repo

```
.
├─ main.go                  # servidor HTTP (API)
├─ internal/
│  ├─ api/                  # handlers (users, books, sales, loans, etc.)
│  └─ db/                   # apertura DB y migraciones
├─ data/
│  └─ uzm.db               # base SQLite (se crea sola)
└─ cmd/
   └─ cli/
      └─ main.go           # cliente de consola
```

## Cómo correr (local)

### 1) Servidor (API)

```bash
# desde la carpeta raíz del proyecto
go run .
# prueba de salud
curl http://localhost:8080/health
```

### 2) Cliente CLI

En **otra** terminal:

```bash
go run ./cmd/cli
```

> Windows (recomendado para ver bien tildes/ñ):
>
> ```powershell
> chcp 65001 > $null
> $OutputEncoding = [Console]::OutputEncoding = [Text.UTF8Encoding]::new()
> ```

### (Opcional) URL de la API por variable de entorno

Si el CLI soporta `UZM_API_URL`:

```bash
# ejemplo contra una VM
UZM_API_URL=http://<IP_VM>:8080 go run ./cmd/cli
```

## Endpoints principales

* **Auth**

  * `POST /login` – login simple (email, password)
* **Users**

  * `POST /users` – crear
  * `GET /users` – listar
  * `GET /users/:id` – detalle
  * `PATCH /users/:id` – `{ "abonar": <monto> }`
* **Books**

  * `POST /books` – crear libro (Venta/Arriendo)
  * `GET /books` – catálogo (solo stock > 0)
  * `PATCH /books/:id` – actualizar `{ price | available_quantity }`
  * `GET /books/popular?limit=10` – ranking por `popularity_score`
* **Sales**

  * `POST /sales` – compra (descuenta saldo, baja stock, +popularidad)
  * `GET /sales` – listar
* **Loans** (préstamos)

  * `POST /loans` – crear (requiere libro en **Arriendo** y stock)
  * `GET /loans` – listar
  * `PATCH /loans/:id/return` – devolver `{ return_date: "DD/MM/YYYY" }`
    Multa = **2 × días de atraso** (saldo puede quedar negativo). Devuelve stock.
* **Transactions**

  * `GET /transactions` – ventas + arriendos (ordenados por fecha)
  * `GET /users/:id/transactions` – historial de un usuario

## Recorrido demo (CLI)

1. **Iniciar sesión** o **Registrarse**.
2. **Mi cuenta → Abonar** (p. ej. 50).
3. **Ver catálogo** y **Carro de compras (Venta)** → comprar.
4. **Populares** → verificar ranking.
5. **Solicitar arriendo** → elegir libro en modalidad Arriendo.
6. **Devolver préstamo** → ingresar fecha de devolución (vacío = hoy; si pones +40 días, aplica multa ≈ 20).
7. **Mi cuenta → Ver historial** → ventas y arriendos.

## Recorrido demo (PowerShell)

```powershell
# Crear usuario + abonar
$u = @{ first_name="Eugenio"; last_name="Perez"; email="eugenio@example.com"; password="123456" } | ConvertTo-Json
Invoke-RestMethod -Method Post http://localhost:8080/users -ContentType 'application/json' -Body $u
$ab = @{ abonar = 50 } | ConvertTo-Json
Invoke-RestMethod -Method Patch http://localhost:8080/users/1 -ContentType 'application/json' -Body $ab

# Libros (Venta + Arriendo)
$b1 = @{ book_name="El principito"; book_category="Infantil"; transaction_type="Venta"; price=12; available_quantity=6 } | ConvertTo-Json
Invoke-RestMethod -Method Post http://localhost:8080/books -ContentType 'application/json' -Body $b1
$b2 = @{ book_name="Papelucho"; book_category="Infantil"; transaction_type="Arriendo"; price=5; available_quantity=2 } | ConvertTo-Json
Invoke-RestMethod -Method Post http://localhost:8080/books -ContentType 'application/json' -Body $b2

# Compra
$sale = @{ user_id=1; book_id=1 } | ConvertTo-Json
Invoke-RestMethod -Method Post http://localhost:8080/sales -ContentType 'application/json' -Body $sale

# Arriendo + devolución tardía
$loan = @{ user_id=1; book_id=2 } | ConvertTo-Json  # ajusta IDs según /books
$lr = Invoke-RestMethod -Method Post http://localhost:8080/loans -ContentType 'application/json' -Body $loan
$fecha = (Get-Date).AddDays(40).ToString("dd/MM/yyyy")
$payload = @{ return_date = $fecha } | ConvertTo-Json
Invoke-RestMethod -Method Patch ("http://localhost:8080/loans/{0}/return" -f $lr.id) -ContentType 'application/json' -Body $payload

# Verificaciones
Invoke-RestMethod http://localhost:8080/users/1
Invoke-RestMethod http://localhost:8080/books | ConvertTo-Json -Depth 10
Invoke-RestMethod http://localhost:8080/transactions | ConvertTo-Json -Depth 10
```

## Reset de base

```bash
# con el servidor detenido
rm -f data/uzm.db            # Linux/Mac
# o
powershell -Command "Remove-Item .\data\uzm.db -ErrorAction Ignore"  # Windows
# luego
go run .  # recrea esquemas
```

## Despliegue en Máquinas Virtuales (MV)

En la VM de la **API**:

```bash
sudo apt update
sudo apt install -y golang git

git clone https://github.com/<tu-usuario>/<tu-repo>.git
cd <tu-repo>

go run .
# prueba
curl http://localhost:8080/health
```

Cliente desde tu PC contra la VM:

```bash
# si tu CLI soporta env
UZM_API_URL=http://<IP_VM>:8080 go run ./cmd/cli
```



## Troubleshooting

* **CLI no conecta**: asegúrate que el server está en `http://localhost:8080` (o ajusta `UZM_API_URL`).
* **Acentos raros en PowerShell**: configurar UTF-8 (arriba).
* **IDs que no coinciden**: consulta `GET /books` o `GET /loans` para ver IDs reales.
* **`/loans` vacío**: normal si aún no creaste préstamos.

## Notas

* Multa por atraso en devolución: **2 usm/día** (saldo puede quedar negativo).
* `popularity_score` sube tanto por ventas como por arriendos.
* `GET /books` solo muestra libros con `available_quantity > 0`.
