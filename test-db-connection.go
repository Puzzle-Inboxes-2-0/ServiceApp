package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	passwords := []string{"", "postgres", "admin", "password", "root", "changeme123"}
	
	host := "localhost"
	port := "5433"  // Docker port
	user := "postgres"
	dbname := "mydb"  // Docker database name
	
	fmt.Println("Testing PostgreSQL passwords...")
	fmt.Println("=====================================")
	
	for _, password := range passwords {
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, dbname)
		
		fmt.Printf("Trying password: '%s' ... ", password)
		
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			fmt.Printf("❌ Open failed: %v\n", err)
			continue
		}
		
		err = db.Ping()
		db.Close()
		
		if err == nil {
			fmt.Printf("✅ SUCCESS!\n")
			fmt.Println("=====================================")
			fmt.Printf("\nFOUND WORKING PASSWORD: '%s'\n", password)
			
			// Save to file
			os.WriteFile("/tmp/found_password.txt", []byte(password), 0644)
			os.Exit(0)
		} else {
			fmt.Printf("❌ %v\n", err)
		}
	}
	
	fmt.Println("=====================================")
	fmt.Println("❌ None of the passwords worked")
	fmt.Println("\nYour local PostgreSQL 17 requires a different password.")
	os.Exit(1)
}

