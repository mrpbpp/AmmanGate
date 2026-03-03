package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ActionEngine handles security actions with approval workflow
type ActionEngine struct {
	db   *sql.DB
	hub  *WSHub
	auth *AuthManager
}

// NewActionEngine creates a new action engine
func NewActionEngine(db *sql.DB, hub *WSHub, auth *AuthManager) *ActionEngine {
	return &ActionEngine{
		db:   db,
		hub:  hub,
		auth: auth,
	}
}

// createPendingAction creates a pending action that requires approval
func (a *ActionEngine) createPendingAction(req ActionRequest) (string, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	actionID := "act_" + uuid.New().String()[:8]

	targetJSON, err := json.Marshal(req.Target)
	if err != nil {
		return "", fmt.Errorf("failed to marshal target: %w", err)
	}

	if req.TTLsec <= 0 {
		req.TTLsec = 1800 // 30 minutes default
	}

	_, err = a.db.Exec(`
		INSERT INTO actions (id, ts, action_type, target, ttl_sec, requested_by, status)
		VALUES (?, ?, ?, ?, ?, ?, 'pending')
	`, actionID, now, req.ActionType, string(targetJSON), req.TTLsec, req.RequestedBy)

	return actionID, err
}

// createApproval creates an approval challenge for an action
func (a *ActionEngine) createApproval(actionID string) (ApprovalChallenge, error) {
	now := time.Now().UTC()
	approvalID := "apr_" + uuid.New().String()[:8]
	expires := now.Add(90 * time.Second).Format(time.RFC3339)

	_, err := a.db.Exec(`
		INSERT INTO approvals (id, ts, expires_at, action_id, method, nonce, status)
		VALUES (?, ?, ?, ?, 'pin', ?, 'waiting')
	`, approvalID, now.Format(time.RFC3339), expires, actionID, uuid.New().String()[:8])

	if err != nil {
		return ApprovalChallenge{}, err
	}

	// Get action details for the message
	var actionType, targetStr string
	err = a.db.QueryRow(`SELECT action_type, target FROM actions WHERE id=?`, actionID).
		Scan(&actionType, &targetStr)
	if err != nil {
		return ApprovalChallenge{}, err
	}

	msg := fmt.Sprintf("Action %s requested on %s. Reply with: PIN #### (expires 90s)",
		actionType, targetStr)

	return ApprovalChallenge{
		ActionID:   actionID,
		ApprovalID: approvalID,
		ExpiresAt:  expires,
		Message:    msg,
	}, nil
}

// RequestApproval creates a pending action and returns approval challenge
func (a *ActionEngine) RequestApproval(req ActionRequest) (ApprovalChallenge, error) {
	if req.RequestedBy == "" {
		return ApprovalChallenge{}, errors.New("requested_by is required")
	}

	validActions := map[string]bool{
		"quarantine_device":    true,
		"unquarantine_device":  true,
		"block_ip":             true,
		"block_domain":         true,
		"unblock_ip":           true,
		"unblock_domain":       true,
	}

	if !validActions[req.ActionType] {
		return ApprovalChallenge{}, fmt.Errorf("invalid action_type: %s", req.ActionType)
	}

	actionID, err := a.createPendingAction(req)
	if err != nil {
		return ApprovalChallenge{}, err
	}

	challenge, err := a.createApproval(actionID)
	if err != nil {
		return ApprovalChallenge{}, err
	}

	// Broadcast to UI
	a.hub.Broadcast("action_pending", map[string]interface{}{
		"action_id":     actionID,
		"action_type":   req.ActionType,
		"requested_by":  req.RequestedBy,
		"approval_id":   challenge.ApprovalID,
		"expires_at":    challenge.ExpiresAt,
	})

	return challenge, nil
}

