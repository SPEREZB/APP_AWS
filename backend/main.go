package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

// Define el tipo Student
type Student struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Age      int    `json:"age"`
	Semestre string `json:"semestre"`
}

var conn *pgx.Conn

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

	// Crear una nueva aplicación Fiber
	app := fiber.New()
	app.Use(logger.New())

	// Servir archivos estáticos generados por React
	app.Static("/", "./dist")

	// Definir rutas CRUD
	app.Get("/api/students", getStudents)
	app.Get("/api/students/:id", getStudentByID)
	app.Post("/api/students", createStudent)
	app.Put("/api/students/:id", updateStudent)
	app.Delete("/api/students/:id", deleteStudent)

	// Iniciar el servidor en el puerto 3000
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	// Obtener el host desde una variable de entorno, o usar el dominio por defecto
	host := os.Getenv("HOST")
	if host == "" {
		host = "appgofinal-env.eba-r4upddvy.us-east-1.elasticbeanstalk.com"
	}

	// Construir la dirección de escucha
	address := fmt.Sprintf("%s:%s", host, port)
	log.Fatal(app.Listen(address))
}

// Obtener todos los estudiantes
func getStudents(c *fiber.Ctx) error {
	var students []Student

	rows, err := conn.Query(context.Background(), "SELECT id_student, name, age, semestre FROM students")
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to query database"})
	}
	defer rows.Close()

	for rows.Next() {
		var student Student
		if err := rows.Scan(&student.ID, &student.Name, &student.Age, &student.Semestre); err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to parse query result"})
		}
		students = append(students, student)
	}

	return c.JSON(students)
}

// Obtener un estudiante por ID
func getStudentByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var student Student

	err := conn.QueryRow(context.Background(), "SELECT id, name, age, semestre FROM students WHERE id_student=$1", id).Scan(&student.ID, &student.Name, &student.Age, &student.Semestre)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "estudiante no encontrado"})
	}

	return c.JSON(student)
}

// Crear un nuevo estudiante
func createStudent(c *fiber.Ctx) error {
	var student Student

	if err := c.BodyParser(&student); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	var id int
	err := conn.QueryRow(context.Background(), "INSERT INTO students (name, age, semestre) VALUES ($1, $2, $3) RETURNING id_student", student.Name, student.Age, student.Semestre).Scan(&id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Error al crear estudiante"})
	}

	student.ID = id
	return c.JSON(student)
}

// Actualizar un estudiante
func updateStudent(c *fiber.Ctx) error {
	id := c.Params("id")
	var student Student

	// Parsear el cuerpo de la solicitud
	if err := c.BodyParser(&student); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Convertir el ID de la URL a un entero
	studentID, err := strconv.Atoi(id)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid student ID"})
	}

	// Establecer el ID del estudiante para la actualización
	student.ID = studentID

	// Ejecutar la consulta de actualización en la base de datos
	_, err = conn.Exec(context.Background(),
		"UPDATE students SET name=$1, age=$2, semestre=$3 WHERE id_student=$4",
		student.Name, student.Age, student.Semestre, student.ID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Error al actualizar estudiante"})
	}

	// Devolver la respuesta con el estudiante actualizado
	return c.JSON(student)
}

// Eliminar un estudiante
func deleteStudent(c *fiber.Ctx) error {
	id := c.Params("id")

	result, err := conn.Exec(context.Background(), "DELETE FROM students WHERE id_student=$1", id)
	if err != nil || result.RowsAffected() == 0 {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Error al borrar estudiante"})
	}

	return c.SendStatus(http.StatusNoContent)
}
