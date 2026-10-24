package main

import (
	"cmp"
	"context"
	"crypto/rand"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store is the relay's SQL. now is epoch seconds from the caller, so the
// tests set the clock.
type Store struct {
	db *pgxpool.Pool
}

// Message is a stored text as a consumer sees it.
type Message struct {
	ID         int64   `json:"id"`
	ReceivedTS float64 `json:"receivedTs"`
	From       string  `json:"from"`
	To         string  `json:"to"`
	Text       string  `json:"text"`
}

func epochNow() float64 {
	return float64(time.Now().UnixNano()) / 1e9
}

func (s *Store) Insert(ctx context.Context, queue string, in telnyxInbound, rawBody []byte, now float64) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO inbox_message (queue_name, received_ts, provider,
		    event_type, from_phone, to_phone, body_text, raw_body,
		    receive_count)
		VALUES ($1, $2, 'telnyx', $3, $4, $5, $6, $7, 0)
		RETURNING inbox_message_id`,
		queue, now, in.EventType, in.From, in.To, in.Text, string(rawBody),
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to store text: %w", err)
	}
	return id, nil
}

// Peek is up to limit receivable messages (not done, not under a live
// lease), oldest first, without changing them.
func (s *Store) Peek(ctx context.Context, queue string, limit int, now float64) ([]Message, error) {
	rows, err := s.db.Query(ctx, `
		SELECT inbox_message_id, received_ts, from_phone, to_phone, body_text
		FROM inbox_message
		WHERE queue_name = $1
		  AND done_ts IS NULL
		  AND (lease_until_ts IS NULL OR lease_until_ts < $2)
		ORDER BY inbox_message_id
		LIMIT $3`, queue, now, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to peek: %w", err)
	}
	return scanMessageList(rows)
}

// Receive leases up to limit receivable messages, oldest first, under one
// new lease id. Rows another receive holds locked are skipped, not waited
// for, so two consumers never get the same row from one pass.
func (s *Store) Receive(ctx context.Context, queue string, limit int, leaseSec int, now float64) (string, []Message, error) {
	leaseID := rand.Text()
	rows, err := s.db.Query(ctx, `
		WITH picked AS (
		    SELECT inbox_message_id
		    FROM inbox_message
		    WHERE queue_name = $1
		      AND done_ts IS NULL
		      AND (lease_until_ts IS NULL OR lease_until_ts < $2)
		    ORDER BY inbox_message_id
		    LIMIT $3
		    FOR UPDATE SKIP LOCKED
		)
		UPDATE inbox_message m
		SET lease_id = $4,
		    lease_until_ts = $2 + $5,
		    receive_count = m.receive_count + 1
		FROM picked
		WHERE m.inbox_message_id = picked.inbox_message_id
		RETURNING m.inbox_message_id, m.received_ts, m.from_phone,
		    m.to_phone, m.body_text`,
		queue, now, limit, leaseID, leaseSec)
	if err != nil {
		return "", nil, fmt.Errorf("failed to receive: %w", err)
	}
	messageList, err := scanMessageList(rows)
	if err != nil {
		return "", nil, err
	}
	// RETURNING has no order of its own.
	slices.SortFunc(messageList, func(a, b Message) int {
		return cmp.Compare(a.ID, b.ID)
	})
	return leaseID, messageList, nil
}

// Ack marks done the ids in idList that this queue's live lease leaseID
// holds; an expired lease acks nothing (the row may be with another
// consumer by now). It returns how many it marked.
func (s *Store) Ack(ctx context.Context, queue string, leaseID string, idList []int64, now float64) (int64, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE inbox_message
		SET done_ts = $4
		WHERE queue_name = $1
		  AND lease_id = $2
		  AND inbox_message_id = ANY($3)
		  AND lease_until_ts >= $4
		  AND done_ts IS NULL`,
		queue, leaseID, idList, now)
	if err != nil {
		return 0, fmt.Errorf("failed to ack: %w", err)
	}
	return tag.RowsAffected(), nil
}

// DeleteDoneBefore removes messages acked before cutoff; undone ones stay
// however old.
func (s *Store) DeleteDoneBefore(ctx context.Context, cutoff float64) (int64, error) {
	tag, err := s.db.Exec(ctx, `
		DELETE FROM inbox_message
		WHERE done_ts < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old messages: %w", err)
	}
	return tag.RowsAffected(), nil
}

func scanMessageList(rows pgx.Rows) ([]Message, error) {
	defer rows.Close()
	messageList := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ReceivedTS, &m.From, &m.To, &m.Text); err != nil {
			return nil, err
		}
		messageList = append(messageList, m)
	}
	return messageList, rows.Err()
}
