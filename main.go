package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/bxcodec/faker/v4"
	_ "github.com/mattn/go-sqlite3"
)

type Product struct {
	ID          int
	Name        string
	Description string
	Price       float64
	ImageURL    string
	Category    string
}

var db *sql.DB

func main() {
	var err error
	db, err = ConnectProducts("./sqlite/dataUX.db")
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// Seed des produits (uniquement une fois ou lorsque nécessaire)
	if err := SeedProducts(db); err != nil {
		log.Fatalf("Error seeding products: %v", err)
	}

	// Servir les fichiers statiques
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Route principale
	http.HandleFunc("/home", Home)

	fmt.Println("Server started at http://localhost:8081/home")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

// SeedProducts insère des produits réalistes dans la base de données
func SeedProducts(db *sql.DB) error {
	// Préparer la requête SQL pour insérer les produits
	stmt, err := db.Prepare(`INSERT INTO products (name, description, price, image_url) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("error preparing insert statement: %v", err)
	}
	defer stmt.Close()

	// Initialiser le générateur de nombres aléatoires
	rand.Seed(time.Now().UnixNano())

	// Liste de catégories de produits
	categories := []string{"Téléphone", "Ordinateur", "Accessoire", "Chaussure", "Montre", "Vêtement"}

	// Insérer 600 produits réalistes
	for i := 1; i <= 2000; i++ {
		category := categories[rand.Intn(len(categories))]               // Choisir une catégorie au hasard
		productName := fmt.Sprintf("%s %s", category, faker.FirstName()) // Nom réaliste
		description := fmt.Sprintf("Découvrez notre %s de qualité supérieure pour toutes vos exigences.", category)
		price := rand.Float64()*900 + 100 // Prix entre 100 et 1000
		imageURL := fmt.Sprintf("https://source.unsplash.com/200x200/?%s", category)

		// Exécuter l'insertion
		_, err := stmt.Exec(productName, description, price, imageURL)
		if err != nil {
			log.Printf("Error inserting product %d: %v", i, err)
		} else {
			fmt.Printf("Inserted product %d: %s\n", i, productName)
		}
	}

	fmt.Println("Seeding completed successfully with realistic data.")
	return nil
}

func ConnectProducts(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(255) NOT NULL,
			description TEXT NOT NULL,
			price DECIMAL(10, 2) DEFAULT 0.00,
			image_url VARCHAR(2083) DEFAULT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("error creating table: %v", err)
	}

	return db, nil
}

func Home(w http.ResponseWriter, r *http.Request) {
	page := r.URL.Query().Get("page")
	if page == "" {
		page = "1" // Valeur par défaut pour la première page
	}

	products, nextPage, prevPage, err := getPagedProducts(db, page)
	if err != nil {
		log.Printf("Error retrieving products: %v", err)
		http.Error(w, "Error retrieving products", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Products":     products,
		"NextPage":     nextPage,
		"PreviousPage": prevPage,
	}

	renderTemplate(w, "./tmpl/home.html", data)
}

func getPagedProducts(db *sql.DB, page string) ([]Product, string, string, error) {
	const itemsPerPage = 24 // 4 lignes avec 4 produits par ligne
	pageInt, err := strconv.Atoi(page)
	if err != nil {
		pageInt = 1
	}

	offset := (pageInt - 1) * itemsPerPage

	rows, err := db.Query("SELECT id, name, description, price, image_url FROM products LIMIT ? OFFSET ?", itemsPerPage, offset)
	if err != nil {
		return nil, "", "", fmt.Errorf("error querying products: %v", err)
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var prod Product
		if err := rows.Scan(&prod.ID, &prod.Name, &prod.Description, &prod.Price, &prod.ImageURL); err != nil {
			return nil, "", "", fmt.Errorf("error scanning products: %v", err)
		}
		products = append(products, prod)
	}

	var nextPage, prevPage string
	if len(products) == itemsPerPage {
		nextPage = fmt.Sprintf("%d", pageInt+1)
	}
	if pageInt > 1 {
		prevPage = fmt.Sprintf("%d", pageInt-1)
	}

	return products, nextPage, prevPage, nil
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	t, err := template.ParseFiles(tmpl)
	if err != nil {
		log.Printf("Error parsing template %s: %v", tmpl, err)
		http.Error(w, "Error loading page", http.StatusInternalServerError)
		return
	}
	if err := t.Execute(w, data); err != nil {
		log.Printf("Error executing template %s: %v", tmpl, err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}
