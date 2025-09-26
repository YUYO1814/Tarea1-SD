# Tarea 1 — UZM (API + CLI)

Sistema mínimo de librería universitaria con **API REST en Go + SQLite** y **cliente de consola (CLI)**.
Flujos: registro/login, usuarios (saldo), libros (venta/arriendo), ventas, préstamos con devolución y multa por atraso, ranking de populares y transacciones.

* **Puerto**: `8080`
* **DB**: SQLite (`data/uzm.db`)
* **Lenguaje**: Go • **Framework**: Gin

---

## Requisitos

* Go **1.22+** (solo si compilas localmente).
* Git.
* (Windows) PowerShell 5+ para el demo CLI.

> **Nota**: Las VMs tienen **muy poco espacio**. Se recomienda **compilar el binario en tu PC** y subir **solo el ejecutable** a la VM (ver “Despliegue en VM — Método recomendado”).

---

## Estructura del repo

```
.
├─ main.go                  # servidor HTTP (API)
├─ internal/
│  ├─ api/                  # handlers (users, books, sales, loans, etc.)
│  └─ db/                   # apertura DB y migraciones
├─ data/
│  └─ .gitkeep              # la base SQLite (uzm.db) se crea sola al iniciar
├─ cmd/
│  └─ cli/
│     └─ main.go            # cliente de consola (opcional)
├─ README.md
└─ .gitignore
```

> `data/uzm.db` **no se versiona** (está ignorado). Se crea al arrancar el servidor.

---

## Cómo correr (local)

### 1) Servidor (API)

```bash
# desde la raíz del proyecto
go run .
# prueba de salud
curl http://localhost:8080/health   # {"status":"ok"}
```

### 2) Cliente CLI (opcional)

En otra terminal:

```bash
go run ./cmd/cli
```

Windows (para tildes/ñ):

```powershell
chcp 65001 > $null
$OutputEncoding = [Console]::OutputEncoding = [Text.UTF8Encoding]::new()
```

**(Opcional) URL de la API por variable de entorno**

```bash
# ejemplo contra una VM
UZM_API_URL=http://<IP_VM>:8080 go run ./cmd/cli
```

---

## Despliegue en Máquinas Virtuales (VM)

### ✅ Estado de despliegue actual

* **Verificado:** VM **SD2025-2-30** (`10.10.31.12`)

  * Binario: `/home/ubuntu/uzm-server`
  * Carpeta trabajo: `/home/ubuntu/Tarea1-SD`
  * Salud: `curl -s http://localhost:8080/health` → `{"status":"ok"}`


### Método recomendado (poco espacio): **subir binario**

1. **Compilar en tu PC** el ejecutable Linux (estático):

```bash
# macOS / Linux:
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o uzm-server .

# PowerShell (Windows):
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
go build -ldflags "-s -w" -o uzm-server .
```

2. **Subir a la VM**:

```bash
scp uzm-server ubuntu@<IP_VM>:/home/ubuntu/
```

3. **Preparar carpeta y dejar el server corriendo**:

```bash
ssh ubuntu@<IP_VM>
chmod +x ~/uzm-server
mkdir -p ~/Tarea1-SD/data
cd ~/Tarea1-SD

# ejecutar en background y loguear a logs.txt
nohup ~/uzm-server > logs.txt 2>&1 &

# comprobar
pgrep -a uzm-server
curl -s http://localhost:8080/health   # {"status":"ok"}
```

**Logs / detener / reiniciar**

```bash
tail -n 200 ~/Tarea1-SD/logs.txt
pkill uzm-server
nohup ~/uzm-server > ~/Tarea1-SD/logs.txt 2>&1 &
```

> Si necesitas liberar espacio en la VM:
>
> ```bash
> sudo apt-get clean
> sudo rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/*
> sudo journalctl --vacuum-time=1d
> df -h /
> ```

### Método alternativo (no recomendado en VM): compilar en la VM

```bash
sudo apt update
sudo apt install -y golang git
git clone https://github.com/<tu-usuario>/<tu-repo>.git
cd <tu-repo>

go run .
curl http://localhost:8080/health
```

