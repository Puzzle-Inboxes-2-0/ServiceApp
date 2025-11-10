package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// DB is the global database connection
var DB *sql.DB

// User represents a user in the database
type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Connect establishes a connection to the PostgreSQL database
func Connect(dsn string, log *logrus.Logger) error {
	var err error

	log.WithFields(logrus.Fields{
		"action": "database_connect",
	}).Info("Attempting to connect to PostgreSQL database")

	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.WithFields(logrus.Fields{
			"action": "database_connect",
			"error":  err.Error(),
		}).Error("Failed to open database connection")
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Set connection pool settings
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err = DB.Ping(); err != nil {
		log.WithFields(logrus.Fields{
			"action": "database_ping",
			"error":  err.Error(),
		}).Error("Failed to ping database")
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.WithFields(logrus.Fields{
		"action": "database_connect",
	}).Info("Successfully connected to PostgreSQL database")

	return nil
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// GetAllUsers retrieves all users from the database
func GetAllUsers() ([]User, error) {
	query := "SELECT id, username, email, created_at FROM users ORDER BY id"
	rows, err := DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// CreateUser creates a new user in the database
func CreateUser(username, email string) (*User, error) {
	query := "INSERT INTO users (username, email) VALUES ($1, $2) RETURNING id, username, email, created_at"
	
	var user User
	err := DB.QueryRow(query, username, email).Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

// GetUserByID retrieves a user by ID
func GetUserByID(id int) (*User, error) {
	query := "SELECT id, username, email, created_at FROM users WHERE id = $1"
	
	var user User
	err := DB.QueryRow(query, id).Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