// Approve approves and executes a pending action
func (a *ActionEngine) Approve(approvalID, pin, approvedBy string) (ActionResult, error) {
	if !a.auth.ValidateActionPIN(pin) {
		return ActionResult{}, errors.New("invalid PIN")
	}

	var actionID, expiresAt, status string
	err := a.db.QueryRow(
		`SELECT action_id, expires_at, status FROM approvals WHERE id=?`,
		approvalID).Scan(&actionID, &expiresAt, &status)

	if err == sql.ErrNoRows {
		return ActionResult{}, errors.New("approval not found")
	}
	if err != nil {
		return ActionResult{}, err
	}

	if status != "waiting" {
		return ActionResult{}, fmt.Errorf("approval already %s", status)
	}

	exp, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return ActionResult{}, err
	}

	if time.Now().UTC().After(exp) {
		_, _ = a.db.Exec(`UPDATE approvals SET status='expired' WHERE id=?`, approvalID)
		return ActionResult{}, errors.New("approval expired")
	}

	// Mark approval as used
	_, _ = a.db.Exec(`UPDATE approvals SET status='approved' WHERE id=?`, approvalID)

	// Execute the action
	result, err := a.executeAction(actionID, approvedBy)
	if err != nil {
		return ActionResult{}, err
	}

	// Broadcast execution
	a.hub.Broadcast("action_executed", map[string]interface{}{
		"action_id":    actionID,
		"status":       result.Status,
		"approved_by":  approvedBy,
	})

	return result, nil
}

// executeAction performs the actual security action
func (a *ActionEngine) executeAction(actionID, approvedBy string) (ActionResult, error) {
	var actionType, targetStr string
	err := a.db.QueryRow(`SELECT action_type, target FROM actions WHERE id=?`,
		actionID).Scan(&actionType, &targetStr)
	if err != nil {
		return ActionResult{}, err
	}

	// Parse target JSON
	var target map[string]interface{}
	if err := json.Unmarshal([]byte(targetStr), &target); err != nil {
		return ActionResult{}, fmt.Errorf("invalid target JSON: %w", err)
	}

	execTS := time.Now().UTC().Format(time.RFC3339)
	var detail string
	var execErr error

	// Execute action based on type
	switch actionType {
	case "block_ip":
		detail, execErr = a.executeBlockIP(target)
	case "unblock_ip":
		detail, execErr = a.executeUnblockIP(target)
	case "quarantine_device":
		detail, execErr = a.executeQuarantineDevice(target)
	case "unquarantine_device":
		detail, execErr = a.executeUnquarantineDevice(target)
	case "block_domain":
		// Domain blocking requires DNS sinkhole - log for now
		detail = fmt.Sprintf("Domain %v blocking logged (requires DNS sinkhole setup)", target)
	case "unblock_domain":
		detail = fmt.Sprintf("Domain %v unblock logged (requires DNS sinkhole setup)", target)
	default:
		detail = fmt.Sprintf("Unknown action type: %s", actionType)
	}

	if execErr != nil {
		// Update action status as failed
		_ = a.logActionFailure(actionID, execTS, execErr.Error())
		return ActionResult{}, execErr
	}

	// Update action status
	_, err = a.db.Exec(`
		UPDATE actions
		SET status='executed', approved_by=?, executed_ts=?, audit=?
		WHERE id=?
	`, approvedBy, execTS, fmt.Sprintf(`{"detail": "%s"}`, detail), actionID)

	if err != nil {
		return ActionResult{}, err
	}

	return ActionResult{
		ActionID: actionID,
		Status:   "approved_executed",
		Detail:   detail,
	}, nil
}

// executeBlockIP blocks an IP using Windows Firewall or iptables
func (a *ActionEngine) executeBlockIP(target map[string]interface{}) (string, error) {
	ip, ok := target["ip"].(string)
	if !ok || ip == "" {
		return "", fmt.Errorf("invalid IP address in target")
	}

	// Get execution mode from environment
	execMode := env("BG_EXEC_MODE", "log")

	switch execMode {
	case "production":
		// Execute actual block command
		err := executeFirewallBlock(ip)
		if err != nil {
			return "", fmt.Errorf("failed to block IP %s: %w", ip, err)
		}
		return fmt.Sprintf("IP %s blocked via firewall", ip), nil
	case "demo":
		return fmt.Sprintf("IP %s would be blocked (demo mode)", ip), nil
	default:
		// log mode - just log the action
		return fmt.Sprintf("IP %s block logged (log mode - no action taken)", ip), nil
	}
}

