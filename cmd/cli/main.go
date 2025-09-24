package main

import (
	bufio "bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ======== Config ========
const baseURL = "http://localhost:8080"

// ======== Tipos que coinciden con la API ========

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	USMPesos  int64  `json:"usm_pesos"`
}

type Book struct {
	ID              int64  `json:"id"`
	BookName        string `json:"book_name"`
	BookCategory    string `json:"book_category"`
	TransactionType string `json:"transaction_type"` // "Venta" | "Arriendo"
	Price           int64  `json:"price"`
	Status          string `json:"status"`
	PopularityScore int64  `json:"popularity_score"`
	Inventory       struct {
		AvailableQuantity int64 `json:"available_quantity"`
	} `json:"inventory"`
}

type BooksResp struct {
	Books []Book `json:"books"`
}

type TransactionsResp struct {
	Transactions []Transaction `json:"transactions"`
}

type Transaction struct {
	ID     int64  `json:"id"`
	Type   string `json:"type"`
	UserID int64  `json:"user_id"`
	BookID int64  `json:"book_id"`
	Date   string `json:"date"`
}

// ======== Utiles de consola ========

var in = bufio.NewReader(os.Stdin)

func readLine(prompt string) string {
	fmt.Print(prompt)
	text, _ := in.ReadString('\n')
	return strings.TrimSpace(text)
}

func readInt(prompt string) int64 {
	for {
		s := readLine(prompt)
		if s == "" {
			return 0
		}
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
		fmt.Println("→ Ingresa un número válido.")
	}
}

// ======== HTTP helpers ========

