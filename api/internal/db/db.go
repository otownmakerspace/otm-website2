// Package db owns the SQLite/libsql database: opening connections,
// running migrations, seeding test data, and the user repository.
package db

import (
	"database/sql"
	"fmt"
	"log"

	stripepkg "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/customer"
	stripesub "github.com/stripe/stripe-go/v82/subscription"
	_ "github.com/tursodatabase/go-libsql"
	"golang.org/x/crypto/bcrypt"

	"github.com/iustin94/makerspace/api/internal/config"
	"github.com/iustin94/makerspace/api/internal/user"
)

// Open returns a *sql.DB pointed at the libsql file given in cfg.DBPath.
// The connection is lazy; first query forces the open.
func Open(cfg config.BackendConf) (*sql.DB, error) {
	return sql.Open("libsql", "file:"+cfg.DBPath)
}

// Migrate creates the user and password_reset_tokens tables if they do not exist.
func Migrate(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS user(
		id INTEGER PRIMARY KEY,
		email TEXT UNIQUE,
		name TEXT,
		phone TEXT,
		password BLOB,
		active INT,
		customer_id TEXT);`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS password_reset_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL,
		token TEXT NOT NULL,
		expires_at DATETIME NOT NULL
	);`)
	if err != nil {
		log.Print("db: table creation failed")
		return err
	}
	log.Print("db: tables ready")
	return nil
}

// SeedTest creates a test user with a real Stripe customer + subscription
// (test-mode pm_card_visa). Idempotent: re-running on an existing real seed
// row is a no-op; the synthetic legacy "cus_test_000" row is replaced.
func SeedTest(db *sql.DB, cfg config.Config) {
	const testEmail = "test@test.com"

	if cfg.Stripe.Key == "" || cfg.Stripe.Key == "REPLACE_ME" {
		log.Print("seed: skipping — STRIPE_KEY not configured")
		return
	}
	if cfg.Stripe.PriceID == "" {
		log.Print("seed: skipping — STRIPE_PRICE_ID not configured")
		return
	}

	var existingCustomerID string
	err := db.QueryRow("SELECT customer_id FROM user WHERE email = ?", testEmail).Scan(&existingCustomerID)
	if err == nil {
		if existingCustomerID == "cus_test_000" {
			log.Print("seed: removing stale synthetic test user for re-seed")
			db.Exec("DELETE FROM user WHERE email = ?", testEmail)
		} else {
			return // already seeded with a real customer
		}
	} else if err != sql.ErrNoRows {
		log.Print("seed: failed to check existing test user: ", err)
		return
	}

	cust, err := customer.New(&stripepkg.CustomerParams{
		Email:         stripepkg.String(testEmail),
		Name:          stripepkg.String("Test User"),
		PaymentMethod: stripepkg.String("pm_card_visa"),
		InvoiceSettings: &stripepkg.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripepkg.String("pm_card_visa"),
		},
	})
	if err != nil {
		log.Print("seed: failed to create Stripe test customer: ", err)
		return
	}

	if _, err := stripesub.New(&stripepkg.SubscriptionParams{
		Customer: stripepkg.String(cust.ID),
		Items: []*stripepkg.SubscriptionItemsParams{
			{Price: stripepkg.String(cfg.Stripe.PriceID)},
		},
	}); err != nil {
		log.Print("seed: failed to create Stripe test subscription: ", err)
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		log.Print("seed: bcrypt failed: ", err)
		return
	}
	_, err = db.Exec("INSERT INTO user (email, name, phone, password, active, customer_id) VALUES (?, ?, ?, ?, ?, ?)",
		testEmail, "Test User", "", hashed, 1, cust.ID)
	if err != nil {
		log.Print("seed: insert failed: ", err)
		return
	}
	log.Printf("seed: test user %s / password (Stripe customer: %s)", testEmail, cust.ID)
}

// AddUser inserts a user row. The caller hashes Password before calling.
func AddUser(db *sql.DB, u user.User) error {
	_, err := db.Exec("INSERT INTO user (email, name, phone, password, active, customer_id) VALUES (?, ?, ?, ?, ?, ?)",
		u.Email, u.Name, u.Phone, u.Password, u.Active, u.CustomerID)
	return err
}

