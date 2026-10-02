package main

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type User struct {
	Name  string
	Email string
	Age   int
}

type Product struct {
	Name  string
	Price int
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
	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))
	mux.HandleFunc("GET /", homeHandler)
	mux.HandleFunc("GET /announcements", announcementHandler)
	mux.HandleFunc("GET /products", productHandler)
	mux.HandleFunc("GET /products/form", productFormHandler)
	mux.HandleFunc("POST /products/simpan", productSimpanHandler)

	http.ListenAndServe(":8000", mux)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"title": "Home",
		"judul": "Website Heading",
		"hewan": []string{"shark", "whale", "dolphin"},
		"users": UserData,
	}
	if err := tplHome.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("failed to render Home: %s", err.Error())
		http.Error(w, "failed to render Home", http.StatusInternalServerError)
	}
}

func productHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"title":    "Products",
		"products": ProductData,
	}

	if err := tplProduct.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("failed to render Product: %s\n", err.Error())
		http.Error(w, "failed to load Products", http.StatusInternalServerError)
	}
}

func productFormHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"title": "Products",
	}
	if err := tplProductForm.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("failed to load product form: %s\n", err.Error())
		http.Error(w, "failed to load product form", http.StatusInternalServerError)
	}
}

func productSimpanHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var errors []string

	name := r.PostFormValue("name")
	priceStr := r.PostFormValue("price")
	price, err := strconv.Atoi(priceStr)
	if err != nil {
		errors = append(errors, "invalid price input")
	}

	if strings.TrimSpace(name) == "" {
		errors = append(errors, "Name could not be empty")
	}

	if len(errors) > 0 {
		data := map[string]any{
			"name":   name,
			"price":  price,
			"priceStr": priceStr,
			"errors": errors,
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
	}

	ProductData = append(ProductData, product)
	http.Redirect(w, r, "/products", http.StatusSeeOther)
}

func announcementHandler(w http.ResponseWriter, r *http.Request) {
	nilai := 90
	is_kkm_oke := false

	if nilai > 70 {
		is_kkm_oke = true
	}

	data := map[string]any{
		"title":      "Announcement",
		"is_kkm_oke": is_kkm_oke,
	}
	if err := tplAnnouncement.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("failed to render Announcement: %s", err.Error())
		http.Error(w, "failed to render Announcement", http.StatusInternalServerError)
	}
}
