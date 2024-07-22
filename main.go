package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Define el tipo Student
type Student struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Last_name string `json:"last_name"`
	Age       int    `json:"age"`
	Semestre  string `json:"semestre"`
}

var conn *pgx.Conn

func getStudents(w http.ResponseWriter, r *http.Request) {
	rows, err := conn.Query(context.Background(), "SELECT id_student, name, last_name, age, semestre FROM students")
	if err != nil {
		http.Error(w, "Fallo al obtener el listado de estudiantes", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var students []Student
	for rows.Next() {
		var student Student
		if err := rows.Scan(&student.ID, &student.Name, &student.Last_name, &student.Age, &student.Semestre); err != nil {
			http.Error(w, "Fallo en el estado de la consulta", http.StatusInternalServerError)
			return
		}
		students = append(students, student)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(students)
}

func createStudent(w http.ResponseWriter, r *http.Request) {
	var student Student

	if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var id int
	err := conn.QueryRow(context.Background(), "INSERT INTO students (name, last_name, age, semestre) VALUES ($1, $2, $3, $4) RETURNING id_student", student.Name, student.Last_name, student.Age, student.Semestre).Scan(&id)
	if err != nil {
		http.Error(w, "Error creating student", http.StatusInternalServerError)
		return
	}

	student.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(student)
}

func updateStudent(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var student Student

	if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	studentID, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	student.ID = studentID

	_, err = conn.Exec(context.Background(),
		"UPDATE students SET name=$1, age=$2, semestre=$3, last_name=$5 WHERE id_student=$4",
		student.Name, student.Age, student.Semestre, student.ID, student.Last_name)
	if err != nil {
		http.Error(w, "Error updating student", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(student)
}

func deleteStudent(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	result, err := conn.Exec(context.Background(), "DELETE FROM students WHERE id_student=$1", id)
	if err != nil || result.RowsAffected() == 0 {
		http.Error(w, "Error deleting student", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func main() {
	// Cargar variables de entorno desde el archivo .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Obtener URL de la base de datos desde las variables de entorno
	databaseURL := os.Getenv("DATABASE_URL")

	// Crear una conexión a la base de datos
	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		log.Fatal("Unable to parse database URLss")
	}

	conn, err = pgx.ConnectConfig(context.Background(), config)
	if err != nil {
		log.Fatal("Unable to connect to database")
	}
	defer conn.Close(context.Background())

	fmt.Println("Connected to PostgreSQL database successfully!")

	r := mux.NewRouter()

	// Registrar los manejadores de cada endpoint
	r.HandleFunc("/api/students", getStudents).Methods("GET")
	r.HandleFunc("/api/students", createStudent).Methods("POST")
	r.HandleFunc("/api/students/{id:[0-9]+}", updateStudent).Methods("PUT")
	r.HandleFunc("/api/students/{id:[0-9]+}", deleteStudent).Methods("DELETE")

	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./dist")))

	// Iniciar el servidor en el puerto 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000" // Usar el puerto 8080 si no se especifica otro
	}

	fmt.Printf("Server is listening on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
