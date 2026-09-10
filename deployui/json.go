package deployui

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fatima-go/fatima-core/opm/api"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func RunJSON(c *Client, o Options) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	var result proto.Message
	var err error
	switch o.Command {
	case "upload":
		if o.RequestID == "" {
			return fmt.Errorf("--request-id is required for an upload in JSON mode")
		}
		result, err = c.Upload(ctx, o.Value, o.RequestID, nil)
	case "", "artifacts":
		result, err = c.Artifacts(ctx)
	case "rollouts":
		result, err = c.Rollouts(ctx)
	case "watch":
		result, err = c.Get(ctx, o.Value)
	case "create":
		if o.RequestID == "" {
			return fmt.Errorf("--request-id is required for create in JSON mode")
		}
		result, err = c.Create(ctx, &api.CreateRollout{RequestId: o.RequestID, ArtifactId: o.ArtifactID, Group: o.Group, FirstPackageId: o.First})
	case "act":
		result, err = c.Act(ctx, &api.RolloutAction{Id: o.Value, Action: o.Action, ExpectedRevision: o.Revision})
	default:
		return fmt.Errorf("unknown command")
	}
	if err != nil {
		return err
	}
	b, err := protojson.MarshalOptions{Indent: "  ", UseProtoNames: true}.Marshal(result)
	if err == nil {
		_, err = fmt.Fprintln(os.Stdout, string(b))
	}
	return err
}
