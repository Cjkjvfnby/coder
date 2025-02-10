package agentapi

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/google/uuid"
	"golang.org/x/xerrors"
	"google.golang.org/protobuf/types/known/emptypb"

	"cdr.dev/slog"

	agentproto "github.com/coder/coder/v2/agent/proto"
	"github.com/coder/coder/v2/coderd/audit"
	"github.com/coder/coder/v2/coderd/database"
)

type AuditAPI struct {
	AgentFn  func(context.Context) (database.WorkspaceAgent, error)
	Log      slog.Logger
	Database database.Store
	Auditor  audit.Auditor
}

func (a *AuditAPI) ReportConnection(ctx context.Context, req *agentproto.ReportConnectionRequest) (*emptypb.Empty, error) {
	connectionID, err := uuid.FromBytes(req.Connection.Id)
	if err != nil {
		return nil, xerrors.Errorf("connection id from bytes: %w", err)
	}

	workspaceAgent, err := a.AgentFn(ctx)
	if err != nil {
		return nil, err
	}
	workspace, err := a.Database.GetWorkspaceByAgentID(ctx, workspaceAgent.ID)
	if err != nil {
		return nil, err
	}
	build, err := a.Database.GetLatestWorkspaceBuildByWorkspaceID(ctx, workspace.ID)
	if err != nil {
		return nil, err
	}

	// TODO(mafredri): Probably best to use enum values here and translate to human in UI.
	var connectionType string
	switch req.Connection.Type {
	case agentproto.Connection_SSH:
		connectionType = "SSH"
	case agentproto.Connection_VSCODE:
		connectionType = "VS Code Remote"
	case agentproto.Connection_JETBRAINS:
		connectionType = "JetBrains"
	case agentproto.Connection_RECONNECTING_PTY:
		connectionType = "Web Terminal"
	default:
		connectionType = "Unknown"
	}

	// We pass the below information to the Auditor so that it
	// can form a friendly string for the user to view in the UI.
	type additionalFields struct {
		audit.AdditionalFields

		AgentID        uuid.UUID `json:"agent_id"`
		ConnectionType string    `json:"connection_type"`
		ConnectedAt    time.Time `json:"connected_at"`
	}
	resourceInfo := additionalFields{
		AdditionalFields: audit.AdditionalFields{
			WorkspaceName: workspace.Name,
			BuildNumber:   strconv.FormatInt(int64(build.BuildNumber), 10),
			BuildReason:   database.BuildReason(string(build.Reason)),
			WorkspaceID:   workspace.ID,
		},
		AgentID:        workspaceAgent.ID,
		ConnectionType: connectionType,
		ConnectedAt:    req.Connection.Start.AsTime(),
	}

	riBytes, err := json.Marshal(resourceInfo)
	if err != nil {
		a.Log.Error(ctx, "marshal workspace resource info for failed job", slog.Error(err))
		riBytes = []byte("{}")
	}

	// bag := audit.BaggageFromContext(ctx)

	audit.BackgroundAudit(ctx, &audit.BackgroundAuditParams[database.WorkspaceAgent]{
		Audit:            a.Auditor,
		Log:              a.Log,
		UserID:           workspace.OwnerID, // TODO(mafredri): If the agent knows it, we should send the ID.
		OrganizationID:   workspace.OrganizationID,
		RequestID:        connectionID, // TODO(mafredri): Should we use connection ID here?
		Action:           database.AuditActionConnect,
		AdditionalFields: riBytes,

		// IP:     bag.IP,
		// Old:    previousBuild,
		// New:    build,
		// Status: http.StatusOK,
	})

	return &emptypb.Empty{}, nil
}