// executeUnblockIP unblocks an IP
func (a *ActionEngine) executeUnblockIP(target map[string]interface{}) (string, error) {
	ip, ok := target["ip"].(string)
	if !ok || ip == "" {
		return "", fmt.Errorf("invalid IP address in target")
	}

	execMode := env("BG_EXEC_MODE", "log")

	switch execMode {
	case "production":
		err := executeFirewallUnblock(ip)
		if err != nil {
			return "", fmt.Errorf("failed to unblock IP %s: %w", ip, err)
		}
		return fmt.Sprintf("IP %s unblocked via firewall", ip), nil
	case "demo":
		return fmt.Sprintf("IP %s would be unblocked (demo mode)", ip), nil
	default:
		return fmt.Sprintf("IP %s unblock logged (log mode - no action taken)", ip), nil
	}
}

// executeQuarantineDevice quarantines a device (blocks all traffic from it)
func (a *ActionEngine) executeQuarantineDevice(target map[string]interface{}) (string, error) {
	device, ok := target["device"].(string)
	if !ok || device == "" {
		return "", fmt.Errorf("invalid device in target")
	}

	execMode := env("BG_EXEC_MODE", "log")

	switch execMode {
	case "production":
		err := executeFirewallBlock(device)
		if err != nil {
			return "", fmt.Errorf("failed to quarantine device %s: %w", device, err)
		}
		return fmt.Sprintf("Device %s quarantined (all traffic blocked)", device), nil
	case "demo":
		return fmt.Sprintf("Device %s would be quarantined (demo mode)", device), nil
	default:
		return fmt.Sprintf("Device %s quarantine logged (log mode - no action taken)", device), nil
	}
}

// executeUnquarantineDevice removes device quarantine
func (a *ActionEngine) executeUnquarantineDevice(target map[string]interface{}) (string, error) {
	device, ok := target["device"].(string)
	if !ok || device == "" {
		return "", fmt.Errorf("invalid device in target")
	}

	execMode := env("BG_EXEC_MODE", "log")

	switch execMode {
	case "production":
		err := executeFirewallUnblock(device)
		if err != nil {
			return "", fmt.Errorf("failed to unquarantine device %s: %w", device, err)
		}
		return fmt.Sprintf("Device %s unquarantined (traffic restored)", device), nil
	case "demo":
		return fmt.Sprintf("Device %s would be unquarantined (demo mode)", device), nil
	default:
		return fmt.Sprintf("Device %s unquarantine logged (log mode - no action taken)", device), nil
	}
}

// logActionFailure logs a failed action execution
func (a *ActionEngine) logActionFailure(actionID, execTS, errMsg string) error {
	_, err := a.db.Exec(`
		UPDATE actions
		SET status='failed', executed_ts=?, audit=?
		WHERE id=?
	`, execTS, fmt.Sprintf(`{"error": "%s"}`, errMsg), actionID)
	return err
}

// GetPendingActions returns all pending actions
func (a *ActionEngine) GetPendingActions() ([]map[string]interface{}, error) {
	rows, err := a.db.Query(`
		SELECT id, ts, action_type, target, ttl_sec, requested_by
		FROM actions
		WHERE status='pending'
		ORDER BY ts DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []map[string]interface{}
	for rows.Next() {
		var id, ts, actionType, target, requestedBy string
		var ttlSec int

		err := rows.Scan(&id, &ts, &actionType, &target, &ttlSec, &requestedBy)
		if err != nil {
			return nil, err
		}

		actions = append(actions, map[string]interface{}{
			"id":           id,
			"ts":           ts,
			"action_type":  actionType,
			"target":       target,
			"ttl_sec":      ttlSec,
			"requested_by": requestedBy,
		})
	}

	return actions, nil
}