func getJSON(path string, out any) error {
	resp, err := http.Get(baseURL + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GET %s → status %s", path, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func postJSON(path string, body any, out any) error {
	b, _ := json.Marshal(body)
	resp, err := http.Post(baseURL+path, "application/json", strings.NewReader(string(b)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("POST %s → status %s", path, resp.Status)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func patchJSON(path string, body any, out any) error {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPatch, baseURL+path, strings.NewReader(string(b)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("PATCH %s → status %s", path, resp.Status)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// ======== Menús ========

func main() {
	for {
		switch firstMenu() {
		case 1:
			registerFlow()
		case 2:
			if u, ok := loginFlow(); ok {
				secondMenu(u)
			}
		case 3:
			fmt.Println("¡Gracias por usar UZM!")
			return
		}
	}
}

func firstMenu() int {
	fmt.Println("\nMenu")
	fmt.Println("1. Registrarse")
	fmt.Println("2. Iniciar sesión")
	fmt.Println("3. Terminar ejecución")
	for {
		opt := readLine("Seleccione una opción: ")
		switch opt {
		case "1":
			return 1
		case "2":
			return 2
		case "3":
			return 3
		}
		fmt.Println("→ Opción inválida.")
	}
}

func registerFlow() {
	fmt.Println("\n== Registro ==")
	fn := readLine("Nombre: ")
	ln := readLine("Apellido: ")
	em := readLine("Email: ")
	pw := readLine("Contraseña: ")
	var u User
	if err := postJSON("/users", map[string]any{
		"first_name": fn,
		"last_name":  ln,
		"email":      em,
		"password":   pw,
	}, &u); err != nil {
		fmt.Println("Error registrando:", err)
		return
	}
	fmt.Println("Usuario creado con éxito. ID:", u.ID)
}

func loginFlow() (User, bool) {
	fmt.Println("\n== Iniciar sesión ==")
	em := readLine("Email: ")
	pw := readLine("Contraseña: ")
	var u User
	if err := postJSON("/login", map[string]any{"email": em, "password": pw}, &u); err != nil {
		fmt.Println("Error de login:", err)
		return User{}, false
	}
	fmt.Printf("Bienvenido, %s %s!\n", u.FirstName, u.LastName)
	return u, true
}

func secondMenu(user User) {
	for {
		fmt.Println("\nMenu")
		fmt.Println("1. Ver catálogo")
		fmt.Println("2. Carro de compras (Venta)")
		fmt.Println("3. Mi cuenta")
		fmt.Println("4. Populares")
		fmt.Println("5. Solicitar arriendo") // ← NUEVO
		fmt.Println("6. Devolver préstamo")  // ← NUEVO
		fmt.Println("7. Salir al menú principal")
		op := readLine("Seleccione una opción: ")
		switch op {
		case "1":
			showCatalog()
		case "2":
			user = cartFlow(user)
		case "3":
			user = myAccount(user)
		case "4":
			showPopular()
		case "5":
			loanRequestFlow(user) // ← NUEVO
		case "6":
			loanReturnFlow(user) // ← NUEVO
		case "7":
			return
		default:
			fmt.Println("→ Opción inválida.")
		}
	}
}

// ======== Catálogo ========

func showCatalog() []Book {
	var br BooksResp
	if err := getJSON("/books", &br); err != nil {
		fmt.Println("Error catálogo:", err)
		return nil
	}
	fmt.Println("-----------------------------------------------------------------")
	fmt.Printf("| %-7s | %-20s | %-10s | %-8s | %-5s |\n", "ID", "Nombre", "Categoría", "Modo", "Valor")
	fmt.Println("-----------------------------------------------------------------")
	for _, b := range br.Books {
		fmt.Printf("| %-7d | %-20s | %-10s | %-8s | %-5d |\n", b.ID, trim(b.BookName, 20), trim(b.BookCategory, 10), b.TransactionType, b.Price)
	}
	fmt.Println("-----------------------------------------------------------------")
	return br.Books
}

func trim(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}

// ======== Carrito (Venta) ========

func cartFlow(user User) User {
	fmt.Println("\n== Carrito (solo Venta por ahora) ==")
	books := showCatalog()
	if len(books) == 0 {
		fmt.Println("No hay libros con stock disponible.")
		return user
	}
	// Mapa por ID
	idx := map[int64]Book{}
	for _, b := range books {
		idx[b.ID] = b
	}

	fmt.Println("Ingrese IDs de libros a comprar (separados por espacio). Enter vacío para terminar.")
	line := readLine("> ")
	if strings.TrimSpace(line) == "" {
		return user
	}
	parts := strings.Fields(strings.ReplaceAll(line, ",", " "))
	var cart []Book
	seen := map[int64]bool{}
	for _, p := range parts {
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			continue
		}
		b, ok := idx[id]
		if !ok {
			continue
		}
		if b.TransactionType != "Venta" {
			fmt.Printf("- %s no está en Venta, se ignora.\n", b.BookName)
			continue
		}
		if b.Inventory.AvailableQuantity <= 0 {
			fmt.Printf("- %s sin stock, se ignora.\n", b.BookName)
			continue
		}
		if !seen[b.ID] {
			cart = append(cart, b)
			seen[b.ID] = true
		}
	}
	if len(cart) == 0 {
		fmt.Println("Carro vacío.")
		return user
	}

	// Resumen
	total := int64(0)
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("| %-20s | %-8s | %-5s |\n", "Nombre", "Modalidad", "Valor")
	fmt.Println("------------------------------------------------------------")
	for _, b := range cart {
		fmt.Printf("| %-20s | %-8s | %-5d |\n", trim(b.BookName, 20), b.TransactionType, b.Price)
		total += b.Price
	}
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("Total: %d usm pesos\n", total)
	fmt.Printf("Tu saldo: %d usm pesos\n", user.USMPesos)

	if user.USMPesos >= total {
		fmt.Print("Confirmar pedido (Enter para confirmar, cualquier texto para cancelar): ")
		if readLine("") == "" {
			user = executeSales(cart, user)
		}
		return user
	}

	fmt.Printf("No alcanza el saldo. Tienes %d y el pedido cuesta %d.\n", user.USMPesos, total)
	fmt.Println("1) Optimizar carrito (agrega del más barato al más caro según fondos)")
	fmt.Println("2) Cancelar")
	opt := readLine("Seleccione opción: ")
	if opt != "1" {
		return user
	}
	// Optimizar
	sorted := append([]Book(nil), cart...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Price < sorted[j].Price })
	var optCart []Book
	sum := int64(0)
	for _, b := range sorted {
		if sum+b.Price <= user.USMPesos {
			optCart = append(optCart, b)
			sum += b.Price
		}
	}
	if len(optCart) == 0 {
		fmt.Println("Ni el libro más barato cabe en tu saldo. Cancela o abona fondos en Mi cuenta.")
		return user
	}
	fmt.Println("Carro optimizado:")
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("| %-20s | %-8s | %-5s |\n", "Nombre", "Modalidad", "Valor")
	fmt.Println("------------------------------------------------------------")
	for _, b := range optCart {
		fmt.Printf("| %-20s | %-8s | %-5d |\n", trim(b.BookName, 20), b.TransactionType, b.Price)
	}
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("Total optimizado: %d usm pesos (saldo %d)\n", sum, user.USMPesos)
	fmt.Print("Confirmar pedido optimizado (Enter confirma): ")
	if readLine("") == "" {
		user = executeSales(optCart, user)
	}
	return user
}

func executeSales(items []Book, user User) User {
	for _, b := range items {
		var saleResp struct {
			ID       int64  `json:"id"`
			UserID   int64  `json:"user_id"`
			BookID   int64  `json:"book_id"`
			SaleDate string `json:"sale_date"`
		}
		err := postJSON("/sales", map[string]any{"user_id": user.ID, "book_id": b.ID}, &saleResp)
		if err != nil {
			fmt.Printf("× Falló compra de %s: %v\n", b.BookName, err)
			continue
		}
		user.USMPesos -= b.Price
		fmt.Printf("✔ Comprado: %s (fecha %s)\n", b.BookName, saleResp.SaleDate)
	}
	return user
}

// ======== Mi cuenta ========

func myAccount(user User) User {
	for {
		fmt.Println("\nMi cuenta")
		fmt.Println("1. Consultar saldo")
		fmt.Println("2. Abonar usm pesos")
		fmt.Println("3. Ver historial de compras y arriendos")
		fmt.Println("4. Volver")
		s := readLine("Seleccione: ")
		switch s {
		case "1":
			var u User
			if err := getJSON("/users/"+strconv.FormatInt(user.ID, 10), &u); err != nil {
				fmt.Println("Error:", err)
				break
			}
			user = u
			fmt.Println("Saldo:", user.USMPesos, "usm pesos")
		case "2":
			amt := readInt("Monto a abonar: ")
			if amt <= 0 {
				fmt.Println("Nada que abonar")
				break
			}
			if err := patchJSON("/users/"+strconv.FormatInt(user.ID, 10), map[string]any{"abonar": amt}, &user); err != nil {
				fmt.Println("Error abonando:", err)
				break
			}
			fmt.Println("Nuevo saldo:", user.USMPesos)
		case "3":
			var tr TransactionsResp
			if err := getJSON("/users/"+strconv.FormatInt(user.ID, 10)+"/transactions", &tr); err != nil {
				fmt.Println("Error:", err)
				break
			}
			fmt.Println("-------------------------------------------------------------------------------------------")
			fmt.Printf("| %-3s | %-9s | %-6s | %-4s | %-12s |\n", "ID", "Tipo", "UserID", "BID", "Fecha")
			fmt.Println("-------------------------------------------------------------------------------------------")
			for _, t := range tr.Transactions {
				fmt.Printf("| %-3d | %-9s | %-6d | %-4d | %-12s |\n", t.ID, t.Type, t.UserID, t.BookID, t.Date)
			}
			fmt.Println("-------------------------------------------------------------------------------------------")
		case "4":
			return user
		default:
			fmt.Println("→ Opción inválida.")
		}
	}
}

// ======== Populares ========

func showPopular() {
	var br BooksResp
	if err := getJSON("/books/popular?limit=5", &br); err != nil {
		fmt.Println("Error populares:", err)
		return
	}
	fmt.Println("---------------------------------------------------------------")
	fmt.Printf("| %-7s | %-20s | %-10s | %-12s |\n", "ID", "Nombre", "Categoría", "Popularidad")
	fmt.Println("---------------------------------------------------------------")
	for _, b := range br.Books {
		fmt.Printf("| %-7d | %-20s | %-10s | %-12d |\n", b.ID, trim(b.BookName, 20), trim(b.BookCategory, 10), b.PopularityScore)
	}
	fmt.Println("---------------------------------------------------------------")
}

// ======== Nota ========
// Próximos pasos que se integrarán aquí:
// - Flujo de arriendo (POST /loans) y devolución (PATCH /loans/:id/return)
// - Validación de cantidades por inventario en el carrito
// - Mostrar fecha de devolución estimada para arriendos
//
// Este CLI asume que el servidor está escuchando en http://localhost:8080.
// Puedes cambiar baseURL arriba o leerlo desde una variable de entorno.

// Fin.

func loanRequestFlow(user User) {
	fmt.Println("\n== Solicitar arriendo ==")
	books := showCatalog()
	if len(books) == 0 {
		return
	}
	fmt.Println("Ingresa el ID del libro en modalidad Arriendo:")
	id := readInt("> ")
	if id == 0 {
		return
	}
	var picked *Book
	for i := range books {
		if books[i].ID == id {
			picked = &books[i]
			break
		}
	}
	if picked == nil {
		fmt.Println("No existe ese ID.")
		return
	}
	if picked.TransactionType != "Arriendo" {
		fmt.Println("Ese libro no está en Arriendo.")
		return
	}
	if picked.Inventory.AvailableQuantity <= 0 {
		fmt.Println("Sin stock.")
		return
	}
	var out struct {
		ID      int64  `json:"id"`
		DueDate string `json:"due_date"`
		Status  string `json:"status"`
	}
	if err := postJSON("/loans", map[string]any{"user_id": user.ID, "book_id": id}, &out); err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("✔ Arriendo creado (id %d). Fecha límite: %s\n", out.ID, out.DueDate)
}

func loanReturnFlow(user User) {
	fmt.Println("\n== Devolver préstamo ==")
	// Obtener todos los préstamos y filtrar por usuario/pendiente
	var resp struct {
		Loans []struct {
			ID        int64  `json:"id"`
			UserID    int64  `json:"user_id"`
			BookID    int64  `json:"book_id"`
			Status    string `json:"status"`
			StartDate string `json:"start_date"`
			DueDate   string `json:"due_date"`
		} `json:"loans"`
	}
	if err := getJSON("/loans", &resp); err != nil {
		fmt.Println("Error:", err)
		return
	}
	var pending []struct {
		ID, BookID int64
		Due        string
	}
	for _, l := range resp.Loans {
		if l.UserID == user.ID && l.Status == "pendiente" {
			pending = append(pending, struct {
				ID, BookID int64
				Due        string
			}{l.ID, l.BookID, l.DueDate})
		}
	}
	if len(pending) == 0 {
		fmt.Println("No tienes préstamos pendientes.")
		return
	}
	fmt.Println("Préstamos pendientes:")
	for _, p := range pending {
		fmt.Printf("- id %d (book %d) vence %s\n", p.ID, p.BookID, p.Due)
	}
	loanID := readInt("Ingresa id de préstamo a devolver: ")
	if loanID == 0 {
		return
	}
	date := readLine("Fecha de devolución (DD/MM/YYYY, vacío = hoy): ")
	if strings.TrimSpace(date) == "" {
		date = time.Now().Format("02/01/2006")
	}
	var out struct {
		Status   string `json:"status"`
		DaysLate int64  `json:"days_late"`
		Penalty  int64  `json:"penalty"`
	}
	if err := patchJSON("/loans/"+strconv.FormatInt(loanID, 10)+"/return", map[string]any{"return_date": date}, &out); err != nil {
		fmt.Println("Error devolviendo:", err)
		return
	}
	fmt.Printf("✔ Devuelto. Atraso: %d días, multa: %d\n", out.DaysLate, out.Penalty)
}

// Fin.
