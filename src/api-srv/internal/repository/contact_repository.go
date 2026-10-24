package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"stillthere-api/internal/model"
)

// ContactRepository owns contact. contact_status and the invite / opt
// timestamps belong to the watchdog (texts in and out); the writes here
// set them only on create and when an invited contact's number changes.
type ContactRepository struct {
	db *pgxpool.Pool
}

// NewContactRepository creates the contact repository.
func NewContactRepository(db *pgxpool.Pool) *ContactRepository {
	return &ContactRepository{db: db}
}

const contactColumnList = `
	contact_id, device_id, contact_name, phone_e164, contact_status,
	notify_event_list, invite_sent_ts, opted_in_ts, opted_out_ts,
	security_context_id, create_ts, create_user_id, update_ts,
	update_user_id, delete_ts, delete_user_id`

func scanContact(row pgx.Row, contact *model.ContactModel) error {
	return row.Scan(
		&contact.ContactID, &contact.DeviceID, &contact.ContactName,
		&contact.PhoneE164, &contact.ContactStatus, &contact.NotifyEventList,
		&contact.InviteSentTS, &contact.OptedInTS, &contact.OptedOutTS,
		&contact.SecurityContext, &contact.CreateTS, &contact.CreateUserID,
		&contact.UpdateTS, &contact.UpdateUserID, &contact.DeleteTS,
		&contact.DeleteUserID,
	)
}

// duplicatePhoneMessage answers a number the device already has.
const duplicatePhoneMessage = "This device already has a contact with " +
	"that number"

// RetrieveContactList returns the tenant's live contacts of live devices:
// all, by id, of one device (deviceID > 0), or changed since afterUpdateTs.
func (r *ContactRepository) RetrieveContactList(
	ctx context.Context,
	idList []int,
	deviceID int,
	securityContext int,
	afterUpdateTs float64,
) ([]*model.ContactModel, error) {
	whereClause, argList := buildParentScopedWhere(
		securityContext, afterUpdateTs, "contact_id", idList,
		"device_id", deviceID)
	whereClause = withLiveParent(
		whereClause, "contact", "device_id", "device", "device_id")
	rowList, err := r.db.Query(ctx, `SELECT `+contactColumnList+`
		FROM contact`+whereClause+` ORDER BY contact_id`, argList...)
	if err != nil {
		return nil, fmt.Errorf("failed to query contacts: %w", err)
	}
	defer rowList.Close()

	contactList := make([]*model.ContactModel, 0)
	for rowList.Next() {
		var contact model.ContactModel
		if err := scanContact(rowList, &contact); err != nil {
			return nil, fmt.Errorf("failed to scan contact: %w", err)
		}
		contactList = append(contactList, &contact)
	}
	if err := rowList.Err(); err != nil {
		return nil, fmt.Errorf("error iterating contacts: %w", err)
	}
	return contactList, nil
}

// CreateContact adds an invited contact to a live device of the tenant;
// invite_sent_ts stays NULL, which is the watchdog's cue to text the
// invite. No such device is ErrNotFound; a number the device already has
// is ErrDuplicate.
func (r *ContactRepository) CreateContact(
	ctx context.Context, contact *model.ContactModel, securityContext int,
) (*model.ContactModel, error) {
	var created model.ContactModel
	err := scanContact(r.db.QueryRow(ctx, `
		INSERT INTO contact (
			device_id, contact_name, phone_e164, contact_status,
			notify_event_list, security_context_id, create_ts,
			create_user_id, update_ts, update_user_id
		)
		SELECT device_id, $2, $3, $4, $5, security_context_id,
		       $6, $7, $8, $9
		FROM device
		WHERE device_id = $1 AND security_context_id = $10
		  AND delete_user_id IS NULL
		RETURNING `+contactColumnList,
		contact.DeviceID, contact.ContactName, contact.PhoneE164,
		model.ContactStatusInvited, contact.NotifyEventList, contact.CreateTS,
		contact.CreateUserID, contact.UpdateTS, contact.UpdateUserID,
		securityContext,
	), &created)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if isUniqueViolation(err) {
		return nil, clientMessage(ErrDuplicate, duplicatePhoneMessage)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create contact: %w", err)
	}
	return &created, nil
}

// UpdateContact writes the name, number and event list under the
// optimistic lock and answers the stored row; a nil event list keeps the
// stored one. Only an invited contact's number can change: anyone who has
// replied agreed to texts at their number, so a new number is a new
// contact (ErrConflict with that message). A changed number owes a new
// invite (invite_sent_ts back to NULL).
func (r *ContactRepository) UpdateContact(
	ctx context.Context, contact *model.ContactModel, originalUpdateTS float64,
) (*model.ContactModel, error) {
	var updated model.ContactModel
	err := scanContact(r.db.QueryRow(ctx, `
		UPDATE contact
		SET contact_name = $1, phone_e164 = $2,
		    notify_event_list = COALESCE($3, notify_event_list),
		    invite_sent_ts =
		        CASE WHEN phone_e164 = $2 THEN invite_sent_ts END,
		    update_ts = $4, update_user_id = $5
		WHERE contact_id = $6 AND security_context_id = $7
		  AND update_ts = $8 AND delete_user_id IS NULL
		  AND (phone_e164 = $2 OR contact_status = $9)
		RETURNING `+contactColumnList,
		contact.ContactName, contact.PhoneE164, contact.NotifyEventList,
		contact.UpdateTS, contact.UpdateUserID, contact.ContactID,
		contact.SecurityContext, originalUpdateTS, model.ContactStatusInvited,
	), &updated)
	if isUniqueViolation(err) {
		return nil, clientMessage(ErrDuplicate, duplicatePhoneMessage)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, r.contactUpdateRefusal(ctx, contact, originalUpdateTS)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update contact: %w", err)
	}
	return &updated, nil
}

// contactUpdateRefusal explains an update that matched no row, with the
// write's tenant predicate: absent, stale, or a number change the
// contact's status refuses.
func (r *ContactRepository) contactUpdateRefusal(
	ctx context.Context, contact *model.ContactModel, originalUpdateTS float64,
) error {
	var updateTS float64
	err := r.db.QueryRow(ctx, `
		SELECT update_ts FROM contact
		WHERE contact_id = $1 AND security_context_id = $2
		  AND delete_user_id IS NULL`,
		contact.ContactID, contact.SecurityContext).Scan(&updateTS)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to check contact: %w", err)
	}
	if updateTS != originalUpdateTS {
		return ErrConflict
	}
	return clientMessage(ErrConflict, "This contact has replied, so "+
		"their number cannot change; add a new contact for the new number")
}

// DeleteContact soft deletes a contact under the optimistic lock.
func (r *ContactRepository) DeleteContact(
	ctx context.Context,
	contactID int,
	securityContext int,
	originalUpdateTS float64,
	deleteTS float64,
	deleteUserID int,
) error {
	result, err := r.db.Exec(ctx, `
		UPDATE contact SET delete_ts = $1, delete_user_id = $2
		WHERE contact_id = $3 AND security_context_id = $4
		  AND update_ts = $5 AND delete_user_id IS NULL`,
		deleteTS, deleteUserID, contactID, securityContext, originalUpdateTS)
	if err != nil {
		return fmt.Errorf("failed to delete contact: %w", err)
	}
	if result.RowsAffected() == 0 {
		return zeroRowOutcome(ctx, r.db, contactTable, contactPK, contactID,
			securityContext)
	}
	return nil
}