// EmailExists reports whether any user row uses the given email.
func EmailExists(db *sql.DB, email string) (bool, error) {
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM user WHERE email = ?", email).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

// GetByCustomerID returns the user row for the given Stripe customer ID.
func GetByCustomerID(db *sql.DB, customerID string) (user.User, error) {
	var u user.User
	err := db.QueryRow(
		"SELECT email, name, phone, password, active, customer_id FROM user WHERE customer_id = ?",
		customerID,
	).Scan(&u.Email, &u.Name, &u.Phone, &u.Password, &u.Active, &u.CustomerID)
	return u, err
}

// ListActive returns every active member (email + name only), ordered by
// email. Used by cmd/googlebackfill to seed the Google Contacts mailing-list
// label from the membership database.
func ListActive(db *sql.DB) ([]user.User, error) {
	rows, err := db.Query("SELECT email, name FROM user WHERE active = 1 ORDER BY email")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []user.User
	for rows.Next() {
		var u user.User
		if err := rows.Scan(&u.Email, &u.Name); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// GetByEmail returns the user row for the given email.
func GetByEmail(db *sql.DB, email string) (user.User, error) {
	var u user.User
	err := db.QueryRow(
		"SELECT email, name, phone, password, active, customer_id FROM user WHERE email = ?",
		email,
	).Scan(&u.Email, &u.Name, &u.Phone, &u.Password, &u.Active, &u.CustomerID)
	return u, err
}

// UpdateProfile updates name and phone for the user identified by email.
func UpdateProfile(db *sql.DB, email, name, phone string) error {
	_, err := db.Exec("UPDATE user SET name = ?, phone = ? WHERE email = ?", name, phone, email)
	if err != nil {
		return fmt.Errorf("db: UpdateProfile: %w", err)
	}
	return nil
}

// UpdatePassword writes a new bcrypt hash for the user identified by email.
func UpdatePassword(db *sql.DB, email string, passwordHash []byte) error {
	_, err := db.Exec("UPDATE user SET password = ? WHERE email = ?", passwordHash, email)
	if err != nil {
		return fmt.Errorf("db: UpdatePassword: %w", err)
	}
	return nil
}

// CountActive returns the number of users with active = 1.
func CountActive(db *sql.DB) (int, error) {
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM user WHERE active = 1").Scan(&n); err != nil {
		log.Printf("db: CountActive query error: %v", err)
		return 0, err
	}
	return n, nil
}

// LookupForLogin reads the password hash + customer_id for a login attempt.
// Returns sql.ErrNoRows for unknown emails so callers can distinguish.
func LookupForLogin(db *sql.DB, email string) (passwordHash []byte, customerID string, err error) {
	err = db.QueryRow("SELECT password, customer_id FROM user WHERE email = ?", email).
		Scan(&passwordHash, &customerID)
	return
}

// SetActive flips the active flag for a customer (used by subscription-deleted webhook).
func SetActive(db *sql.DB, customerID string, active bool) error {
	_, err := db.Exec("UPDATE user SET active = ? WHERE customer_id = ?", active, customerID)
	if err != nil {
		return fmt.Errorf("db: SetActive: %w", err)
	}
	return nil
}

// ReactivateReturning rebinds a returning member (identified by email) to a
// Stripe customer ID and marks them active. Used by checkout fulfillment when a
// lapsed member re-subscribes — including the self-heal case where their stored
// customer was gone and checkout minted a fresh one, so customer_id changes.
// Keyed by email (not customer_id) precisely because the customer_id may be the
// thing that's changing. Idempotent: re-running with the same values is a no-op.
func ReactivateReturning(db *sql.DB, email, customerID string) error {
	_, err := db.Exec("UPDATE user SET customer_id = ?, active = 1 WHERE email = ?", customerID, email)
	if err != nil {
		return fmt.Errorf("db: ReactivateReturning: %w", err)
	}
	return nil
}
