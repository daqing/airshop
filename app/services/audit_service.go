package services

import (
	"log"

	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

// Audit records a high-risk admin operation. Best-effort by design: a failed
// audit write is logged but never fails the operation it describes.
//
// Instrumented points (T9.5, A-light): order refund, shipping, shipment
// status changes, coupon enable/disable, product activate/deactivate and
// price changes. Query via psql or the REPL — there is no admin UI yet.
func Audit(adminID int64, action, entity, entityID, detail string) {
	_, err := repo.CreateFrom[models.AdminLog](sql.H{
		"admin_id":  adminID,
		"action":    action,
		"entity":    entity,
		"entity_id": entityID,
		"detail":    detail,
	})
	if err != nil {
		log.Printf("audit write failed (action=%s entity=%s id=%s): %v", action, entity, entityID, err)
	}
}

// AdminLogs returns recent audit entries, newest first.
func AdminLogs(limit int) ([]*models.AdminLog, error) {
	b := sql.Select("*").From("admin_logs").OrderBy("id DESC").Limit(limit)
	return repo.Find[models.AdminLog](repo.CurrentDB(), b)
}
