package agentapi_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"

	agentproto "github.com/coder/coder/v2/agent/proto"
	"github.com/coder/coder/v2/coderd/agentapi"
	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbmock"
)

func TestAuditReport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id    uuid.UUID
		typ   *agentproto.Connection_Type
		start time.Time
		end   time.Time
	}{
		{
			id:    uuid.New(),
			typ:   agentproto.Connection_SSH.Enum(),
			start: time.Now(),
		},
	}
	for _, tt := range tests {
		mDB := dbmock.NewMockStore(gomock.NewController(t))
		mDB.EXPECT().InsertAuditLog(gomock.Any(), database.InsertAuditLogParams{
			// TODO(mafredri): Fill in the expected values.
		})

		api := &agentapi.AuditAPI{Database: mDB}
		api.ReportConnection(context.Background(), &agentproto.ReportConnectionRequest{
			Connection: &agentproto.Connection{
				Id:    tt.id[:],
				Type:  *tt.typ,
				Start: timestamppb.New(tt.start),
				End:   timestamppb.New(tt.end),
			},
		})
	}
}
