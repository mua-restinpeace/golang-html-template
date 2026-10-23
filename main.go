package main

import (
	"context"
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

type User struct {
	Name  string
	Email string
	Age   int
}

type Product struct {
	ID          int
	Name        string
	Price       int
	Stock       int
	Description string
}

type ProductForm struct {
	Heading string
	Action  string
	Product Product
	Errors  []string
}

var UserData = []User{
	{Name: "Budi Bakoko", Email: "budi@mail.com", Age: 23},
	{Name: "Cahya Bakoko", Email: "cahya@mail.com", Age: 21},
	{Name: "Dedi Bakoko", Email: "dedi@mail.com", Age: 24},
}

var ProductData = []Product{
	{Name: "Mouse", Price: 300000},
	{Name: "Keyboard", Price: 500000},
	{Name: "Laptop", Price: 12000000},
}

var funcs = template.FuncMap{}

func parsePage(file string) *template.Template {
	return template.Must(template.New("layout").Funcs(funcs).ParseFiles(
		"template/layout.html",
		"template/header.html",
		"template/sidebar.html",
		"template/footer.html",
		"template/pages/"+file,
	))
}

var (
	tplHome         = parsePage("home.html")
	tplAnnouncement = parsePage("announcement.html")
	tplProduct      = parsePage("product.html")
	tplProductForm  = parsePage("product_form.html")
)

func main() {
	_ = godotenv.Load()

	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		log.Fatal("couldn't find DB_URL")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Database connection failed: %s", err)
	}

	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to contact database: %s", err)
	}

	log.Println("Database connected")

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))
	mux.HandleFunc("GET /", homeHandler)
	mux.HandleFunc("GET /announcements", announcementHandler)
	mux.HandleFunc("GET /products", productHandler(db))
	mux.HandleFunc("GET /products/form", productFormHandler)
	mux.HandleFunc("POST /products", productSimpanHandler(db))

	// udpate
	mux.HandleFunc("GET /products/{id}/edit", productEditHandler(db))
	mux.HandleFunc("POST /products/{id}/edit", productUpdateHandler(db))

	// delete
	mux.HandleFunc("POST /products/{id}/delete", productHardDeleteHandler(db))
	mux.HandleFunc("POST /products/{id}/deactivate", productDeactivateHandler(db))

	http.ListenAndServe("localhost:8000", mux)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"Title":   "Home",
		"Heading": "Website Heading",
		"Hewan":   []string{"shark", "whale", "dolphin"},
		"Users":   UserData,
	}
	if err := tplHome.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("failed to render Home: %s", err.Error())
		http.Error(w, "failed to render Home", http.StatusInternalServerError)
	}
}

func productHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		products, err := getAllProduct(r.Context(), db)
		if err != nil {
			log.Println("failed to load products")
			return
		}

		data := map[string]any{
			"Title":    "Products",
			"Products": products,
		}

		if err := tplProduct.ExecuteTemplate(w, "layout", data); err != nil {
			log.Printf("failed to render Product: %s\n", err.Error())
			http.Error(w, "failed to load Products", http.StatusInternalServerError)
		}
	}
}

func productFormHandler(w http.ResponseWriter, r *http.Request) {
	productForm := ProductForm{
		Heading: "Add Products",
		Action:  "/products",
	}
	data := map[string]any{
		"Title":       "Products",
		"ProductForm": productForm,
	}
	if err := tplProductForm.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("failed to load product form: %s\n", err.Error())
		http.Error(w, "failed to load product form", http.StatusInternalServerError)
	}
}

func productEditHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Fatal(err)
		}

		product, err := getProductById(r.Context(), db, id)
		if err != nil {
			log.Fatal(err)
		}
		productForm := ProductForm{
			Heading: "Edit Products",
			Product: *product,
		}
		data := map[string]any{
			"Title":       "Products",
			"ProductForm": productForm,
		}
		if err := tplProductForm.ExecuteTemplate(w, "layout", data); err != nil {
			log.Printf("failed to load product form: %s\n", err.Error())
			http.Error(w, "failed to load product form", http.StatusInternalServerError)
		}

	}
}

func productUpdateHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		var errors []string
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Fatal(err)
			return
		}
		name := r.PostFormValue("name")
		priceStr := r.PostFormValue("price")
		stockStr := r.PostFormValue("stock")

		price, err := strconv.Atoi(priceStr)
		if err != nil || price <= 0 {
			errors = append(errors, "invalid price input")
		}

		stock, err := strconv.Atoi(stockStr)
		if err != nil || stock <= 0 {
			errors = append(errors, "invalid price input")
		}

		if strings.TrimSpace(name) == "" {
			errors = append(errors, "Name could not be empty")
		}

		product := Product{
			ID:    id,
			Name:  name,
			Price: price,
			Stock: stock,
		}

		if len(errors) > 0 {
			productForm := ProductForm{
				Heading: "Edit Products",
				Product: product,
				Errors:  errors,
			}

			data := map[string]any{
				"Title":       "Products",
				"ProductForm": productForm,
			}

			if err := tplProductForm.ExecuteTemplate(w, "layout", data); err != nil {
				log.Printf("failed to load product form: %s\n", err.Error())
				http.Error(w, "failed to load product form", http.StatusInternalServerError)
			}
			return
		}

		_, err = updateProduct(r.Context(), db, product)
		http.Redirect(w, r, "/products", http.StatusSeeOther)
	}
}

func productSimpanHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		var errors []string

		name := r.PostFormValue("name")
		priceStr := r.PostFormValue("price")
		stockStr := r.PostFormValue("stock")

		price, err := strconv.Atoi(priceStr)
		if err != nil || price <= 0 {
			errors = append(errors, "invalid price input")
		}

		stock, err := strconv.Atoi(stockStr)
		if err != nil || stock <= 0 {
			errors = append(errors, "invalid stock input")
		}

		if strings.TrimSpace(name) == "" {
			errors = append(errors, "Name could not be empty")
		}

		if len(errors) > 0 {
			productForm := ProductForm{
				Heading: "Add Products",
				Action:  "/products",
				Product: Product{
					Name:  name,
					Price: price,
					Stock: stock,
				},
				Errors: errors,
			}

			data := map[string]any{
				"Title":       "Products",
				"ProductForm": productForm,
			}

			if err := tplProductForm.ExecuteTemplate(w, "layout", data); err != nil {
				log.Printf("failed to load product form: %s\n", err.Error())
				http.Error(w, "failed to load product form", http.StatusInternalServerError)
			}
			return
		}

		product := Product{
			Name:  name,
			Price: price,
			Stock: stock,
		}

		_, err = addProduct(r.Context(), db, product)
		http.Redirect(w, r, "/products", http.StatusSeeOther)
	}
}

func productHardDeleteHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Fatal("invalid product id")
			return
		}

		deleteErr := deleteProduct(r.Context(), db, id)
		if deleteErr != nil {
			log.Fatal(deleteErr)
			return
		}

		http.Redirect(w, r, "/products", http.StatusSeeOther)
	}
}

func productDeactivateHandler(db  *sql.DB) http.HandlerFunc {
	return  func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			log.Fatal("invalid product id")
			return 
		}

		res := deactivateProduct(r.Context(), db, id)
		if res != nil {
			log.Fatal("failed to deactivate product")
			return 
		}

		http.Redirect(w, r, "/products", http.StatusSeeOther)
	}
}

func announcementHandler(w http.ResponseWriter, r *http.Request) {
	score := 90
	is_pass := false

	if score > 70 {
		is_pass = true
	}

	data := map[string]any{
		"Title":      "Announcement",
		"Is_pass": is_pass,
	}
	if err := tplAnnouncement.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("failed to render Announcement: %s", err.Error())
		http.Error(w, "failed to render Announcement", http.StatusInternalServerError)
	}
}

func getProductById(ctx context.Context, db *sql.DB, id int) (*Product, error) {
	query := `
	SELECT id, nama, harga, stok FROM products WHERE id = ?`

	var product Product
	err := db.QueryRowContext(ctx, query, id).Scan(&product.ID, &product.Name, &product.Price, &product.Stock)
	if err != nil {
		log.Printf("getProductById error: %s", err)
		return nil, err
	}

	return &product, err
}

func getAllProduct(ctx context.Context, db *sql.DB) ([]Product, error) {
	query := `
	SELECT id, nama, harga, stok FROM products WHERE aktif = 1 ORDER BY id`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("getAllProduct error: %s", err)
		return nil, err
	}

	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Stock); err != nil {
			log.Printf("getAllProduct scan error: %s", err)
			return nil, err
		}

		products = append(products, p)
	}

	return products, rows.Err()
}

func addProduct(ctx context.Context, db *sql.DB, p Product) (int64, error) {
	query := `
	INSERT INTO products (nama, harga, stok, keterangan) VALUES(?, ?, ?, ?)`

	rows, err := db.ExecContext(ctx, query, p.Name, p.Price, p.Stock, p.Description)
	if err != nil {
		return 0, err
	}
	id, err := rows.LastInsertId()

	return id, err
}

func updateProduct(ctx context.Context, db *sql.DB, p Product) (int64, error) {
	query := `
	UPDATE products set nama=?, harga=?, stok=?, keterangan=? WHERE id=?`

	res, err := db.ExecContext(ctx, query, p.Name, p.Price, p.Stock, p.Description, p.ID)
	if err != nil {
		log.Printf("updateProduct failed: %s\n", err)
		return 0, err
	}
	rows, err := res.RowsAffected()

	return rows, err
}

func deleteProduct(ctx context.Context, db *sql.DB, id int) error {
	query := `
	DELETE FROM products where id = ?`

	_, err := db.ExecContext(ctx, query, id)
	if err != nil {
		log.Fatalf("deleteProducts error: %s", err)
	}

	return err
}

func deactivateProduct(ctx context.Context, db *sql.DB, id int) error {
	query := `
	UPDATE products SET aktif = 0 where id = ?`

	_, err := db.ExecContext(ctx, query, id)
	if err != nil {
		log.Fatalf("deactivateProducts error: %s", err)
	}

	return err
}