> Este método descarga toolchains/módulos y suele **quedarse sin espacio**. Fue horrible :( 

---

## Smoke Test (automático)

Ejecutar en la **VM donde corre el server**:

```bash
cd ~/Tarea1-SD

cat > smoke.sh <<'EOF'
#!/usr/bin/env bash
set -e

echo "Health:"; curl -s http://localhost:8080/health; echo

EMAIL="smoke.$(date +%s)@example.com"

# 1) Usuario
UJSON=$(curl -s -X POST http://localhost:8080/users \
  -H 'Content-Type: application/json' \
  -d "{\"first_name\":\"Smoke\",\"last_name\":\"User\",\"email\":\"$EMAIL\",\"password\":\"123456\"}")
echo "$UJSON"
USER_ID=$(echo "$UJSON" | grep -o '"id":[[:space:]]*[0-9]\+' | head -n1 | tr -dc '0-9')
echo "USER_ID=$USER_ID"

# 2) Login
curl -s -X POST http://localhost:8080/login \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"123456\"}"; echo

# 3) Abonar
curl -s -X PATCH http://localhost:8080/users/$USER_ID \
  -H 'Content-Type: application/json' \
  -d '{"abonar":50}'; echo

# 4) Libro Venta
B1JSON=$(curl -s -X POST http://localhost:8080/books \
  -H 'Content-Type: application/json' \
  -d '{"book_name":"SMOKE Libro Venta","book_category":"Test","transaction_type":"Venta","price":12,"available_quantity":2}')
echo "$B1JSON"
BID_SALE=$(echo "$B1JSON" | grep -o '"id":[[:space:]]*[0-9]\+' | head -n1 | tr -dc '0-9')
echo "BID_SALE=$BID_SALE"

# 5) Libro Arriendo
B2JSON=$(curl -s -X POST http://localhost:8080/books \
  -H 'Content-Type: application/json' \
  -d '{"book_name":"SMOKE Libro Arriendo","book_category":"Test","transaction_type":"Arriendo","price":5,"available_quantity":1}')
echo "$B2JSON"
BID_RENT=$(echo "$B2JSON" | grep -o '"id":[[:space:]]*[0-9]\+' | head -n1 | tr -dc '0-9')
echo "BID_RENT=$BID_RENT"

# 6) Catálogo
curl -s http://localhost:8080/books; echo

# 7) Compra
curl -s -X POST http://localhost:8080/sales \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\": $USER_ID, \"book_id\": $BID_SALE}"; echo

# 8) Préstamo
LJSON=$(curl -s -X POST http://localhost:8080/loans \
  -H 'Content-Type: application/json' \
  -d "{\"user_id\": $USER_ID, \"book_id\": $BID_RENT}")
echo "$LJSON"
LOAN_ID=$(echo "$LJSON" | grep -o '"id":[[:space:]]*[0-9]\+' | head -n1 | tr -dc '0-9')
echo "LOAN_ID=$LOAN_ID"

# 9) Devolver con atraso (~10 días)
FECHA_TARDE=$(date -d "+40 days" +"%d/%m/%Y")
curl -s -X PATCH http://localhost:8080/loans/$LOAN_ID/return \
  -H 'Content-Type: application/json' \
  -d "{\"return_date\":\"$FECHA_TARDE\"}"; echo

# 10) Verificaciones
curl -s http://localhost:8080/users/$USER_ID; echo
curl -s http://localhost:8080/users/$USER_ID/transactions; echo
curl -s "http://localhost:8080/books/popular?limit=5"; echo
curl -s http://localhost:8080/loans; echo
EOF

chmod +x smoke.sh
bash smoke.sh
```

**Esperado (resumen):**

* `{"status":"ok"}` en `/health`
* Usuario creado, login OK, saldo sube a 50
* Se crean libros “SMOKE…” Venta/Arriendo
* Compra OK, préstamo OK
* Devolución tardía: `days_late ~10`, `penalty = 20`
* Transacciones y ranking popular se actualizan
* `/loans` muestra el préstamo finalizado

---

## Endpoints principales

**Auth**

* `POST /login` – login simple (email, password)

**Users**

* `POST /users` – crear
* `GET /users` – listar
* `GET /users/:id` – detalle
* `PATCH /users/:id` – `{ "abonar": <monto> }`

**Books**

* `POST /books` – crear libro (Venta/Arriendo)
* `GET /books` – catálogo (solo stock > 0)
* `PATCH /books/:id` – actualizar `{ price | available_quantity }`
* `GET /books/popular?limit=10` – ranking por `popularity_score`

**Sales**

* `POST /sales` – compra (descuenta saldo, baja stock, +popularidad)
* `GET /sales` – listar

**Loans (préstamos)**

* `POST /loans` – crear (requiere `Arriendo` y stock)
* `GET /loans` – listar
* `PATCH /loans/:id/return` – devolver `{ "return_date": "DD/MM/YYYY" }`
  Multa = `2 × días de atraso` (saldo puede quedar negativo). Devuelve stock.

**Transactions**

* `GET /transactions` – ventas + arriendos (por fecha)
* `GET /users/:id/transactions` – historial de un usuario

---

## Recorrido demo (CLI)

1. Iniciar sesión o Registrarse.
2. Mi cuenta → Abonar (p. ej. 50).
3. Ver catálogo y Carro de compras (Venta) → comprar.
4. Populares → verificar ranking.
5. Solicitar arriendo → elegir libro en modalidad Arriendo.
6. Devolver préstamo → fecha (vacío = hoy; +40 días → multa ≈ 20).
7. Mi cuenta → Ver historial → ventas y arriendos.

---

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

# Arriendo + devolución tardía (≈20 de multa)
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

---

## Reset de base

Con el servidor detenido:

```bash
rm -f data/uzm.db            # Linux/Mac
# o en Windows PowerShell:
powershell -Command "Remove-Item .\data\uzm.db -ErrorAction Ignore"
```

Al reiniciar el server, recrea esquemas.

---


## Troubleshooting

* **CLI no conecta**: verifica que la API esté en `http://localhost:8080` en la VM (o usa `UZM_API_URL`).
* **Acentos raros en PowerShell**: ver configuración UTF-8 arriba.
* **IDs no coinciden**: usa `GET /books`, `GET /loans` para ver IDs reales.
* **/loans vacío**: normal si aún no creaste préstamos.
* **Poco espacio en VM**: sube **solo el binario**, no compiles en la VM.

---

## Notas

* Multa por atraso en devolución: `2 usm/día` (saldo puede quedar negativo).
* `popularity_score` sube por **ventas y arriendos**.
* `GET /books` lista solo libros con `available_quantity > 0`.
