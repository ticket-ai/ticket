//budget is only invoked by the stripe webhook and llm.go
//keep track of users? db?
	// schema litterally just uuid and account balance
	// sqlite?
//stripe webhook function updating the db
//accessor methods to update the balance based off of usage

package budget

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/mattn/go-sqlite3"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/webhook"
)

type BudgetManager struct {
	db                *sql.DB
	secret            string
	
	stmtUpdateBalance *sql.Stmt
	stmtGetBalance    *sql.Stmt
	stmtDeduct        *sql.Stmt
	stmtInsertEvent   *sql.Stmt
}

func NewBudgetManager(dbPath, stripeWebhookSecret string) (*BudgetManager, error) {

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	
	schema := `
CREATE TABLE IF NOT EXISTS user_balances (
    uuid TEXT PRIMARY KEY,
    balance_cents INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS processed_events (
    event_id TEXT PRIMARY KEY
);
`
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}

	

	stmtUpdate, err := db.Prepare(`
    INSERT INTO user_balances(uuid, balance_cents) VALUES(?, ?)
    ON CONFLICT(uuid) DO UPDATE SET balance_cents = balance_cents + excluded.balance_cents
`) 

	if err != nil {
		return nil, fmt.Errorf("prepare update: %w", err)
	}

	stmtGet, err := db.Prepare(`SELECT balance_cents FROM user_balances WHERE uuid = ?`)

	if err != nil {
		return nil, fmt.Errorf("prepare get: %w", err)
	}

	stmtDeduct, err := db.Prepare(`
    UPDATE user_balances 
    SET balance_cents = balance_cents - ? 
    WHERE uuid = ? AND balance_cents >= ?
`)

	if err != nil {

		return nil, fmt.Errorf("prepare deduct: %w", err)
	}

	stmtInsertEvent, err := db.Prepare(`INSERT INTO processed_events(event_id) VALUES(?)`)
	if err != nil {
		return nil, fmt.Errorf("prepare event: %w", err)
	}

	return &BudgetManager{
		db:                db,
		secret:            stripeWebhookSecret,
		
		stmtUpdateBalance: stmtUpdate,
		stmtGetBalance:    stmtGet,
		stmtDeduct:        stmtDeduct,
		stmtInsertEvent:   stmtInsertEvent,
	}, nil

}

func (bm *BudgetManager) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()
	payload, err := io.ReadAll(r.Body)

	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}

	evt, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), bm.secret)
	if err != nil {
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	type evtData struct{ UUID string; AmountCents int64 }
	var d evtData

	switch evt.Type {

	case "payment_intent.succeeded":
		var pi stripe.PaymentIntent

		if err := json.Unmarshal(evt.Data.Raw, &pi); err != nil {
			log.Printf("unmarshal payment_intent: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		u, ok := pi.Metadata["uuid"]; if !ok {
			http.Error(w, "missing uuid", http.StatusBadRequest)
			return
		}

		d = evtData{UUID: u, AmountCents: pi.Amount}



	case "charge.refunded":
		var ch stripe.Charge
		if err := json.Unmarshal(evt.Data.Raw, &ch); err != nil {
			log.Printf("unmarshal charge: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		u, ok := ch.Metadata["uuid"]; if !ok {
			http.Error(w, "missing uuid", http.StatusBadRequest)
			return
		}

		d = evtData{UUID: u, AmountCents: -ch.AmountRefunded}


	default:
		log.Printf("unhandled event: %s", evt.Type)
		w.WriteHeader(http.StatusOK)
		return
	}

	tx, err := bm.db.Begin()

	if err != nil {
		log.Printf("tx begin: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	res, err := tx.Stmt(bm.stmtInsertEvent).Exec(evt.ID)

	if err != nil {
		tx.Rollback()
		log.Printf("insert event: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if n, _ := res.RowsAffected(); n == 0 {
		tx.Rollback()
		w.WriteHeader(http.StatusOK)
		return
	}

	if _, err := tx.Stmt(bm.stmtUpdateBalance).Exec(d.UUID, d.AmountCents); err != nil {
		tx.Rollback()
		log.Printf("update balance: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("tx commit: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (bm *BudgetManager) GetBalance(uuid string) (int64, error) {
	var cents int64
	err := bm.stmtGetBalance.QueryRow(uuid).Scan(&cents)

	if err == sql.ErrNoRows {
		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	return cents, nil
}

func (bm *BudgetManager) DeductFromBalance(uuid string, amountCents int64) error {
	res, err := bm.stmtDeduct.Exec(amountCents, uuid, amountCents)

	if err != nil {
		return err
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("insufficient funds")
	}

	return nil

}

func (bm *BudgetManager) Close() error {
	bm.stmtUpdateBalance.Close()
	bm.stmtGetBalance.Close()
	bm.stmtDeduct.Close()
	bm.stmtInsertEvent.Close()
	return bm.db.Close()
}
